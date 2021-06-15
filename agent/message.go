package agent

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

const (
	extendBlockID  = "slack-agent-extend"
	pickerActionID = "slack-agent-extend-timepicker"
	buttonActionID = "slack-agent-extend-button"
)

func messageCIResult(color, text, job, pod string, extend bool) slack.MsgOption {
	blockSet := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
			[]*slack.TextBlockObject{
				// These test blocks are rendered as 2 columns of side-by-side text.
				// So list field names and values two each.
				slack.NewTextBlockObject(slack.MarkdownType, "Job", false, false),
				slack.NewTextBlockObject(slack.MarkdownType, "Pod", false, false),
				slack.NewTextBlockObject(slack.MarkdownType, job, false, false),
				slack.NewTextBlockObject(slack.MarkdownType, pod, false, false),
			},
			nil,
		),
	}

	if extend {
		extendBlock := slack.NewActionBlock(
			extendBlockID,
			&slack.TimePickerBlockElement{
				Type:        slack.METTimepicker,
				ActionID:    pickerActionID,
				InitialTime: time.Now().Add(30 * time.Minute).UTC().Format("03:04"),
			},
			slack.NewButtonBlockElement(
				buttonActionID,
				pod,
				slack.NewTextBlockObject(slack.PlainTextType, "Extend", false, false),
			),
		)
		blockSet = append(blockSet, extendBlock)
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
	return slack.MsgOptionText(fmt.Sprintf("%s is extended successfully.\n- %s", pod, extendedTime), false)
}

func messagePodExtendFailure(pod string) slack.MsgOption {
	return slack.MsgOptionText(fmt.Sprintf("Failed to extend pod.\n- %s", pod), false)
}
