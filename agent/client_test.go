package agent

import (
	"testing"

	"github.com/cybozu-go/meows/runner"
	"github.com/google/go-cmp/cmp"
)

func TestMakePayload(t *testing.T) {
	branchPushJob := &runner.JobInfo{
		Actor:          "user",
		GitRef:         "branch/name",
		JobID:          "job",
		PullRequestNum: 0,
		Repository:     "owner/repo",
		RunID:          123456789,
		RunNumber:      987,
		WorkflowName:   "Work flow",
	}
	pullRequestJob := &runner.JobInfo{
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
		inputJobInfo   *runner.JobInfo

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

// TestJobResultConstantsMatchAgentMaps guards against drift between
// runner.JobResult* constants and the colors/captions maps here. The same
// set of values is also referenced by the NotifyOn enum in
// api/v1alpha1/runnerpool_types.go; if a new JobResult is added, update all
// three sites together.
func TestJobResultConstantsMatchAgentMaps(t *testing.T) {
	jobResults := []string{
		runner.JobResultSuccess,
		runner.JobResultFailure,
		runner.JobResultCancelled,
		runner.JobResultUnknown,
	}

	for _, r := range jobResults {
		if _, ok := colors[r]; !ok {
			t.Errorf("colors map is missing entry for JobResult %q", r)
		}
		if _, ok := captions[r]; !ok {
			t.Errorf("captions map is missing entry for JobResult %q", r)
		}
	}

	if len(colors) != len(jobResults) {
		t.Errorf("colors has %d entries; want %d (one per JobResult constant)", len(colors), len(jobResults))
	}
	if len(captions) != len(jobResults) {
		t.Errorf("captions has %d entries; want %d (one per JobResult constant)", len(captions), len(jobResults))
	}
}
