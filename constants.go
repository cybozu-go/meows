package constants

// constants for GitHub actions controller
const (
	// RunnerContainerName is a container name which runs GitHub Actions runner.
	RunnerContainerName = "runner"

	// PodNameEnvName is a env field key for POD_NAME.
	PodNameEnvName = "POD_NAME"

	// PodNamespaceEnvName is a env field key for POD_NAME.
	PodNamespaceEnvName = "POD_NAMESPACE"

	// RunnerOrgEnvName is a env field key for RUNNER_ORG.
	RunnerOrgEnvName = "RUNNER_ORG"

	// RunnerRepoEnvName is a env field key for RUNNER_REPO.
	RunnerRepoEnvName = "RUNNER_REPO"

	// SlackAgentEnvName is a env field key for SLACK_AGENT_URL.
	SlackAgentEnvName = "SLACK_AGENT_URL"

	// RunnerTokenEnvName is a env field key for RUNNER_TOKEN.
	RunnerTokenEnvName = "RUNNER_TOKEN"

	// RunnerOrgLabelKey is a label key for organization name.
	RunnerOrgLabelKey = "actions.cybozu.com/organization"

	// RunnerRepoLabelKey is a label key for repository name.
	RunnerRepoLabelKey = "actions.cybozu.com/repository"

	// PodDeletionTimeKey is an annotation key to manage pod deletion time.
	PodDeletionTimeKey = "actions.cybozu.com/deleted-at"
)
