package agent

import (
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/slack-go/slack"
	"k8s.io/apimachinery/pkg/types"
)

func makeJobResultMsg(
	jobName string,
	podNamespace string,
	podName string,
	isFailed bool,
	timestamp time.Time,
) *slack.WebhookMessage {
	text := "CI Failed"
	color := "danger"
	if !isFailed {
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
					{Title: constants.MsgJobNameTitle, Value: jobName, Short: true},
					{Title: constants.MsgTimestampTitle, Value: timestamp.Format(time.RFC3339), Short: true},
					{Title: constants.MsgPodNamespaceTitle, Value: podNamespace, Short: true},
					{Title: constants.MsgPodNameTitle, Value: podName, Short: true},
				},
			},
		},
	}

	if !isFailed {
		return msg
	}

	msg.Blocks = &slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, "Do you extend the lifetime of the instance?", false, false),
				nil,
				slack.NewAccessory(
					slack.NewButtonBlockElement(
						constants.SlackButtonActionID,
						podName,
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
		case constants.MsgPodNameTitle:
			name = v.Value
		case constants.MsgPodNamespaceTitle:
			namespace = v.Value
		}
	}

	if len(name) == 0 {
		return nil, fmt.Errorf(`the field "%s" should not be empty`, constants.MsgPodNameTitle)
	}
	if len(namespace) == 0 {
		return nil, fmt.Errorf(`the field "%s" should not be empty`, constants.MsgPodNamespaceTitle)
	}

	return &types.NamespacedName{Name: name, Namespace: namespace}, nil
}
