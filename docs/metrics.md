# Metrics

## Controller

Controller provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], it exposes metrics related to controller-runtime and RunnerPools.

| Name                                    | Description                                                        | Type      | Labels                 |
| --------------------------------------- | ------------------------------------------------------------------ | --------- | ---------------------- |
| `meows_runnerpool_secret_retry_count`   | The number of times meows retried continuously to get github token | Counter   | `runnerpool`           |
| `meows_runnerpool_replicas`             | The number of the RunnerPool replicas.                             | Gauge     | `runnerpool`           |
| `meows_runner_online`                   | 1 if the runner is online.                                         | Gauge     | `runnerpool`, `runner` |
| `meows_runner_busy`                     | 1 if the runner is busy.                                           | Gauge     | `runnerpool`, `runner` |
| `meows_runnerpool_job_results_total`    | The number of finished jobs by result.                             | Counter   | `runnerpool`, `result` |
| `meows_runnerpool_job_duration_seconds` | The duration of finished jobs.                                     | Histogram | `runnerpool`, `result` |

The controller records the result of a job when it finds a runner pod in the `debugging` state, once per pod.
`result` is one of `success`, `failure`, `cancelled` and `unknown`.
The series for all results are exposed with zero values when the controller starts managing the RunnerPool.
The duration is measured from the time when `job-started` was called to the end of the job,
so it is not recorded for jobs that did not call `job-started`.
Jobs that finished before the controller started are not recorded.

## Runner Pod

Runner pod provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], it exposes metrics related to the pod.

| Name                               | Description                                                                  | Type    | Labels                                                                     |
| ---------------------------------- | ---------------------------------------------------------------------------- | ------- | -------------------------------------------------------------------------- |
| `meows_runner_pod_state`           | 1 if the state of the runner pod is the state specified by the `state` label | Gauge   | `runnerpool`, `state`                                                      |
| `meows_runner_listener_exit_state` | Counter for exit codes returned by the `Runner.Listener`                     | Counter | `runnerpool`, `state`                                                      |
| `meows_runner_job_info`            | Information about the job that the runner is running. The value is always 1. | Gauge   | `runnerpool`, `repository`, `workflow`, `job_id`, `run_id`, `pull_request` |

`meows_runner_job_info` is exposed after `job-started` is called in the job.
`pull_request` is empty if the job was not triggered by a pull request.
Because the job itself writes the job information, the runner does not trust it:
the metric is not exposed if `run_id` is not a positive integer,
and string label values are truncated to 128 characters.
The branch name and the actor are not exposed because metrics are often visible to more people than the repository is.

For more information, see [Design notes | How Runner's state is managed](design.md#how-runners-state-is-managed)

[standard]: https://povilasv.me/prometheus-go-metrics/
