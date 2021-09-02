Runner Pod API
==============

- [`GET /deletion_time`](#get-deletion_time)
- [`PUT /deletion_time`](#put-deletion_time)
- [`GET /job_result`](#get-job_result)

## `GET /deletion_time`

This API returns a pod's deletion time in UTC using RFC 3339 format.

When the pod state is `initializing` or `running`, it returns the zero value of 
type `Time` of Go(`0001-01-01T00:00:00Z`) (cf. [Time.IsZero](https://golang.org/pkg/time/#Time.IsZero)).
When the state is `debugging` or a time is set by the [`PUT /deletion_time`](#put-deletion_time),
it returns a non-zero time.

If the deletion time returned by this API has passed, a controller manager will delete the pod.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Current DeletionTime in JSON

**Failure responses**

- If the deletion time of the pod is incorrect
  HTTP status code: 500 Internal Server Error

```console
$ curl -s -XGET localhost:8080/deletion_time
{
	"deletion_time":"0001-01-01T00:00:00Z"
}
```

## `PUT /deletion_time`

This API updates a pod's deletion time. The time format is RFC 3339 in UTC.

**Successful response**

- HTTP status code: 204 No Content

**Failure responses**

- If the request body is invalid 
  HTTP status code: 400 Bad Request
- If `Content-Type` is not `application/json`
  HTTP status code: 415 Unsupported Media Type

```console
 curl -s -XPUT localhost:8080/deletion_time -H "Content-Type: application/json" -d '
{
	"deletion_time":"0001-01-01T00:00:00Z"
}'
```

## `GET /job_result`

This API returns a pod's job result.

When the pod state is `initializing`, `running` or `stale`, it returns a json contains `status` key with `'unfinished'` as value.
When the pod state is `debugging` (i.e. the pod is finished), it returns a json contains `status` key with one
of `'success', 'failure', 'cancelled', 'unknown'` as value and other job result values.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Current JobResultResponse in JSON

**Failure responses**

- If it fails to get the job information  
500 internal server error in HTTP status code.

```console
$ # When the pod stats is `initializing`, `running` or `stale`.
$ curl -s -XGET localhost:8080/job_result
{
	"status":"unfinished"
}

$ # When the pod state is `debugging`.
$ curl -s -XGET localhost:8080/job_result
{
	"status":"unknown",
	"finished_at":"2021-01-01T00:00:00Z",
	"extend":true,
	"job_info":{
		"actor":"user",
		"git_ref":"branch/name",
		"job_id":"job",
		"repository":"owner/repo",
		"run_id":123456789,
		"run_number":987,
		"workflow_name":"Work flow"
	}
}
```