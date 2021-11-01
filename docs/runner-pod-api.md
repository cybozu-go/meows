Runner Pod API
==============

- [`PUT /deletion_time`](#put-deletion_time)
- [`GET /status`](#get-status)

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
	"deletion_time": "0001-01-01T00:00:00Z"
}'
```

## `GET /status`

This API returns a pod's status.

When the pod state is `initializing`, `running` or `stale`, it returns a json contains only `state` key with the state as value.
When the pod state is `debugging` (i.e. the pod is finished), it returns a json contains several other fields besides `status` key.

**Successful response**

- HTTP status code: 200 OK
- HTTP response header: `Content-Type: application/json`
- HTTP response body: Current JobResultResponse in JSON

**Failure responses**

- If it fails to get the job information  
HTTP status code: 500 Internal Server Error

```console
$ # When the pod state is `initializing`, `running` or `stale`:
$ curl -s -XGET localhost:8080/status
{
	"state": "initializing" ... "initializing", "running" or "stale"
}

$ # When the pod state is `debugging`:
$ curl -s -XGET localhost:8080/status
{
	"state": "debugging",
	"result": "failure",  ... Job result. "success", "failure, "cancelled" or "unknown".
	"finished_at": "2021-01-01T00:00:00Z", ... The time the job was finished.
	"deletion_time": "2021-01-01T00:20:00Z", ... Scheduled deletion time. When the "extend" is false, this value is the same as "finished_at".
	"extend": true, ... Pod extension is required or not.
	"job_info": {
		"actor": "user",
		"git_ref": "branch/name",
		"job_id": "job",
		"repository": "owner/repo",
		"run_id": 123456789,
		"run_number": 987,
		"workflow_name": "Work flow"
	},
	"slack_channel": "" ... May be blank. The name of the Slack channel specified in the workflow.
}
```
