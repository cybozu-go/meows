package agent

import (
	"net/http"

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
func (s *Notifier) Start() error {
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
	router.POST(postResultPath, s.postResult)
	return router
}

func (s *Notifier) postResult(c *gin.Context) {
	p := new(postResultPayload)
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.postWebhook(s.webhookURL, p.makeCIResultWebhookMsg()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
