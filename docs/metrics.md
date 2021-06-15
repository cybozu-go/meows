Metrics
===========

## runner-pool

[`runner-pool`](crd-runner-pool.md) provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], runner-pool.

All these metrics are prefixed wit `runner_` and have `runner_pool_name` labels.

| Name     | label                | Description                                                                                     | Type  |
| -------- | -------------------- | ----------------------------------------------------------------------------------------------- | ----- |
| `pod`    |                      | `status` label show the status of the pod.                                                      | Gauge |
| `pod`    | status=`starting`    | 1 if the pod is starting, 0 otherwise.                                                          | Gauge |
| `pod`    | status=`registering` | 1 if the pod is registering on GitHubActions, 0 otherwise.                                      | Gauge |
| `pod`    | status=`deleting`    | 1 if the pod is waiting for deleting, 0 otherwise.                                              | Gauge |
| `job`    |                      | `status` label show the status of the job.                                                      | Gauge |
| `job`    | status=`waiting`     | 1 if the job is waiting until `job-started` is called after the pod is registered, 0 otherwise. | Gauge |
| `job`    | status=`running`     | 1 if the job is running, 0 otherwise.                                                           | Gauge |
| `job`    | status=`completed`   | 1 if the job is completed, 0 otherwise.                                                         | Gauge |
| `result` |                      | `result` label show the result of the job.                                                      | Gauge |
| `result` | status=`success`     | 1 if the job result is success, 0 otherwise.                                                    | Gauge |
| `result` | status=`failure`     | 1 if the job result is failure, 0 otherwise.                                                    | Gauge |
| `result` | status=`cancelled`   | 1 if the job result is cancelled, 0 otherwise.                                                  | Gauge |
| `result` | status=`unknown`     | 1 if the job result is unknown, 0 otherwise.                                                    | Gauge |
