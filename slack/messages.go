package slack

import (
	"errors"
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"k8s.io/apimachinery/pkg/types"
)

const (
	MsgJobNameTitle      = "Job"
	MsgTimestampTitle    = "Timestamp"
	MsgPodNameTitle      = "Pod"
	MsgPodNamespaceTitle = "Namespace"
)

func makeJobResultMsg(
	jobName string,
	podNamespace string,
	podName string,
	isSucceeded bool,
	timestamp time.Time,
) *slack.WebhookMessage {
	text := "CI Failed"
	color := "danger"
	if isSucceeded {
		text = "CI Succeeded"
		color = "good"
	}

	msg := &slack.WebhookMessage{
		Attachments: []slack.Attachment{
			{
				AuthorName: "Self-hosted Runner",
				Color:      color,
				Title:      "GitHub Actions",
				Text:       text,
				Fields: []slack.AttachmentField{
					{Title: MsgJobNameTitle, Value: jobName, Short: true},
					{Title: MsgTimestampTitle, Value: timestamp.Format(time.RFC3339), Short: true},
					{Title: MsgPodNamespaceTitle, Value: podNamespace, Short: true},
					{Title: MsgPodNameTitle, Value: podName, Short: true},
				},
			},
		},
	}

	if isSucceeded {
		return msg
	}

	msg.Blocks = &slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Failed: xxx", false, false),
				nil,
				slack.NewAccessory(
					slack.NewButtonBlockElement(
						"id",
						"instance_name",
						slack.NewTextBlockObject(
							slack.PlainTextType,
							"Extend",
							true,
							false,
						),
					),
				),
			),
		},
	}
	return msg
}

func extractNameFromJobResultMsg(body *slack.InteractionCallback) (*types.NamespacedName, error) {
	if len(body.Message.Attachments) != 1 {
		return nil, fmt.Errorf(
			"length of attachments should be 1, but got %d: %#v",
			len(body.Message.Attachments),
			body.Message.Attachments,
		)
	}

	var name, namespace string
	a := body.Message.Attachments[0]
	for _, v := range a.Fields {
		switch v.Title {
		case MsgPodNameTitle:
		case MsgPodNamespaceTitle:
		}
	}

	if len(name) == 0 {
		return nil, errors.New(MsgPodNameTitle + " should not be empty")
	}
	if len(namespace) == 0 {
		return nil, errors.New(MsgPodNamespaceTitle + " should not be empty")
	}

	return &types.NamespacedName{Name: name, Namespace: namespace}, nil
}
