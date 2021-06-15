package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestIsExtendButtonEvent(t *testing.T) {
	testCases := []struct {
		title    string
		expected bool
		input    string
	}{
		{
			title:    "valid",
			expected: true,
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
			title:    "unexpected type",
			expected: false,
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
			title:    "unexpected block_id",
			expected: false,
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
			title:    "unexpected action_id",
			expected: false,
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
			title:    "no action",
			expected: false,
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
			if tc.expected != actual {
				t.Error(tc.title, "| expected:", tc.expected, " actual:", actual)
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

func parseTime(value string) time.Time {
	if len(value) == 0 {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}

func TestGetTimeFromCallbackEvent(t *testing.T) {
	const baseTimeString = "2021-12-30T11:22:33Z"
	testCases := []struct {
		title     string
		errorCase bool
		expected  string
		input     string
	}{
		{
			title:    "valid (1)",
			expected: "2021-12-30T09:09:00Z",
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"slack-agent-extend": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": "09:09"
				}
			}
		}
	}
}`,
		},
		{
			title:    "valid (2)",
			expected: "2021-12-30T00:59:00Z",
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"slack-agent-extend": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": "00:59"
				}
			}
		}
	}
}`,
		},
		{
			title:     "unexpected block_id",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"unexpected": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": "mm:dd"
				}
			}
		}
	}
}`,
		},
		{
			title:     "unexpected action_id",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"slack-agent-extend": {
				"unexpected": {
					"type": "timepicker",
					"selected_time": "mm:dd"
				}
			}
		}
	}
}`,
		},
		{
			title:     "invalid value (1)",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"slack-agent-extend": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": "mm:dd"
				}
			}
		}
	}
}`,
		},
		{
			title:     "invalid value (2)",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"slack-agent-extend": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": "12"
				}
			}
		}
	}
}`,
		},
		{
			title:     "blank value",
			errorCase: true,
			input: `
{
	"type": "block_actions",
	"state": {
		"values": {
			"slack-agent-extend": {
				"slack-agent-extend-timepicker": {
					"type": "timepicker",
					"selected_time": ""
				}
			}
		}
	}
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cb := new(slack.InteractionCallback)
			if err := json.Unmarshal([]byte(tc.input), cb); err != nil {
				t.Fatal(tc.title, err)
			}
			baseTime := parseTime(baseTimeString)
			expected := parseTime(tc.expected)

			actual, err := getTimeFromCallbackEvent(cb, baseTime)
			if !tc.errorCase {
				if !expected.Equal(actual) {
					t.Error(tc.title, "| expected:", expected, " actual:", actual)
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
