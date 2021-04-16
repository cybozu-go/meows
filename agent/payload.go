package agent

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

const (
	pickerActionID    = "slack-agent-extend"
	pickerActionValue = "slack-agent-extend"

	podNameTitle      = "Pod"
	podNamespaceTitle = "Namespace"
	branchTitle       = "Branch"

	runLinkBase = "https://github.com/"
	runLinkFmt  = "https://github.com/%s/%s/actions/runs/%d"
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

func newPostResultPayloadFromCB(body *slack.InteractionCallback) (*postResultPayload, error) {
	var isFailed bool
	switch len(body.Message.Attachments) {
	case 1:
		isFailed = false
	case 2:
		isFailed = true
	default:
		return nil, fmt.Errorf(
			"length of attachments should be 1 or 2, but got %d: %#v",
			len(body.Message.Attachments),
			body.Message.Attachments,
		)
	}

	a := body.Message.Attachments[0]
	m := make(map[string]string)
	for _, v := range a.Fields {
		m[v.Title] = v.Value
	}

	podName, ok := m[podNameTitle]
	if !ok {
		return nil, fmt.Errorf(`the field "%s" should not be empty`, podNameTitle)
	}

	podNamespace, ok := m[podNamespaceTitle]
	if !ok {
		return nil, fmt.Errorf(`the field "%s" should not be empty`, podNamespaceTitle)
	}

	branchName, ok := m[branchTitle]
	if !ok {
		return nil, fmt.Errorf(`the field "%s" should not be empty`, branchTitle)
	}

	s := strings.Split(strings.TrimPrefix(a.TitleLink, runLinkBase), "/")
	if len(s) != 5 {
		return nil, fmt.Errorf("the title link should have %s fmt", runLinkFmt)
	}
	organizationName := s[0]
	repositoryName := s[1]
	runID, err := strconv.ParseUint(s[4], 10, 64)
	if err != nil {
		return nil, err
	}
	return newPostResultPayload(
		repositoryName,
		organizationName,
		a.Title,
		branchName,
		uint(runID),
		podNamespace,
		podName,
		isFailed,
	), nil
}

func (p *postResultPayload) makeExtendResultMsgOption(t time.Time) slack.MsgOption {
	return slack.MsgOptionText(
		fmt.Sprintf(
			"%s in %s is extended successfully by %s",
			p.PodName,
			p.PodNamespace,
			t.Format(time.RFC3339),
		),
		false,
	)
}

func (p *postResultPayload) makeCIResultWebhookMsg() *slack.WebhookMessage {
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
					runLinkFmt,
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
