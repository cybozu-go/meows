Metrics
===========

## Runner Pod

Runner `Pod` provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], runner-pool.

All these metrics are prefixed wit `runner_` and have `runner_pool` labels.

| Name                  | Description                                                                  | Type    | Labels  |
| --------------------- | ---------------------------------------------------------------------------- | ------- | ------- |
| `pod_state`           | 1 if the state of the runner pod is the state specified by the `state` label | Gauge   | 'state' |
| `listener_exit_state` | Counter for exit codes returned by the `Runner.Listener`                     | Counter | 'state' |


For more information, see [Design notes | How Runner's state is managed](design.md#how-runners-state-is-managed)

[standard]: https://povilasv.me/prometheus-go-metrics/
