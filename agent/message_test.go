package agent

import (
	"os"
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestPostMessages(t *testing.T) {
	channel := os.Getenv("SLACK_CHANNEL")
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if channel == "" || botToken == "" {
		t.Skip("Skip testing messages")
	}

	testCases := []struct {
		title   string
		message slack.MsgOption
	}{
		{
			title:   "CIResult",
			message: messageCIResult(colorGreen, "Message", "Job", "my-namespace/my-pod", false),
		},
		{
			title:   "CIResult (Extend Button)",
			message: messageCIResult(colorRed, "Message", "Job", "my-namespace/my-pod", true),
		},
		{
			title:   "PodExtendSuccess",
			message: messagePodExtendSuccess("my-namespace/my-pod", time.Now()),
		},
		{
			title:   "PodExtendFailure",
			message: messagePodExtendFailure("my-namespace/my-pod"),
		},
	}

	apiClient := slack.New(botToken)
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			// Post messages actually and make sure that no error occurs.
			_, _, err := apiClient.PostMessage(channel, tc.message)
			if err != nil {
				t.Error(tc.title, "| got error", err)
			}
		})
	}
}
