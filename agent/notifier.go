package agent

import (
	"context"
	"net/http"
	"time"

	"github.com/cybozu-go/well"
	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
)

// Notifier receives requests from Pods and send message to Slack.
type Notifier struct {
	listenAddr  string
	webhookURL  string
	postWebhook func(string, *slack.WebhookMessage) error
}

// NewNotifier creates Notifier.
func NewNotifier(
	listenAddr string,
	webhookURL string,
	postWebhook func(string, *slack.WebhookMessage) error,
) *Notifier {
	return &Notifier{
		listenAddr,
		webhookURL,
		postWebhook,
	}
}

// Start start HTTP server.
func (s *Notifier) Start(_ context.Context) error {
	serv := &well.HTTPServer{
		Server: &http.Server{
			Addr:    s.listenAddr,
			Handler: s.prepareRouter(),
		},
	}
	return serv.ListenAndServe()
}

func (s *Notifier) prepareRouter() http.Handler {
	router := gin.Default()
	router.POST("/slack/success", s.postSlackSuccess)
	router.POST("/slack/fail", s.postSlackFail)
	return router
}

func (s *Notifier) postSlack(c *gin.Context, isSucceeded bool) {
	var p struct {
		JobName      string `json:"job_name"`
		PodNamespace string `json:"pod_namespace"`
		PodName      string `json:"pod_name"`
	}

	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.postWebhook(
		s.webhookURL,
		makeJobResultMsg(
			p.JobName,
			p.PodNamespace,
			p.PodName,
			isSucceeded,
			time.Now(),
		),
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Notifier) postSlackSuccess(c *gin.Context) {
	s.postSlack(c, true)
}

func (s *Notifier) postSlackFail(c *gin.Context) {
	s.postSlack(c, false)
}
