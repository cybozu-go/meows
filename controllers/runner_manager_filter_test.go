package controllers

import (
	"testing"

	"github.com/cybozu-go/meows/runner"
)

func TestResultMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		results []string
		result  string
		want    bool
	}{
		{
			name:    "empty filter notifies success",
			results: nil,
			result:  runner.JobResultSuccess,
			want:    true,
		},
		{
			name:    "empty filter notifies failure",
			results: []string{},
			result:  runner.JobResultFailure,
			want:    true,
		},
		{
			name:    "failure-only filter drops success",
			results: []string{runner.JobResultFailure},
			result:  runner.JobResultSuccess,
			want:    false,
		},
		{
			name:    "failure-only filter allows failure",
			results: []string{runner.JobResultFailure},
			result:  runner.JobResultFailure,
			want:    true,
		},
		{
			name:    "multi filter allows listed",
			results: []string{runner.JobResultFailure, runner.JobResultCancelled},
			result:  runner.JobResultCancelled,
			want:    true,
		},
		{
			name:    "multi filter drops unlisted",
			results: []string{runner.JobResultFailure, runner.JobResultCancelled},
			result:  runner.JobResultUnknown,
			want:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resultMatches(makeResultSet(tc.results), tc.result)
			if got != tc.want {
				t.Errorf("resultMatches(makeResultSet(%v), %q) = %v; want %v", tc.results, tc.result, got, tc.want)
			}
		})
	}
}
