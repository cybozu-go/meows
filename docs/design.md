# Design notes

## Motivation

To run GitHub Actions [self-hosted runners](https://docs.github.com/en/actions/hosting-your-own-runners/about-self-hosted-runners)
faster and stably by making full use of idle machine resources.

### Goals

- Deploy and manage GitHub Actions self-hosted runners on multiple servers easily
  by using Kubernetes.
- Enable runners to finish a time-consuming initialization step before jobs are
  assigned in order not to make users wait longer.
- Extend lifetime of runners from outside when jobs are failed, to investigate
  what causes the failure.
- Notify users whether jobs are failed or not via Slack and extend the lifetime
  from Slack.

### Non-Goals

- Autoscaling

## Word Definition

- Workflow: GitHub Actions [workflow](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
  defined in one YAML file as a unit (e.g. `.github/workflows/main.yaml`).
- Job: Job is a user-defined sequence of commands defined under `jobs`.
  A workflow consists of a/some job(s).
- Runner: Machine or container a GitHub Actions workflow runs on. In this document,
  you can read the word "runner" as "self-hosted runner".

## Architecture & Components

This section provides a brief description of meows. First, the architecture diagram is as follows.

As you see, the meows uses two kinds of namespaces.
One is the `meows` namespace, and the other is runner's namespaces.
(As a matter of convenience, I wrote only one runner's namespace in the diagram.
But you can use multiple namespaces as needed.)
The `meows` namespace contains controllers which admin users create.
The runner's namespace contains RunnerPool resources which users create. And some Kubernetes resources which the meows generates are there.

![architecture diagram](./images/architecture.png)

### Kubernetes Custom Resources

The meows provides one Custom Resource.

#### `RunnerPool`

This is a Kubernetes resource for defining the specification of runner pods.
According to the definition of this resource, the meows will create runner pods and register runners.

Users can create RunnerPool resources in any namespaces.

### Kubernetes workloads

The meows consists of three types of Kubernetes workloads.

#### `meows-controller`

A deployment that controls runner pods on a Kubernetes cluster and runners registered to GitHub.

It consists of 3 sub-components.

1. RunnerPool Reconciler
    - A controller for the `RunnerPool` custom resource.
2. Runner manager
    - A goroutine to manage pods and runners.
    - It deletes pods that exceed the deletion time or the recreate deadline.
    - It deletes runners who are offline and do not have a related runner pod.
3. Secret Updater
    - A goroutine to update secret for a registration token of GitHub Actions.

#### `slack-agent`

A deployment for extending the lifetime of runner pods.
With this, you can use Slack to control the pod extension.

It consists of 2 sub-components.

1. Notifier
    - An HTTP server.
    - It accepts requests from the `meows-controller` and sends a message to Slack.
2. Extender
    - A Socket Mode client of the Slack.
    - It watches Slack button events and extends the lifetime of a runner pod by calling the extended deletion time [API](runner-pod-api.md#put-deletion_time) of the pod.

#### Runner pod (Runner deployment)

It is a pod (a deployment) to run GitHub Actions self-hosted runner on.
On this, the [GitHub Actions Runner](https://github.com/actions/runner) will run under our agent program (`endpoint`) controls.

## Operation

### How GitHub Actions schedules jobs on self-hosted runner

GitHub Actions schedules jobs on runners in the ways written in this section.

#### How runner is registered

The following steps are needed to register a runner on GitHub Actions.

1. Fetch a registration token via GitHub Actions [API](https://docs.github.com/en/rest/reference/actions#create-a-registration-token-for-a-repository).
1. Execute [`config.sh`](https://github.com/actions/runner/blob/main/src/Misc/layoutroot/config.sh)
  to configure a runner.
1. Execute [`Runner.Listener`](https://github.com/actions/runner/blob/main/src/Runner.Listener/Program.cs) to start the long polling for GitHub Actions API.

`Runner.Listener` start a long polling in the end, and `cmd/entrypoint/cmd/root.go#runService`
handles some errors and then restarts the `Runner.Listener` automatically for upgrade themselves.
They upgrade the binary by themselves when a new release is out. This help
us avoid unnecessary `Pod` recreation.

#### `Runner.Listener` should be executed right after `config.sh`

We should execute `Runner.Listener` within about 30 seconds after executing `config.sh`.
After that, `Runner.Listener` fails to open a connection with GitHub Actions
API.  Note that this behavior is not clearly written in the official documentation
and might change unexpectedly.

#### How runner state is managed on GitHub Actions API

Runner has the `status` and `busy` state as written [here](https://docs.github.com/en/rest/reference/actions#get-a-self-hosted-runner-for-a-repository).

- `status`:
  - `online`: The runner is running a long polling.
  - `offline`: The runner is NOT running a long polling.
- `busy`:
  - `true`: The runner is running a workflow.
  - `false`: The runner is NOT running a workflow.

If the `--ephemeral` option is given to `config.sh` does not repeat the
long polling again, and never gets `online` after the assigned job is done.
This behavior is useful for ensuring to make a clean environment for each job.
ref: https://docs.github.com/en/actions/hosting-your-own-runners/autoscaling-with-self-hosted-runners#using-ephemeral-runners-for-autoscaling

#### A job is scheduled only on a `online` runner

Some experiments reveal the following behaviors.
Note that this behavior is not clearly written in the official documentation
and might change unexpectedly.

- If there is no `online` runners at the time a job is created, the job is not
  scheduled on any runner.
- If there is any `online` and non-`busy` runner at the time a job is created,
  the job is scheduled one of the runners.
- If all the runners are `online` and `busy` at the time a job is created, the
  job is queued first and then scheduled right after any runner finishes its job
  and gets non-`busy`.
- If all the runners are `online` and `busy` at the time a job is created and
  then a runner is created before any existing runner finishes its job, the job
  is scheduled on the newly created runner.
- Two identical runners registered with the same name are recognized as the same
  runner. If one runner dies `offline` holding a unprocessed job and another
  runner is created with the same name, the new runner starts the job once it
  gets `online`.

### Runners can have multiple custom labels

The [custom label](https://docs.github.com/en/actions/hosting-your-own-runners/using-labels-with-self-hosted-runners)
is a label to route jobs to specific types of runners. Users usually use
self-hosted runners by setting `self-hosted` to `runs-on` in a workflow
definition. If custom labels are given, users are allowed to set one of the
custom label value to `runs-on`. This is useful when you want to use multiple
types of runners, for example, `highmem`and `highcpu`.

meows sets the namespaced name of a `RunnerPool` as a custom label.

### How self-hosted runners are created and runs jobs

1. The `RunnerPool` reconciler watches `RunnerPool` creation events and creates
  a `Deployment` and empty `Secret` which own runner `Pod`s. The controller adds
  some labels to `Pod`s to tell other components that the `Pod`s are managed by
  the controller. After these steps, Other components recognize which `Pod`s
  they should handle with those labels.
1. The secret updater get a registration token via GitHub API and update the initially created empty `Secret`.
   After that, the secret updater will automatically update the `Secret` based on the expiration date of the registration token.
1. Each runner `Pod` does the following steps.
   1. Register itself as a self-hosted runner with the injected token.
   1. Initialize runner environment by doing the user-defined process.
   1. Start a long polling process and wait for GitHub Actions to assign a job.
   1. Run an assigned job.
   1. Call the Slack agent to notify users. GitHub API does not seem to provide
      a way to know which runner ran a succeeded or failed job. So, this repository
      provides a simple `job-failed` command, and asks users to execute this
      command when the job is failed.  The `if: failure()` syntax allows users
      to run the step only when one of previous steps exit with non-zero code.
   1. Publish the timestamp of when to delete this pod in the `/deletion_time` endpoint.
      If the job is succeeded or canceled, the `Pod` publishes the current time for 
      delete itself. If the job is failed, the `Pod` publishes the future time for
      delete itself, for example 20 min later.
1. The Slack agent notifies the result of the job on a Slack channel.
1. Users can extend the failed runner if they want to by clicking a button on Slack.
1. The Slack agent is running a WebSocket process to watch extending messages
  from Slack. If it receives a message, it requests the `Pod` to update the
  designated time.
1. The Runner manager periodically checks if there are `Pod`s past a deletion time and
   if any, it deletes `Pod`s.

### How Runner's state is managed

A Runner `Pod` has the following state as a GitHub Actions job runner.

- `initializing`: `Pod` initializing. Prepare the necessary environment for Job.
    for example, booting a couple of VMs needed in a job before the job is assigned.
- `running`: `Pod` is running. Registered in GitHub Actions.
- `debugging`: The job has finished with failure and Users can enter `Pod` to debug.
- `stale`: The environment in the `Pod` is dirty. If a runner restarts before completing a job, 
    the environment in the `Pod` may be dirty. This state means waiting for the Pod
    to be removed to prevent Job execution with that stale Pod.

In addition, it has the following states as the exit state of the execution result of `Runner.Listener`.

- `retryable_error`: If execution fails due to a factor other than a job, restart `Runner.Listener`.
- `updating`: When a new `Runner.Listener` is released, it updates itself and restarts` Runner.Listener`.
- `undefined`: When the exit code of `Runner.Listener` is undefined. It restarts` Runner.Listener`.

The above states are exposed from `/metrics` endpoint as Prometheus metrics. See [metrics.md](metrics.md).

Detailed `running` state of the runner as seen on GitHub is not provided
in the `/metrics` endpoint of the runner `Pod`.
Because those detailed states are going to be provided metrics by controller based
on the [state](design.md#how-runner-state-is-managed-on-github-actions-api) that
controller can get from GitHubActionsAPI.
