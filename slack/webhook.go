package slack

import (
	"context"

	"github.com/slack-go/slack"
)

// WebhookMessageNotifier notifies message.
type WebhookMessageNotifier interface {
	Notify(context.Context, *slack.WebhookMessage) error
}

// WebhookClient is a client for Slack Webhook.
type WebhookClient struct {
	WebhookURL string
}

// NewWebhookClient creates Client.
func NewWebhookClient(webhookURL string) *WebhookClient {
	return &WebhookClient{webhookURL}
}

// Notify sends message to Slack WebhookURL.
func (c *WebhookClient) Notify(
	ctx context.Context,
	msg *slack.WebhookMessage,
) error {
	return slack.PostWebhookContext(ctx, c.WebhookURL, msg)
}

// fakeWebhookClient is a fake client.
type fakeWebhookClient struct{}

// newFakeWebhookClient creates fake client for test.
func newFakeWebhookClient() *fakeWebhookClient {
	return &fakeWebhookClient{}
}

// Notify emulates communication with Slack.
func (c *fakeWebhookClient) Notify(
	ctx context.Context,
	msg *slack.WebhookMessage,
) error {
	return nil
}
