Metrics
===========

## Runner Pod

Runner `Pod` provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], runner-pool.

All these metrics are prefixed wit `runner_` and have `runner_pool_name` labels.

| Name         | Description                                                                  | Type  | Labels   |
| ------------ | ---------------------------------------------------------------------------- | ----- | -------- |
| `pod_state`  | 1 if the state of the runner pod is the state specified by the `state` label | Gauge | 'state'  |
| `job_result` | 1 if the result of the job is the result specified by the `result` label     | Gauge | 'result' |

For more information, see [Design notes | Exposing Runner's the state as Prometheus metrics](design.md#exposing-runners-the-state-as-prometheus-metrics)

[standard]: https://povilasv.me/prometheus-go-metrics/
