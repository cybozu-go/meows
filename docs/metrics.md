Metrics
===========

## runner-pool

[`runner-pool`](crd-runner-pool.md) provides the following kind of metrics in Prometheus format.
Aside from [the standard Go runtime and process metrics][standard], runner-pool.

All these metrics are prefixed wit `runner_` and have `runner_pool_name` labels.

| Name         | label                 | Description                                                                                       | Type  |
| ------------ | --------------------- | ------------------------------------------------------------------------------------------------- | ----- |
| `pod_status` |                       | `status` label show the status of the pod.                                                        | Gauge |
| `pod_status` | status=`initializing` | 1 if the pod is initializing, 0 otherwise.                                                        | Gauge |
| `pod_status` | status=`running`      | 1 if the pod is running on GitHubActions, 0 otherwise.                                            | Gauge |
| `pod_status` | status=`debugging`    | 1 if the pod is debugging until deleted, 0 otherwise.                                             | Gauge |
| `job_status` |                       | `status` label show the status of the job.                                                        | Gauge |
| `job_status` | status=`listening`    | 1 if the job is listening until `job-started` is called after the pod is registered, 0 otherwise. | Gauge |
| `job_status` | status=`assigned`     | 1 if the job is assigned, 0 otherwise.                                                            | Gauge |
| `job_status` | status=`finished`     | 1 if the job is finished, 0 otherwise.                                                            | Gauge |
| `result`     |                       | `result` label show the result of the job.                                                        | Gauge |
| `result`     | status=`success`      | 1 if the job result is success, 0 otherwise.                                                      | Gauge |
| `result`     | status=`failure`      | 1 if the job result is failure, 0 otherwise.                                                      | Gauge |
| `result`     | status=`cancelled`    | 1 if the job result is cancelled, 0 otherwise.                                                    | Gauge |
| `result`     | status=`unknown`      | 1 if the job result is unknown, 0 otherwise.                                                      | Gauge |
