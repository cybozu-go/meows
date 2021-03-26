package agent

import (
	"fmt"
	"log"
	"os"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// SocketModeClient is a client for Slack socket mode.
type SocketModeClient struct {
	client  *socketmode.Client
	handler func(*slack.InteractionCallback) error
}

// NewSocketModeClient creates SocketModeClient.
func NewSocketModeClient(
	appToken string,
	botToken string,
	handler func(*slack.InteractionCallback) error,
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
		handler: handler,
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
			continue
		}

		payload, ok := envelope.Data.(slack.InteractionCallback)
		if !ok {
			return fmt.Errorf(
				"received data cannot be converted into slack.InteractionCallback: %#v",
				envelope.Data,
			)
		}
		if payload.Type != slack.InteractionTypeBlockActions {
			continue
		}

		if err := s.handler(&payload); err != nil {
			return err
		}

		s.client.Ack(*envelope.Request)
	}
	return nil
}
