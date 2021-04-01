package agent

import (
	"fmt"

	"github.com/slack-go/slack"
)

const (
	pickerActionID    = "slack-agent-extend"
	pickerActionValue = "slack-agent-extend"
	podNameTitle      = "Pod"
	podNamespaceTitle = "Namespace"
)

type postResultPayload struct {
	RepositoryName   string `json:"repository_name"`
	OrganizationName string `json:"organization_name"`
	WorkflowName     string `json:"workflow_name"`
	BranchName       string `json:"branch_name"`
	RunID            uint   `json:"run_id"`
	PodNamespace     string `json:"pod_namespace"`
	PodName          string `json:"pod_name"`
	IsFailed         bool   `json:"is_failed"`
}

func newPostResultPayload(
	repositoryName string,
	organizationName string,
	workflowName string,
	branchName string,
	runID uint,
	podNamespace string,
	podName string,
	isFailed bool,
) *postResultPayload {
	return &postResultPayload{
		RepositoryName:   repositoryName,
		OrganizationName: organizationName,
		WorkflowName:     workflowName,
		BranchName:       branchName,
		RunID:            runID,
		PodNamespace:     podNamespace,
		PodName:          podName,
		IsFailed:         isFailed,
	}
}

func (p *postResultPayload) makeWebhookMessage() *slack.WebhookMessage {
	text := fmt.Sprintf("CI on %s/%s has succeded", p.OrganizationName, p.RepositoryName)
	color := "#36a64f"
	if p.IsFailed {
		text = fmt.Sprintf("CI on %s/%s has failed", p.OrganizationName, p.RepositoryName)
		color = "#d10c20"
	}

	msg := &slack.WebhookMessage{
		Text: text,
		Attachments: []slack.Attachment{
			{
				Color: color,
				Title: p.WorkflowName,
				TitleLink: fmt.Sprintf(
					"https://github.com/%s/%s/actions/runs/%d",
					p.OrganizationName,
					p.RepositoryName,
					p.RunID,
				),
				Fields: []slack.AttachmentField{
					{Title: "Branch", Value: p.BranchName, Short: false},
					{Title: podNamespaceTitle, Value: p.PodNamespace, Short: true},
					{Title: podNameTitle, Value: p.PodName, Short: true},
				},
			},
		},
	}
	if !p.IsFailed {
		return msg
	}

	msg.Attachments = append(msg.Attachments,
		slack.Attachment{
			Color: color,
			Blocks: slack.Blocks{
				BlockSet: []slack.Block{
					slack.NewActionBlock(
						"timepicker-block",
						// TODO: Uncomment this block after https://github.com/slack-go/slack/pull/918
						// is released and change the way to parse messages in socket.go.
						//
						// &slack.TimePickerBlockElement{
						// 	Type:        slack.METTimepicker,
						// 	ActionID:    pickerActionID + "-ignored",
						// 	InitialTime: time.Now().Add(30 * time.Minute).UTC().Format("03:04"),
						// },
						slack.NewButtonBlockElement(
							pickerActionID,
							pickerActionValue,
							slack.NewTextBlockObject(
								slack.PlainTextType,
								"Extend",
								false,
								false,
							),
						),
					),
				},
			},
		},
	)
	return msg
}
