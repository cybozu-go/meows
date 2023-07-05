package constants

const (
	// Version is the meows version.
	Version = "0.12.0"
)

// Container names
const (
	// RunnerContainerName is a container name which runs GitHub Actions runner.
	RunnerContainerName = "runner"
)

// Metadata keys
const (
	RunnerSecretExpiresAtAnnotationKey = "meows.cybozu.com/expires-at"

	// RunnerPoolFinalizer is a finalizer for runnerpool resource.
	RunnerPoolFinalizer = "meows.cybozu.com/runnerpool"

	// AppNameLabelKey is a label key for application name.
	AppNameLabelKey = "app.kubernetes.io/name"

	// AppComponentLabelKey is a label key for the component.
	AppComponentLabelKey = "app.kubernetes.io/component"

	// AppInstanceLabelKey is a label key for the instance name.
	AppInstanceLabelKey = "app.kubernetes.io/instance"

	// RunnerPodName is the label key to select individual pod.
	RunnerPodName = "meows.cybozu.com/runner-pod-name"
)

const (
	// AppName is the application name.
	AppName = "meows"

	// AppComponentRunner is the component name for runner.
	AppComponentRunner = "runner"
)

// Container ports
const (
	// RunnerListenPort is the port number for runner container.
	RunnerListenPort = 8080

	// RunnerMetricsPortName is the port name for runner container.
	RunnerMetricsPortName = "metrics"
)

// Container endpoints
const (
	// DeletionTimeEndpoint is the endpoint to get deletion time for runner container.
	DeletionTimeEndpoint = "deletion_time"

	// StatusEndPoint is the endpoint to get status of a runner pod.
	StatusEndPoint = "status"
)

// Runner pods state.
const (
	RunnerPodStateInitializing = "initializing"
	RunnerPodStateRunning      = "running"
	RunnerPodStateDebugging    = "debugging"
	RunnerPodStateStale        = "stale"
)

// Exit state of Actions Listener.
const (
	ListenerExitStateRetryableError = "retryable_error"
	ListenerExitStateUpdating       = "updating"
	ListenerExitStateUndefined      = "undefined"
)

// Constants for controller option configmap.
const (
	// OptionConfigMapName is a configmap name for the controller option.
	OptionConfigMapName = "meows-cm"

	// Data keys for controller option.
	OptionConfigMapDataOrganizationRule = "organization-rule"
	OptionConfigMapDataRepositoryRule   = "repository-rule"
)

// Constants for GitHub credential secret.
const (
	// DefaultCredentialSecretName is the default secret name for GitHub credential secret.
	DefaultCredentialSecretName = "meows-github-cred"

	// Data keys for GitHub App's credential.
	CredentialSecretDataAppID             = "app-id"
	CredentialSecretDataAppInstallationID = "app-installation-id"
	CredentialSecretDataAppPrivateKey     = "app-private-key"

	// Data keys for GitHub personal access token (PAT).
	CredentialSecretDataPATToken = "token"
)

const (
	DefaultSlackAgentServiceName = "slack-agent.meows.svc"
)

// Directory path for runner pods.
const (
	// RunnerRootDirPath is a directory path where GitHub Actions Runner will be installed.
	RunnerRootDirPath = "/runner"

	// RunnerWorkDirPath is a working directory path for job execution.
	RunnerWorkDirPath = "/runner/_work"

	// RunnerVarDirPath is a directory path for storing variable files.
	RunnerVarDirPath = "/var/meows"

	// SlackChannelFilePath is a file path for the Slack channel to be notified.
	SlackChannelFilePath = RunnerVarDirPath + "/slack_channel"

	// SecretsDirName is a directory name for storing secret files.
	SecretsDirName = "secrets"

	// RunnerTokenFileName is a file name for GitHub registration token.
	RunnerTokenFileName = "runnertoken"
)

// Environment variables
const (
	// PodNameEnvName is a env field key for POD_NAME.
	PodNameEnvName = "POD_NAME"

	// PodNamespaceEnvName is a env field key for POD_NAMESPACE.
	PodNamespaceEnvName = "POD_NAMESPACE"

	// RunnerOrgEnvName is a env field key for RUNNER_ORG.
	RunnerOrgEnvName = "RUNNER_ORG"

	// RunnerRepoEnvName is a env field key for RUNNER_REPO.
	RunnerRepoEnvName = "RUNNER_REPO"

	// RunnerPoolNameEnvName is a env field key for RUNNER_POOL_NAME.
	RunnerPoolNameEnvName = "RUNNER_POOL_NAME"

	// RunnerOptionEnvName is a env field key for RUNNER_OPTION
	RunnerOptionEnvName = "RUNNER_OPTION"

	// SlackChannelEnvName is a env field key for MEOWS_SLACK_CHANNEL
	SlackChannelEnvName = "MEOWS_SLACK_CHANNEL"
)
