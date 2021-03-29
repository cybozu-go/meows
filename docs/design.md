Design notes
============

Motivation
----------

To run GitHub Actions self-hosted runners faster and stably by making full use of
idle machine resources.

Goals
-----

- Deploy and manage GitHub Actions self-hosted runners on multiple servers easily
  by using Kubernetes.
- Enable runners to finish a time-consuming initialization step before jobs are
  assigned in order not to make users wait longer.
- Extend lifetime of runners from outside when jobs are failed, to investigate
  what causes the failure.
- Notify users whether jobs are failed or not via Slack and extend the lifetime
  from Slack.

Non-Goals
---------

- Autoscaling

Components
------------

- Runner `Pod`: `Pod` to run GitHub Actions self-hosted runner on.
- Controller manager
  - `RunnerPool` reconciler: A controller for the `RunnerPool` custom resource(CR).
  -  `Pod` Mutating webhook: A mutating webhook to inject a GitHub Actions
     registration token to `Pod` `env`.
  - Runner sweeper: A component to sweep registered information about runners
    on GitHub periodically.
  - `Pod` sweeper: A component to sweep Pods which exceeds the deletion time
    limit periodically.
- Slack agent
  - Notifier: HTTP Server which accepts requests from Runner `Pod`s and notify user
    whether jobs are failed or not via Slack Webhook.
  - Extender: WebSocket client which watches Slack button events and extends the
    lifetime of a `Pod`.

Diagram: T.B.A

Architecture
------------

### How self-hosted runners are created and runs jobs

1. The `RunnerPool` reconciler watches `RunnerPool` creation events and creates
  a `Deployment` which own runner `Pod`s. The controller adds some labels to
  `Pod`s to tell other components like webhook that the `Pod`s are managed by
  the controller. After this steps, Other components recognize which `Pod`s
  they should handle with those labels.
1. The mutating webhook creates registration tokens via GitHub API and injects
  them to the `Pod`s' `env` fields.
1. Each runner `Pod` runs the following commands.
   1. Register itself as a self-hosted runner with the injected token.
   1. Initialize runner environment by doing the user-defined process.
   1. Start a long polling process and wait for GitHub Actions to assign a job.
   1. Run an assigned job.
   1. Call the Slack agent to notify users. GitHub API does not seem to provide
      a way to know which runner ran a succeeded or failed job. So, this repository
      provides a simple `job-failed` command, and asks users to execute this
      command when the job is failed.  The `if: failure()` syntax allows users
      to run the step only when one of previous steps exit with non-zero code.
   1. Annotate the `Pod` manifest for itself with a timestamp when to delete this
     `Pod`. If the job is succeded or canceled, the `Pod` annotates itself with
      the current time.If the job is failed, the `Pod` annotate itself with the
      future time, for example 20 min later.
1. The Slack agent notifies the result of the job on a Slack channel.
1. Users can extend the failed runner if they want to by clicking a button on Slack.
1. The Slack agent is running a WebSocket process to watch extending messages.
  If it receives a message, it annotates the `Pod` manifest with the designated time.
1. `Pod` sweeper periodically checks if there are `Pod`s annotated with the old
  timestamp, and if any, it deletes `Pod`s.

### How Runner's state is managed

A Runner `Pod` has the following state as a GitHub Actions job runner.

- registered: `Pod` registered itself on GitHub Actions.
- initialized: `Pod` finished the initialization.
- listening: `Pod` starts listening with Long Polling.
- assigned: `Pod` is assigned a job and starts running it.
- debug: The job has finished with failure and Users can enter `Pod` to debug.

Runner `Pod`s might not need to manage all of these state, but some of them are
useful to check the system is properly running or not.
The state "debug" is a must to be watched to keep runner `Pod`s in stock.
Users can leave many failed runner `Pod`s for investigation, but the number of
replicas is also limited. Exposing the state as Prometheus metrics allows users
to aggregate the number of `Pod`s in each state and raise alerts that they are
running out of available runners.

However, as mentioned above, GitHub API does not provide a way to connect the job
status information with the self-hosted runner.  So, runner `Pod`s have to execute
commands to tell the metrics exporter server that the state has changed.

- Notifier

### Why is webhook needed ?

This is mainly because GitHub Actions registration token expires after 1 hour as written [here](https://docs.github.com/en/rest/reference/actions#create-a-registration-token-for-a-repository).

- If a token is given in the `env` field in a pod template of `Deployment`, `Pod`
  cannot register itself again when recreated.
- If a token is given with a `Secret` resource mounted on the `Pod`, recreated
  `Pod`s can access an updated token.
  However, webhook is simpler because only the webhook is responsible for tokens.
  Both a controller and a periodical runner have to take responsibility to manage
  tokens in this alternative design.
- If a token is given in the `env` field in a pod template of `Deployment` and
  `Pod` restarts by having liveness probe fail, `Pod`s do not need to re-register
  itself to GitHub Actions.
  However, if `Pod`s are recreated in some reasons, it cannot register itself again.
- Persistent volumes can store the resitration information, but stateful workloads
  are generally more diffcult to manage than stateless workloads.

