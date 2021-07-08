Runner Pod API
==============

* [GET /deletion_time](#getdeletiontime)
* [PUT /deletion_time](#putdeletiontime)

## <a name="getdeletiontime" />`GET /deletion_time`

Used by Podsweeper to check the deletion time of pods.
Not set if the deletion_time value is zero (UTC time "0001-01-01T00: 00: 00Z" in RFC3339 format).
When the value is other than that, the deletion is judged based on whether
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

## <a name="putdeletiontime" />`PUT /deletion_time`

Used to update the pod deletion time from slack-agent. The time format is RFC 3339
for UTC time.

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
