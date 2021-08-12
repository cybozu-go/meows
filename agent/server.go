package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/meows/runner/client"
	"github.com/cybozu-go/well"
	"github.com/go-logr/logr"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	resultAPIPath    = "/result"
	slackPostTimeout = 3 * time.Second
)

type resultAPIPayload struct {
	Color   string `json:"color"`
	Text    string `json:"text"`
	Job     string `json:"job"`
	Pod     string `json:"pod"`
	Extend  bool   `json:"extend"`
	Channel string `json:"channel"`
}

// Server receives requests from clients and communicates with Slack.
type Server struct {
	log            logr.Logger
	listenAddr     string
	defaultChannel string
	apiClient      *slack.Client
	smClient       *socketmode.Client
	devMood        bool
	clientset      *kubernetes.Clientset
	runnerClient   client.Client
}

// NewServer creates slack agent server.
func NewServer(logger logr.Logger, listenAddr string, defaultChannel string, appToken, botToken string, devMode bool, verbose bool) (*Server, error) {
	apiClient := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
	)
	smClient := socketmode.New(
		apiClient,
		socketmode.OptionDebug(verbose),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	var clientset *kubernetes.Clientset
	if !devMode {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
	}

	return &Server{
		log:            logger,
		listenAddr:     listenAddr,
		defaultChannel: defaultChannel,
		apiClient:      apiClient,
		smClient:       smClient,
		devMood:        devMode,
		clientset:      clientset,
		runnerClient:   client.NewClient(),
	}, nil
}

// Run starts the slack agent server.
func (s *Server) Run(ctx context.Context) error {
	env := well.NewEnvironment(ctx)

	// Run HTTP server.
	serv := &well.HTTPServer{
		Env: env,
		Server: &http.Server{
			Addr:    s.listenAddr,
			Handler: s,
		},
	}
	err := serv.ListenAndServe()
	if err != nil {
		return err
	}

	// Run goroutines for socket mode client.
	env.Go(s.listenInteractiveEvents)
	env.Go(s.runSocket)

	env.Stop()
	return env.Wait()
}

func errorResponse(w http.ResponseWriter, statusCode int, messages ...string) {
	w.WriteHeader(statusCode)
	if len(messages) != 0 {
		w.Write([]byte(strings.Join(messages, ";")))
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != resultAPIPath {
		errorResponse(w, http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		errorResponse(w, http.StatusUnsupportedMediaType)
		return
	}

	payload := new(resultAPIPayload)
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var channel string
	switch {
	case payload.Channel != "":
		channel = payload.Channel
	case s.defaultChannel != "":
		channel = s.defaultChannel
	default:
		err := errors.New("channel is not specified")
		errorResponse(w, http.StatusBadRequest, err.Error())
		s.log.Error(err, "failed to send slack message", "pod", payload.Pod)
		return
	}

	msg := messageCIResult(payload.Color, payload.Text, payload.Job, payload.Pod, payload.Extend)
	ctx, cancel := context.WithTimeout(r.Context(), slackPostTimeout)
	defer cancel()
	_, _, err = s.apiClient.PostMessageContext(ctx, channel, msg)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		s.log.Error(err, "failed to send slack message", "pod", payload.Pod, "channel", channel)
		return
	}
	s.log.Info("success to send slack message", "pod", payload.Pod, "channel", channel)
}
