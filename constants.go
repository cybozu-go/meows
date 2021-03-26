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
	PodDeletionTimeKey = "actions.cybozu.com/deletedAt"
)

// constants for Slack agent
const (
	// SlackButtonActionID is the unique identifier for the Slack interactive button
	// to extend Pod's lifetime.
	SlackButtonActionID = "slack-agent-extend"

	// MsgJobNameTitle is the title of the job field in Slack message payload.
	MsgJobNameTitle = "Job"

	// MsgPodNameTitle is the title of the timestamp field in Slack message payload.
	MsgTimestampTitle = "Timestamp"

	// MsgPodNameTitle is the title of the Pod field in Slack message payload.
	MsgPodNameTitle = "Pod"

	// MsgPodNameTitle is the title of the namespace field in Slack message payload.
	MsgPodNamespaceTitle = "Namespace"
)
