package cloudwatch

import (
	"testing"

	"github.com/egnyte/ax/pkg/backend/common"
)

func TestParsing(t *testing.T) {
	msg := attemptParseJSON(`2017-09-27T09:01:01.245468966Z {"asctime": "2017-09-27 09:01:01,245", "created": 1506502861.2452097, "filename": "connectionpool.py", "funcName": "_make_request", "levelname": "DEBUG", "levelno": 10, "module": "connectionpool", "msecs": 245.2096939086914, "message": "http://localhost:None \"POST /v1.29/exec/1744fb9d8aa1ed1f94f729d4e0474251dfab9e0523385d42e77ea10acda53957/start HTTP/1.1\" 101 0", "name": "urllib3.connectionpool", "pathname": "/usr/local/lib/python3.6/site-packages/urllib3/connectionpool.py", "process": 5, "processName": "MainProcess", "relativeCreated": 2276.298999786377, "thread": 140018892404480, "threadName": "MainThread", "turbo_request_id": null, "user": null, "tid": 5, "source": "/usr/local/lib/python3.6/site-packages/urllib3/connectionpool.py:395", "client_id": null}
		`)
	if msg["filename"] != "connectionpool.py" {
		t.Errorf("Parsed: %+v", msg)
		t.Fail()
	}
	msg = attemptParseJSON(`{"asctime": "2017-09-27 09:01:01,245", "created": 1506502861.2452097, "filename": "connectionpool.py", "funcName": "_make_request", "levelname": "DEBUG", "levelno": 10, "module": "connectionpool", "msecs": 245.2096939086914, "message": "http://localhost:None \"POST /v1.29/exec/1744fb9d8aa1ed1f94f729d4e0474251dfab9e0523385d42e77ea10acda53957/start HTTP/1.1\" 101 0", "name": "urllib3.connectionpool", "pathname": "/usr/local/lib/python3.6/site-packages/urllib3/connectionpool.py", "process": 5, "processName": "MainProcess", "relativeCreated": 2276.298999786377, "thread": 140018892404480, "threadName": "MainThread", "turbo_request_id": null, "user": null, "tid": 5, "source": "/usr/local/lib/python3.6/site-packages/urllib3/connectionpool.py:395", "client_id": null}
		`)
	if msg["filename"] != "connectionpool.py" {
		t.Errorf("Parsed: %+v", msg)
		t.Fail()
	}
}

func TestFilterGenerator(t *testing.T) {
	output := queryToFilterPattern(common.Query{
		QueryString: "Test",
		Filters: []common.QueryFilter{
			{
				FieldName: "name",
				Operator:  "=",
				Value:     "zef",
			},
		},
	})
	if output != `Test { ($.name = "zef") }` {
		t.Fatal(output)
	}

	output = queryToFilterPattern(common.Query{
		Filters: []common.QueryFilter{
			{
				FieldName: "name",
				Operator:  "=",
				Value:     "zef",
			},
		},
	})
	if output != `{ ($.name = "zef") }` {
		t.Fatal(output)
	}

	output = queryToFilterPattern(common.Query{
		Filters: []common.QueryFilter{
			{
				FieldName: "name",
				Operator:  "=",
				Value:     "zef",
			},
			{
				FieldName: "age",
				Operator:  "=",
				Value:     "34",
			},
		},
	})
	if output != `{ ($.name = "zef") && ($.age = "34") }` {
		t.Fatal(output)
	}
}
