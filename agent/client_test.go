package agent

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMakePayload(t *testing.T) {
	branchPushJob := &JobInfo{
		Actor:          "user",
		GitRef:         "branch/name",
		JobID:          "job",
		PullRequestNum: 0,
		Repository:     "owner/repo",
		RunID:          123456789,
		RunNumber:      987,
		WorkflowName:   "Work flow",
	}
	pullRequestJob := &JobInfo{
		Actor:          "user",
		GitRef:         "branch-name",
		JobID:          "job",
		PullRequestNum: 123,
		Repository:     "owner/repo",
		RunID:          123456789,
		RunNumber:      987,
		WorkflowName:   "Work flow",
	}

	testCases := []struct {
		title string

		inputResult    string
		inputNamespace string
		inputPod       string
		inputJobInfo   *JobInfo

		expected *resultAPIPayload
	}{
		{
			title: "success (info=nil)",

			inputResult:    "success",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   nil,

			expected: &resultAPIPayload{
				Color: colorGreen,
				Text:  "Success: (failed to get job status)",
				Job:   "(unknown)",
				Pod:   "my-namespace/my-pod",
			},
		},
		{
			title: "failure (info=nil)",

			inputResult:    "failure",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   nil,

			expected: &resultAPIPayload{
				Color: colorRed,
				Text:  "Failure: (failed to get job status)",
				Job:   "(unknown)",
				Pod:   "my-namespace/my-pod",
			},
		},
		{
			title: "cancelled (info=nil)",

			inputResult:    "cancelled",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   nil,

			expected: &resultAPIPayload{
				Color: colorGray,
				Text:  "Cancelled: (failed to get job status)",
				Job:   "(unknown)",
				Pod:   "my-namespace/my-pod",
			},
		},
		{
			title: "unknown (info=nil)",

			inputResult:    "unknown",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   nil,

			expected: &resultAPIPayload{
				Color: colorYellow,
				Text:  "Finished(Unknown): (failed to get job status)",
				Job:   "(unknown)",
				Pod:   "my-namespace/my-pod",
			},
		},
		{
			title: "unexpected (info=nil)",

			inputResult:    "unexpected",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   nil,

			expected: &resultAPIPayload{
				Color: colorYellow,
				Text:  "Finished(Unknown): (failed to get job status)",
				Job:   "(unknown)",
				Pod:   "my-namespace/my-pod",
			},
		},
		{
			title: "branch push",

			inputResult:    "success",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   branchPushJob,

			expected: &resultAPIPayload{
				Color: colorGreen,
				Text:  "Success: user's CI job in <https://github.com/owner/repo|owner/repo>",
				Job:   "<https://github.com/owner/repo/actions/runs/123456789|Work flow #987> [job]",
				Pod:   "my-namespace/my-pod",
			},
		},
		{
			title: "pull request",

			inputResult:    "failure",
			inputNamespace: "my-namespace",
			inputPod:       "my-pod",
			inputJobInfo:   pullRequestJob,

			expected: &resultAPIPayload{
				Color: colorRed,
				Text:  "Failure: user's CI job in <https://github.com/owner/repo|owner/repo>",
				Job:   "<https://github.com/owner/repo/actions/runs/123456789|Work flow #987> [job]",
				Pod:   "my-namespace/my-pod",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			actual := makePayload(tc.inputResult, tc.inputNamespace, tc.inputPod, tc.inputJobInfo)
			if !cmp.Equal(tc.expected, actual) {
				t.Error(tc.title, "| payload", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
