package client

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestJobInfo(t *testing.T) {
	testCases := []struct {
		title     string
		errorCase bool
		input     *inputEnv

		expectedJobInfo        *JobInfo
		expectedRepositoryURL  string
		expectedWorkflowURL    string
		expectedBranchTagURL   string
		expectedPullRequestURL string
	}{
		{
			title: "branch-push",
			input: &inputEnv{
				GITHUB_ACTOR:      "user",
				GITHUB_HEAD_REF:   "", // blank
				GITHUB_JOB:        "job",
				GITHUB_REF:        "refs/heads/branch/name", // refs/heads/<BRANCH>
				GITHUB_REPOSITORY: "owner/repo",
				GITHUB_RUN_ID:     "123456789",
				GITHUB_RUN_NUMBER: "987",
				GITHUB_WORKFLOW:   "Work flow",
			},
			expectedJobInfo: &JobInfo{
				Actor:          "user",
				GitRef:         "branch/name",
				JobID:          "job",
				PullRequestNum: 0,
				Repository:     "owner/repo",
				RunID:          123456789,
				RunNumber:      987,
				WorkflowName:   "Work flow",
			},
			expectedRepositoryURL:  "https://github.com/owner/repo",
			expectedWorkflowURL:    "https://github.com/owner/repo/actions/runs/123456789",
			expectedBranchTagURL:   "https://github.com/owner/repo/tree/branch/name",
			expectedPullRequestURL: "",
		},
		{
			title: "tag-push",
			input: &inputEnv{
				GITHUB_ACTOR:      "user",
				GITHUB_HEAD_REF:   "", // blank
				GITHUB_JOB:        "job",
				GITHUB_REF:        "refs/tags/v9.9.9", // refs/heads/<TAG>
				GITHUB_REPOSITORY: "owner/repo",
				GITHUB_RUN_ID:     "123456789",
				GITHUB_RUN_NUMBER: "987",
				GITHUB_WORKFLOW:   "Work flow",
			},
			expectedJobInfo: &JobInfo{
				Actor:          "user",
				GitRef:         "v9.9.9",
				JobID:          "job",
				PullRequestNum: 0,
				Repository:     "owner/repo",
				RunID:          123456789,
				RunNumber:      987,
				WorkflowName:   "Work flow",
			},
			expectedRepositoryURL:  "https://github.com/owner/repo",
			expectedWorkflowURL:    "https://github.com/owner/repo/actions/runs/123456789",
			expectedBranchTagURL:   "https://github.com/owner/repo/tree/v9.9.9",
			expectedPullRequestURL: "",
		},
		{
			title: "pullrequest",
			input: &inputEnv{
				GITHUB_ACTOR:      "user",
				GITHUB_HEAD_REF:   "branch-name", // branch name
				GITHUB_JOB:        "job",
				GITHUB_REF:        "refs/pull/123/merge", // refs/pull/<PR_NUM>/merge
				GITHUB_REPOSITORY: "owner/repo",
				GITHUB_RUN_ID:     "123456789",
				GITHUB_RUN_NUMBER: "987",
				GITHUB_WORKFLOW:   "Work flow",
			},
			expectedJobInfo: &JobInfo{
				Actor:          "user",
				GitRef:         "branch-name",
				JobID:          "job",
				PullRequestNum: 123,
				Repository:     "owner/repo",
				RunID:          123456789,
				RunNumber:      987,
				WorkflowName:   "Work flow",
			},
			expectedRepositoryURL:  "https://github.com/owner/repo",
			expectedWorkflowURL:    "https://github.com/owner/repo/actions/runs/123456789",
			expectedBranchTagURL:   "https://github.com/owner/repo/tree/branch-name",
			expectedPullRequestURL: "https://github.com/owner/repo/pull/123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			actual, err := envToJobInfo(tc.input)
			if !tc.errorCase {
				if !cmp.Equal(tc.expectedJobInfo, actual) {
					t.Error(tc.title, "| jobInfo", cmp.Diff(tc.expectedJobInfo, actual))
				}
				if tc.expectedRepositoryURL != actual.RepositoryURL() {
					t.Error(tc.title, "| expected:", tc.expectedRepositoryURL, " actual:", actual.RepositoryURL())
				}
				if tc.expectedWorkflowURL != actual.WorkflowURL() {
					t.Error(tc.title, "| expected:", tc.expectedWorkflowURL, " actual:", actual.WorkflowURL())
				}
				if tc.expectedBranchTagURL != actual.BranchTagURL() {
					t.Error(tc.title, "| expected:", tc.expectedBranchTagURL, " actual:", actual.BranchTagURL())
				}
				if tc.expectedPullRequestURL != actual.PullRequestURL() {
					t.Error(tc.title, "| expected:", tc.expectedPullRequestURL, " actual:", actual.PullRequestURL())
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
