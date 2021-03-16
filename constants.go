package githubactionscontroller

const (
	// RunnerContainerName is a container name which runs GitHub Actions runner.
	RunnerContainerName = "runner"

	// RunnerNameEnvName is a env field key for RUNNER_NAME.
	RunnerNameEnvName = "RUNNER_NAME"

	// RunnerOrgEnvName is a env field key for RUNNER_ORG.
	RunnerOrgEnvName = "RUNNER_ORG"

	// RunnerRepoEnvName is a env field key for RUNNER_REPO.
	RunnerRepoEnvName = "RUNNER_REPO"

	// RunnerTokenEnvName is a env field key for RUNNER_TOKEN.
	RunnerTokenEnvName = "RUNNER_TOKEN"

	// RunnerOrgLabelKey is a label key for organization name.
	RunnerOrgLabelKey = "actions.cybozu.com/organization"

	// RunnerRepoLabelKey is a label key for repository name.
	RunnerRepoLabelKey = "actions.cybozu.com/repository"

	// PodDeletionTimeKey is an annotation key to manage pod deletion time.
	PodDeletionTimeKey = "runnerDeleteAt"
)
