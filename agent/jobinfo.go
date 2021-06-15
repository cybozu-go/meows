package agent

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const DefaultJobInfoFile = "/tmp/github.env"

// JobInfo represents information about a CI job.
type JobInfo struct {
	Actor          string `json:"actor,omitempty"`
	GitRef         string `json:"git_ref,omitempty"`
	JobID          string `json:"job_id,omitempty"`
	PullRequestNum int    `json:"pull_request_number,omitempty"`
	Repository     string `json:"repository,omitempty"`
	RunID          int    `json:"run_id,omitempty"`
	RunNumber      int    `json:"run_number,omitempty"`
	WorkflowName   string `json:"workflow_name,omitempty"`
}

// GetJobInfo reads environment variables and creates JobInfo.
func GetJobInfo() (*JobInfo, error) {
	env := readEnv()
	return envToJobInfo(env)
}

func (info *JobInfo) RepositoryURL() string {
	return fmt.Sprintf("https://github.com/%s", info.Repository)
}

func (info *JobInfo) WorkflowURL() string {
	return fmt.Sprintf("https://github.com/%s/actions/runs/%d", info.Repository, info.RunID)
}

func (info *JobInfo) BranchTagURL() string {
	return fmt.Sprintf("https://github.com/%s/tree/%s", info.Repository, info.GitRef)
}

func (info *JobInfo) PullRequestURL() string {
	if info.PullRequestNum == 0 {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/pull/%d", info.Repository, info.PullRequestNum)
}

type inputEnv struct {
	GITHUB_ACTOR      string
	GITHUB_HEAD_REF   string
	GITHUB_JOB        string
	GITHUB_REF        string
	GITHUB_REPOSITORY string
	GITHUB_RUN_ID     string
	GITHUB_RUN_NUMBER string
	GITHUB_WORKFLOW   string
}

func readEnv() *inputEnv {
	return &inputEnv{
		GITHUB_ACTOR:      os.Getenv("GITHUB_ACTOR"),
		GITHUB_HEAD_REF:   os.Getenv("GITHUB_HEAD_REF"),
		GITHUB_JOB:        os.Getenv("GITHUB_JOB"),
		GITHUB_REF:        os.Getenv("GITHUB_REF"),
		GITHUB_REPOSITORY: os.Getenv("GITHUB_REPOSITORY"),
		GITHUB_RUN_ID:     os.Getenv("GITHUB_RUN_ID"),
		GITHUB_RUN_NUMBER: os.Getenv("GITHUB_RUN_NUMBER"),
		GITHUB_WORKFLOW:   os.Getenv("GITHUB_WORKFLOW"),
	}
}

func envToJobInfo(env *inputEnv) (*JobInfo, error) {
	// Extract git ref and pull request number from env variables.
	// The format is different depending on the type of GitHub event.
	// 1. branch
	//    - GITHUB_REF = "refs/heads/<BRANCH_NAME>"
	// 2. tag
	//    - GITHUB_REF = "refs/tags/<TAG_NAME>"
	// 3. pull request
	//    - GITHUB_REF = "refs/pull/<PR_NUM>/merge"
	//    - GITHUB_HEAD_REF = "<BRANCH_NAME>"
	var gitRef string
	var pullRequestNumber int
	switch {
	case strings.HasPrefix(env.GITHUB_REF, "refs/heads/"):
		gitRef = strings.TrimPrefix(env.GITHUB_REF, "refs/heads/")
	case strings.HasPrefix(env.GITHUB_REF, "refs/tags/"):
		gitRef = strings.TrimPrefix(env.GITHUB_REF, "refs/tags/")
	case strings.HasPrefix(env.GITHUB_REF, "refs/pull/"):
		gitRef = env.GITHUB_HEAD_REF
		split := strings.SplitN(env.GITHUB_REF, "/", 4)
		num, err := strconv.Atoi(split[2])
		if err != nil {
			return nil, fmt.Errorf("failed to parse pull request number: GITHUB_REF = \"%s\", %w", env.GITHUB_REF, err)
		}
		pullRequestNumber = num
	default:
		return nil, fmt.Errorf("unknown format: GITHUB_REF = \"%s\"", env.GITHUB_REF)
	}

	runID, err := strconv.Atoi(env.GITHUB_RUN_ID)
	if err != nil {
		return nil, fmt.Errorf("invalid value: GITHUB_RUN_ID = \"%s\", %w", env.GITHUB_RUN_ID, err)
	}
	runNumber, err := strconv.Atoi(env.GITHUB_RUN_NUMBER)
	if err != nil {
		return nil, fmt.Errorf("invalid value: GITHUB_RUN_NUMBER = \"%s\", %w", env.GITHUB_RUN_NUMBER, err)
	}

	return &JobInfo{
		Actor:          env.GITHUB_ACTOR,
		GitRef:         gitRef,
		JobID:          env.GITHUB_JOB,
		PullRequestNum: pullRequestNumber,
		Repository:     env.GITHUB_REPOSITORY,
		RunID:          runID,
		RunNumber:      runNumber,
		WorkflowName:   env.GITHUB_WORKFLOW,
	}, nil
}
