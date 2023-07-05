package agent

import (
	"encoding/json"
	"testing"

	"github.com/slack-go/slack"
)

func TestEventType(t *testing.T) {
	testCases := []struct {
		title       string
		extendEvent bool
		deleteEvent bool
		input       string
	}{
		{
			title:       "valid extend",
			extendEvent: true,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-extend-button"
		}
	]
}`,
		},
		{
			title:       "valid delete",
			extendEvent: false,
			deleteEvent: true,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-delete",
			"action_id": "slack-agent-delete-button"
		}
	]
}`,
		},
		{
			title:       "unexpected type 1",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "unexpected_type",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-extend-button"
		}
	]
}`,
		},
		{
			title:       "unexpected type 2",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "unexpected_type",
	"actions": [
		{
			"block_id": "slack-agent-delete",
			"action_id": "slack-agent-delete-button"
		}
	]
}`,
		},
		{
			title:       "unexpected block_id 1",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "unexpected",
			"action_id": "slack-agent-extend-button"
		}
	]
}`,
		},
		{
			title:       "unexpected block_id 2",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "unexpected",
			"action_id": "slack-agent-delete-button"
		}
	]
}`,
		},
		{
			title:       "unexpected action_id 1",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "unexpected"
		}
	]
}`,
		},
		{
			title:       "unexpected action_id 2",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-delete",
			"action_id": "unexpected"
		}
	]
}`,
		},
		{
			title:       "mixed id 1",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-delete-button"
		}
	]
}`,
		},
		{
			title:       "mixed id 2",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-delete",
			"action_id": "slack-agent-extend-button"
		}
	]
}`,
		},
		{
			title:       "no action",
			extendEvent: false,
			deleteEvent: false,
			input: `
{
	"type": "block_actions",
	"actions": []
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cb := new(slack.InteractionCallback)
			if err := json.Unmarshal([]byte(tc.input), cb); err != nil {
				t.Fatal(tc.title, err)
			}

			actual := isExtendButtonEvent(cb)
			if tc.extendEvent != actual {
				t.Error(tc.title, "| extend: expected:", tc.extendEvent, " actual:", actual)
			}

			actual = isDeleteButtonEvent(cb)
			if tc.deleteEvent != actual {
				t.Error(tc.title, "| delete: expected:", tc.deleteEvent, " actual:", actual)
			}
		})
	}
}

func TestGetPodFromCallbackEvent(t *testing.T) {
	testCases := []struct {
		title             string
		errorCase         bool
		expectedNamespace string
		expectedPod       string
		input             string
	}{
		{
			title:             "valid",
			expectedNamespace: "my-namespace",
			expectedPod:       "my-pod",
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-extend-button",
			"type": "button",
			"value": "my-namespace/my-pod"
		}
	]
}`,
		},
		{
			title:     "invalid value (1)",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-extend-button",
			"type": "button",
			"value": "invalid"
		}
	]
}`,
		},
		{
			title:     "valid value (2)",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-extend-button",
			"type": "button",
			"value": "invalid/invalid/invalid"
		}
	]
}`,
		},
		{
			title:     "blank value",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"actions": [
		{
			"block_id": "slack-agent-extend",
			"action_id": "slack-agent-extend-button",
			"type": "button",
			"value": ""
		}
	]
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cb := new(slack.InteractionCallback)
			if err := json.Unmarshal([]byte(tc.input), cb); err != nil {
				t.Fatal(tc.title, err)
			}

			namespace, pod, err := getPodFromCallbackEvent(cb)
			if !tc.errorCase {
				if tc.expectedNamespace != namespace {
					t.Error(tc.title, "| expected:", tc.expectedNamespace, " actual:", namespace)
				}
				if tc.expectedPod != pod {
					t.Error(tc.title, "| expected:", tc.expectedPod, " actual:", pod)
				}
				if err != nil {
					t.Error(tc.title, "| got error", err)
				}
			} else {
				if err == nil {
					t.Error(tc.title, "| error did not occurred")
				}
			}
		})
	}
}
