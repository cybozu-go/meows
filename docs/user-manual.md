User Manual
===========

How to create GitHub App
------------------------

GitHub Actions controller is responsible for the registering/deregistering `Pod`s
to/from a GitHub Actions runner and the controller requires to have a GitHub App
secret.

Please create a GitHub App first on the GitHub page following [the official documentation](https://docs.github.com/en/developers/apps/creating-a-github-app).
Here are the minimal changes from the default setting on the regitration page:

- Fill **GitHub Apps Name**
- Fill **Homepage URL**
- Uncheck `Active` under **Webhook** section
- Set **Administration** `Read & Write` permission to the repository scope

Then, you are redirected to the **General** page and what you should do is:

1. Click `Generate a private key` and downloads the generated private key.
1. Keep `App ID` shown on the top of the page somewhere.

Next, you should proceed to the Install App page from the sidebar and click the
`Install` button. You are asked to give the permission to `All repositories`
or `Only select repositories`. `Only select repositories` is recommended because
the permission is very strong. You should decide the scope wide enough depending on
how you use this controller.

Finally, you should get the installation ID from the URL of which page you are
redicted to. The URL should look like `https://github.com/organizations/cybozu-go/settings/installations/12345`
and `12345` is your installation ID.

How to deploy GitHub Actions controller
---------------------------------------

### Manifests

You should deploy the controller `Deployment` with a `Secret` resource which
contains the GitHub App private key you created.  The command below creates the
`Secret` resource.

```bash
kubectl create secret generic github-app-secret \
  -n ${NAMESPACE} \
  --from-literal=app-id=${GITHUB_APP_ID} \
  --from-literal=app-installation-id=${GITHUB_APP_INSTALLATION_ID} \
  --from-file=app-private-key=${GITHUB_APP_PRIVATE_KEY_PATH}
```

In addition to this, the admission webhook requires a TLS certificate.
You should use [`github.com/jetstack/cert-manager`](https://github.com/jetstack/cert-manager)
or create a certificate by yourself.

This document does not give you a comprehensive list of what you shold deploy.
Please refer to the manifests under `config/` for further information.

### CLI options

The CLI allows you to use the following options:

```bash
$ controller -h
Kubernetes controller for GitHub Actions self-hosted runner

Usage:
  GitHub Actions controller [flags]

Flags:
      --add_dir_header                     If true, adds the file directory to the header
      --alsologtostderr                    log to standard error as well as files
      --app-id int                         The ID for GitHub App.
      --app-installation-id int            The installation ID for GitHub App.
      --app-private-key-path string        The path for GitHub App private key.
      --health-probe-bind-address string   The address the probe endpoint binds to. (default ":8081")
  -h, --help                               help for GitHub
      --log_backtrace_at traceLocation     when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                     If non-empty, write log files in this directory
      --log_file string                    If non-empty, use this log file
      --log_file_max_size uint             Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                        log to standard error instead of files (default true)
      --metrics-bind-address string        The address the metric endpoint binds to. (default ":8080")
  -o, --organization-name string           The GitHub organization name
      --pod-sweep-interval duration        Interval to watch and delete Pods. (default 1m0s)
  -r, --repository-names strings           The GitHub repository names, separated with comma.
      --runner-sweep-interval duration     Interval to watch and sweep unused GitHub Actions runners. (default 30m0s)
      --skip_headers                       If true, avoid header prefixes in the log messages
      --skip_log_headers                   If true, avoid headers when opening log files
      --stderrthreshold severity           logs at or above this threshold go to stderr (default 2)
  -v, --v Level                            number for the log level verbosity
      --vmodule moduleSpec                 comma-separated list of pattern=N settings for file-filtered logging
      --webhook-addr string                The address the webhook endpoint binds to (default ":9443")
      --zap-devel                          Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
      --zap-encoder encoder                Zap log encoding (one of 'json' or 'console')
      --zap-log-level level                Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
      --zap-stacktrace-level level         Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
```

How to create Slack App
-----------------------

Slack agent notifies users whether CI succeeds or not and receives messages to
extend a `Pod` lifetime.
So, users have to get a Webhook URL to send messages to and run a WebSocket client
to watch button events.

Here's a procedure for how to configure the Slack App.

1. Go to [this](https://api.slack.com/apps) page.
1. Click the `Create New App` button and fill the application name field and choose
   which workspace to develop it on.
1. Go to **Socket Mode** from the sidebar and enable socket mode.
1. Then, a window comes up. Create App-Level Token and keep the generated App
   Token.
1. Go to **OAuth & Permissions** from the sidebar and keep **Bot User OAuth Token**.
1. Go to **Incoming Webhooks** from the sidebar and click **Add New Webhook to Workspace**.
   Choose the channel you want to use and keep the generated URL.
1. Go to **Beta features** from the sidebar and enable time picker element.

How to deploy Slack agent
-------------------------

### Manifests

You should deploy the Slack agent `Deployment` with a `Secret` resource which
contains the Slack App tokens you created. The command below creates the `Secret`
resource.

```bash
kubectl create secret generic slack-app-secret \
  -n ${NAMESPACE} \
  --from-literal=SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL} \
  --from-literal=SLACK_APP_TOKEN=${SLACK_APP_TOKEN} \
  --from-literal=SLACK_BOT_TOKEN=${SLACK_BOT_TOKEN}
```

Please refer to the manifest `config/agent/manifests.yaml` for detail.

### CLI options

The Slack agent CLI has subcommands `notifier`, `extender` and `client`.

- `notifier`: Send messages to Webhook
- `extender`: Receive button events and annotate a `Pod` with a deletion time
- `client`: Send requests to `notifier`

The Slack agent `Pod` runs both `notifier` and `extender` in different containers.
These subcommands allow you to use the following options:

```bash
$ slack-agent notifier -h
notifier starts Slack agent to send job results to Slack

Usage:
  Slack notifier [flags]

Flags:
  -h, --help                 help for notifier
      --listen-addr string   The address the notifier endpoint binds to (default ":8080")
  -o, --webhook-url string   The Slack Webhook URL to notify messages to

Global Flags:
  -d, --development        Development mode.
      --logfile string     Log filename
      --logformat string   Log format [plain,logfmt,json]
      --loglevel string    Log level [critical,error,warning,info,debug]


$ slack-agent extender -h
notifier starts Slack agent to receive interactive messages from Slack

Usage:
  Slack extender [flags]

Flags:
      --app-token string   The Slack App token.
      --bot-token string   The Slack Bot token.
  -h, --help               help for extender
      --retry uint         How many times the extender retries to connect Slack.

Global Flags:
  -d, --development        Development mode.
      --logfile string     Log filename
      --logformat string   Log format [plain,logfmt,json]
      --loglevel string    Log level [critical,error,warning,info,debug]
```

RunnerPool Custom Resource Example
----------------------------------

This is an example of the `RunnerPool` custom resource.

```yaml
apiVersion: actions.cybozu.com/v1alpha1
kind: RunnerPool
metadata:
  name: runnerpool-sample
spec:
  repositoryName: repository-sample
  slackAgentURL: slack-agent
  replicas: 3
  selector:
    matchLabels:
      app: actions-runner
  template:
    metadata:
      labels:
        app: actions-runner
    spec:
      containers:
      - name: runner
        image: quay.io/cybozu/actions-runner:latest
      serviceAccountName: runner
```

The controller creates a `Deployment` based on `template` in `RunnerPool`.
It modifies `template` just to satisfy the minimal requirement to run GitHub Actions.

The controller is mainly responsible for:

- Add the GitHub organization name specified with the controller CLI's option and
  the repository name defined in the `RunnerPool` manifest to `metadata.labels`.
- Add environment variables needed to run `entrypoing.sh`.
  `entrypoing.sh` is a default command for the runner container and contains
  [`github.com/actions/runner`](https://github.com/actions/runner).

You are responsible for:

- A `Pod` created with `RunnerPool` annotates itself with what time to delete the
  `Pod` when CI finishes and needs the `patch` role for `Pod`. The corresponding
  `ServiceAccount` is not created by the controller for now, so please create it
  by yourself.  Please refer to the manifest `config/samples/manifests.yaml` for
  detail.
- `slackAgentURL` is a URL that can be resolved inside a Kubernetes cluster, so
  a `Service` name is acceptable. If `slackAgentURL` is omitted, the `Pod`s do
  not send notifications to Slack.

How to use self-hosted runners
------------------------------

There are some small scripts provided under `scripts/`.

- `job-started`
- `job-failed`

You should include these script in GitHub Actions workflows you execute on
self-hosted runners.
Here is an example for how to define a workflow.

```yaml
name: Main
on:
  pull_request:
  push:
    branches:
      - 'main'

jobs:
  build:
    name: build
    runs-on: self-hosted
    steps:
      - run: job-started
      - run: ...
      - run: ...
      - run: ...
      - if: failure()
        run: job-failed
```

How to extend GitHub Actions jobs
---------------------------------

![failure message](./images/slack_failure.png)

Choose the time in UTC you want to extend the `Pod` by and click `Extend`.
This button can be clicked multiple times if the `Pod` still exists.

NOTE:  
The time picker feature is still beta, and [`github.com/slack-go/slack`](https://github.com/slack-go/slack)
cannot parse interactive messages which contains a time picker component.

