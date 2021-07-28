Runner Pod API
==============

- [`GET /deletion_time`](#get-deletion_time)
- [`PUT /deletion_time`](#put-deletion_time)

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
