CLI options
===========

`controller`
-----------

The CLI allows you to use the following options:

```bash
$ controller -h
Kubernetes controller for GitHub Actions self-hosted runner

Usage:
  controller [flags]

Flags:
      --add_dir_header                     If true, adds the file directory to the header
      --alsologtostderr                    log to standard error as well as files
      --app-id int                         The ID for GitHub App.
      --app-installation-id int            The installation ID for GitHub App.
      --app-private-key-path string        The path for GitHub App private key.
      --health-probe-bind-address string   The address the probe endpoint binds to. (default ":8081")
  -h, --help                               help for controller
      --log_backtrace_at traceLocation     when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                     If non-empty, write log files in this directory
      --log_file string                    If non-empty, use this log file
      --log_file_max_size uint             Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logfile string                     Log filename
      --logformat string                   Log format [plain,logfmt,json]
      --loglevel string                    Log level [critical,error,warning,info,debug]
      --logtostderr                        log to standard error instead of files (default true)
      --metrics-bind-address string        The address the metric endpoint binds to. (default ":8080")
  -o, --organization-name string           The GitHub organization name
  -r, --repository-names strings           The GitHub repository names, separated with comma.
      --runner-image string                The image of runner container (default "quay.io/cybozu/meows-runner:latest")
      --runner-manager-interval duration   Interval to watch and delete Pods. (default 1m0s)
      --secret-updater-interval duration   Interval to watch and update secret. (default 1m0s)
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


`slack-agent`
-------------

The Slack agent is a server program.
This notifies CI results and accepts requests for extending Pods' lifecycles

```bash
$ slack-agent -h
Slack agent notifies CI results and accepts requests for extending Pods' lifecycles

Usage:
  slack-agent [flags]

Flags:
      --app-token string     The Slack App token.
      --bot-token string     The Slack Bot token.
  -c, --channel string       The Slack channel to notify messages to
  -d, --development          Development mode.
  -h, --help                 help for slack-agent
      --listen-addr string   The address the notifier endpoint binds to (default ":8080")
      --logfile string       Log filename
      --logformat string     Log format [plain,logfmt,json]
      --loglevel string      Log level [critical,error,warning,info,debug]
  -v, --verbose              Verbose.
```


`meows`
------

This is a tool command to do some operations.
It enables to send requests to the slack-agent, or to control the GitHub runners.

### `meows slackagent send RUNNER_PODNAME`

This sub command sends a request to the slack-agent.

### `meows runner list REPOSITORY`

This sub command lists runners on the specified repository.

### `meows runner remove REPOSITORY`

This sub command removes **offline** runners on the specified repository.
