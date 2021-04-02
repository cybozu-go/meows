package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SocketModeClient is a client for Slack socket mode.
type SocketModeClient struct {
	log      logr.Logger
	client   *socketmode.Client
	annotate func(context.Context, string, string, time.Time) error
}

// NewSocketModeClient creates SocketModeClient.
func NewSocketModeClient(
	logger logr.Logger,
	appToken string,
	botToken string,
	annotate func(context.Context, string, string, time.Time) error,
) *SocketModeClient {
	return &SocketModeClient{
		log: logger,
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
func (s *SocketModeClient) ListenInteractiveEvents(ctx context.Context) error {
	for envelope := range s.client.Events {
		if envelope.Type != socketmode.EventTypeInteractive {
			s.log.Info("skipped event because type is not "+string(socketmode.EventTypeInteractive),
				"type", envelope.Type,
				"data", envelope.Data,
			)
			continue
		}
		cb, ok := envelope.Data.(slack.InteractionCallback)
		if !ok {
			err := fmt.Errorf(
				"received data cannot be converted into slack.InteractionCallback: %#v",
				envelope.Data,
			)
			s.log.Error(err, "failed to convert type to "+string(socketmode.EventTypeInteractive),
				"data", envelope.Data,
			)
			return err
		}
		if cb.Type != slack.InteractionTypeBlockActions {
			s.log.Info("skipped event because data type is not "+string(slack.InteractionTypeBlockActions),
				"type", cb.Type,
			)
			continue
		}
		if envelope.Request == nil {
			err := fmt.Errorf("request should not be nil: %#v", envelope.Data)
			s.log.Error(err, "request should not be nil")
			return err
		}

		p, err := newPostResultPayloadFromCB(&cb)
		if err != nil {
			s.log.Error(err, "failed to get result from callback", "cb", cb)
			return err
		}

		// TODO: time.Now() is replaced after timepicker gets available.
		err = s.annotate(ctx, p.PodName, p.PodNamespace, time.Now().Add(30*time.Minute))
		if err != nil {
			s.log.Error(err, "failed to annotate deletion time",
				"name", p.PodName,
				"namespace", p.PodNamespace,
			)
			return err
		}

		s.client.Ack(*envelope.Request, p.makeExtendCallbackPayload())
	}
	return nil
}
