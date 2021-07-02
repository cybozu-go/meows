package constants

// Container names
const (
	// RunnerContainerName is a container name which runs GitHub Actions runner.
	RunnerContainerName = "runner"
)

// Metadata keys
const (
	// RunnerOrgLabelKey is a label key for organization name.
	RunnerOrgLabelKey = "actions.cybozu.com/organization"

	// RunnerRepoLabelKey is a label key for repository name.
	RunnerRepoLabelKey = "actions.cybozu.com/repository"

	// PodDeletionTimeKey is an annotation key to manage pod deletion time.
	PodDeletionTimeKey = "actions.cybozu.com/deleted-at"

	// RunnerPoolFinalizer is a finalizer for runnerpool resource.
	RunnerPoolFinalizer = "actions.cybozu.com/runnerpool"

	// AppNameLabelKey is a label key for application name.
	AppNameLabelKey = "app.kubernetes.io/name"

	// AppComponentLabelKey is a label key for the component.
	AppComponentLabelKey = "app.kubernetes.io/component"

	// AppInstanceLabelKey is a label key for the instance name.
	AppInstanceLabelKey = "app.kubernetes.io/instance"
)

const (
	// AppName is the application name.
	AppName = "actions"

	// AppComponentRunner is the component name for runner.
	AppComponentRunner = "runner"
)

// Container ports
const (
	// RunnerMetricsPort is the port number for runner container.
	RunnerMetricsPort = 8080

	// RunnerMetricsPortName is the port name for runner container.
	RunnerMetricsPortName = "metrics"
)

// Container endpoints
const (
	// DeletionTimeEndpoint is the endpoint to get deletion time for runner container.
	DeletionTimeEndpoint = "deletion_time"
)

// Environment variables
const (
	// PodNameEnvName is a env field key for POD_NAME.
	PodNameEnvName = "POD_NAME"

	// PodNamespaceEnvName is a env field key for POD_NAME.
	PodNamespaceEnvName = "POD_NAMESPACE"

	// RunnerOrgEnvName is a env field key for RUNNER_ORG.
	RunnerOrgEnvName = "RUNNER_ORG"

	// RunnerRepoEnvName is a env field key for RUNNER_REPO.
	RunnerRepoEnvName = "RUNNER_REPO"

	// RunnerPoolNameEnvName is a env field key for RUNNER_POOL_NAME.
	RunnerPoolNameEnvName = "RUNNER_POOL_NAME"

	// SlackAgentEnvName is a env field key for SLACK_AGENT_SERVICE_NAME.
	SlackAgentEnvName = "SLACK_AGENT_SERVICE_NAME"

	// RunnerTokenEnvName is a env field key for RUNNER_TOKEN.
	RunnerTokenEnvName = "RUNNER_TOKEN"

	// ExtendDurationEnvName is a env field key for EXTEND_DURATION
	ExtendDurationEnvName = "EXTEND_DURATION"
)
