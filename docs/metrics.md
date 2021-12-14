Metrics
===========

## Controller

Controller provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], it exposes metrics related to controller-runtime and RunnerPools.

| Name                                  | Description                                                        | Type    | Labels                 |
| ------------------------------------- | ------------------------------------------------------------------ | ------- | ---------------------- |
| `meows_runnerpool_secret_retry_count` | The number of times meows retried continuously to get github token | Counter | `runnerpool`           |
| `meows_runnerpool_replicas`           | The number of the RunnerPool replicas.                             | Gauge   | `runnerpool`           |
| `meows_runner_online`                 | 1 if the runner is online.                                         | Gauge   | `runnerpool`, `runner` |
| `meows_runner_busy`                   | 1 if the runner is busy.                                           | Gauge   | `runnerpool`, `runner` |

## Runner Pod

Runner pod provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], it exposes metrics related to the pod.

| Name                               | Description                                                                  | Type    | Labels                |
| ---------------------------------- | ---------------------------------------------------------------------------- | ------- | --------------------- |
| `meows_runner_pod_state`           | 1 if the state of the runner pod is the state specified by the `state` label | Gauge   | `runnerpool`, `state` |
| `meows_runner_listener_exit_state` | Counter for exit codes returned by the `Runner.Listener`                     | Counter | `runnerpool`, `state` |

For more information, see [Design notes | How Runner's state is managed](design.md#how-runners-state-is-managed)

[standard]: https://povilasv.me/prometheus-go-metrics/
