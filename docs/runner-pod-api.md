Runner Pod API
==============

* [`GET /deletion_time`](#get-deletion_time)
* [`PUT /deletion_time`](#put-deletion_time)

## `GET /deletion_time`

This API indicates the pod's deletion time.
Used by `Pod` sweeper to check the deletion time of pods.
If the value is zero(`0001-01-01T00: 00: 00Z`), the deletion time is treated as not set.
If the value is other than that, the deletion is judged based on whether
or not the time has passed. 

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

This API updates the pod's deletion time.
Used to update a pod deletion time from Extender in Slack agent.
The time format is RFC 3339 in UTC.

**Successful response**

- HTTP status code: 204 No Content

**Failure responses**

- If the request body is invalid 
  HTTP status code: 400 Bad Request

```console
 curl -s -XPUT localhost:8080/deletion_time -d '
{
	"deletion_time":"0001-01-01T00:00:00Z"
}'
```
