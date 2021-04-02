package agent

import (
	"fmt"
	"log"
	"os"
	"time"

	clog "github.com/cybozu-go/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SocketModeClient is a client for Slack socket mode.
type SocketModeClient struct {
	client   *socketmode.Client
	annotate func(string, string, time.Time) error
}

// NewSocketModeClient creates SocketModeClient.
func NewSocketModeClient(
	appToken string,
	botToken string,
	annotate func(string, string, time.Time) error,
) *SocketModeClient {
	return &SocketModeClient{
		client: socketmode.New(
			slack.New(
				botToken,
				slack.OptionAppLevelToken(appToken),
				slack.OptionDebug(true),
				slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
			),
			socketmode.OptionDebug(true),
			socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
		),
		annotate: annotate,
	}
}

// Run makes a connectionh with Slack over WebSocket.
func (s *SocketModeClient) Run() error {
	return s.client.Run()
}

// ListenInteractiveEvents listens to events from interactive components and
// runs the event handler.
func (s *SocketModeClient) ListenInteractiveEvents() error {
	for envelope := range s.client.Events {
		if envelope.Type != socketmode.EventTypeInteractive {
			clog.Info("skipped event because type is not "+string(socketmode.EventTypeInteractive), map[string]interface{}{
				"type": envelope.Type,
				"data": envelope.Data,
			})
			continue
		}
		cb, ok := envelope.Data.(slack.InteractionCallback)
		if !ok {
			clog.Error("failed to convert type to "+string(socketmode.EventTypeInteractive), map[string]interface{}{
				"data": envelope.Data,
			})
			return fmt.Errorf(
				"received data cannot be converted into slack.InteractionCallback: %#v",
				envelope.Data,
			)
		}
		if cb.Type != slack.InteractionTypeBlockActions {
			clog.Info("skipped event because data type is not "+string(slack.InteractionTypeBlockActions), map[string]interface{}{
				"type": cb.Type,
			})
			continue
		}
		if envelope.Request == nil {
			clog.Error("request should not be nil", map[string]interface{}{})
			return fmt.Errorf("request should not be nil: %#v", envelope.Data)
		}

		p, err := newPostResultPayloadFromCB(&cb)
		if err != nil {
			clog.Error("failed to get result from callback", map[string]interface{}{
				clog.FnError: err,
				"cb":         cb,
			})
			return err
		}

		// TODO: time.Now() is replaced after timepicker gets available.
		err = s.annotate(p.PodName, p.PodNamespace, time.Now().Add(30*time.Minute))
		if err != nil {
			clog.Error("failed to annotate deletion time", map[string]interface{}{
				clog.FnError: err,
				"name":       p.PodName,
				"namespace":  p.PodNamespace,
			})
			return err
		}

		s.client.Ack(*envelope.Request, p.makeExtendCallbackPayload())
	}
	return nil
}
