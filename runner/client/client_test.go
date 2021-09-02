package client

import (
	"encoding/json"
	"testing"
	"time"

	"k8s.io/utils/pointer"
)

func TestJobResultResponse(t *testing.T) {
	finishedAt, err := time.Parse("2006-Jan-02", "2021-Jan-01")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		input    *JobResultResponse
		expected string
	}{
		{
			input:    &JobResultResponse{Status: JobResultUnfinished},
			expected: `{"status":"unfinished"}`,
		},
		{
			input: &JobResultResponse{
				Status:     JobResultUnknown,
				FinishedAt: &finishedAt,
				Extend:     pointer.Bool(true),
				JobInfo: &JobInfo{
					Actor:          "user",
					GitRef:         "branch/name",
					JobID:          "job",
					PullRequestNum: 0,
					Repository:     "owner/repo",
					RunID:          123456789,
					RunNumber:      987,
					WorkflowName:   "Work flow",
				},
			},
			expected: `{"status":"unknown","finished_at":"2021-01-01T00:00:00Z","extend":true,"job_info":{"actor":"user","git_ref":"branch/name","job_id":"job","repository":"owner/repo","run_id":123456789,"run_number":987,"workflow_name":"Work flow"}}`,
		},
	}

	for _, tc := range testCases {
		s, err := json.Marshal(tc.input)
		if err != nil {
			t.Error("json.Marshal is failed", err, tc.input)
		}
		if string(s) != tc.expected {
			t.Error("json is not matched", string(s), tc.expected)
		}
	}
}
