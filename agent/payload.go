package agent

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

const (
	pickerActionID    = "slack-agent-extend"
	podNameTitle      = "Pod"
	podNamespaceTitle = "Namespace"
)

type postResultPayload struct {
	RepositoryName string `json:"repository_name"`
	WorkflowName   string `json:"workflow_name"`
	BranchName     string `json:"branch_name"`
	RunID          uint   `json:"run_id"`
	PodNamespace   string `json:"pod_namespace"`
	PodName        string `json:"pod_name"`
	IsFailed       bool   `json:"is_failed"`
}

func newPostResultPayload(
	repositoryName string,
	workflowName string,
	branchName string,
	runID uint,
	podNamespace string,
	podName string,
	isFailed bool,
) *postResultPayload {
	return &postResultPayload{
		RepositoryName: repositoryName,
		WorkflowName:   workflowName,
		BranchName:     branchName,
		RunID:          runID,
		PodNamespace:   podNamespace,
		PodName:        podName,
		IsFailed:       isFailed,
	}
}

func (p *postResultPayload) makeWebhookMessage() *slack.WebhookMessage {
	text := p.WorkflowName + " workflow has failed"
	color := "danger"
	if !p.IsFailed {
		text = p.WorkflowName + " workflow has succeeded"
		color = "good"
	}

	msg := &slack.WebhookMessage{
		Text: text,
		Attachments: []slack.Attachment{
			{
				Color: color,
				Title: "Title",
				TitleLink: fmt.Sprintf(
					"https://github.com/%s/actions/runs/%d",
					p.RepositoryName,
					p.RunID,
				),
				Blocks: slack.Blocks{
					BlockSet: []slack.Block{
						slack.NewSectionBlock(
							slack.NewTextBlockObject(
								slack.MarkdownType,
								"Choose time to extend Pod by",
								false,
								false,
							),
							nil,
							slack.NewAccessory(
								&slack.TimePickerBlockElement{
									Type:        slack.METTimepicker,
									ActionID:    pickerActionID,
									InitialTime: time.Now().Add(time.Hour).UTC().Format("03:04"),
								},
							),
						),
					},
				},
				// Fields: []slack.AttachmentField{
				// 	{Title: "Branch", Value: p.BranchName, Short: false},
				// 	{Title: podNamespaceTitle, Value: p.PodNamespace, Short: true},
				// 	{Title: podNameTitle, Value: p.PodName, Short: true},
				// },
				// Ts: json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
			},
		},
	}
	if !p.IsFailed {
		return msg
	}

	return msg
}
