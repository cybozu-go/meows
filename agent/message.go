package agent

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

const (
	extendBlockID        = "slack-agent-extend"
	extendButtonActionID = "slack-agent-extend-button"
	deleteBlockID        = "slack-agent-delete"
	deleteButtonActionID = "slack-agent-delete-button"
)

const (
	extendDurationHours  = 2
	extendDurationString = "2 hours" // singular/plural hack
	extendLimitHours     = 6
)

func messageCIResult(color, text, job, pod string, extend bool) slack.MsgOption {
	blockSet := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
			[]*slack.TextBlockObject{
				slack.NewTextBlockObject(slack.MarkdownType, "*Job*\n"+job, false, false),
				slack.NewTextBlockObject(slack.MarkdownType, "*Pod*\n"+pod, false, false),
			},
			nil,
		),
	}

	if extend {
		blockSet = append(blockSet,
			slack.NewActionBlock(
				extendBlockID,
				slack.NewButtonBlockElement(
					extendButtonActionID,
					pod,
					slack.NewTextBlockObject(slack.PlainTextType, "Extend "+extendDurationString, false, false),
				),
			),
			slack.NewActionBlock(
				deleteBlockID,
				slack.NewButtonBlockElement(
					deleteButtonActionID,
					pod,
					slack.NewTextBlockObject(slack.PlainTextType, "Delete immediately", false, false),
				),
			),
		)
	}

	// We want to use the color bar. So use the attachment.
	// ref: https://api.slack.com/messaging/attachments-to-blocks#switching_to_blocks
	// > There is one exception, and that's the color parameter, which currently does not have a block alternative.
	// > If you are strongly attached to the color bar, use the blocks parameter within an attachment.
	return slack.MsgOptionAttachments(
		slack.Attachment{
			Color: color,
			Blocks: slack.Blocks{
				BlockSet: blockSet,
			},
		},
	)
}

func messagePodExtendSuccess(pod string, extendedTime time.Time) slack.MsgOption {
	return slack.MsgOptionText(fmt.Sprintf("%s is updated successfully.\n- %s", pod, extendedTime), false)
}

func messagePodExtendFailure(pod string) slack.MsgOption {
	return slack.MsgOptionText(fmt.Sprintf("Failed to update pod.\n- %s", pod), false)
}
