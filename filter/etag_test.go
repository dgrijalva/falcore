package filter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
)

var etagServerData = []struct {
	path   string
	status int
	etag   string
	body   []byte
}{
	{
		"/hello",
		200,
		"abc123",
		[]byte("hello world"),
	},
	{
		"/pre",
		304,
		"abc123",
		[]byte{},
	},
}

var etagTestData = []struct {
	name string
	// input
	path string
	etag string
	// output
	status int
	body   []byte
}{
	{
		"no etag",
		"/hello",
		"",
		200,
		[]byte("hello world"),
	},
	{
		"match",
		"/hello",
		"abc123",
		304,
		[]byte{},
	},
	{
		"pre-filtered",
		"/pre",
		"abc123",
		304,
		[]byte{},
	},
}

func getEtagResponse(t *testing.T, path string, etag string) (*Request, *http.Response) {
	r, err := http.NewRequest("GET", path, nil)
	r.Header.Set("If-None-Match", etag)
	req := &Request{
		HttpRequest:  r,
		CurrentStage: new(PipelineStageStat),
	}
	if err != nil {
		t.Errorf("Error creating http.Request: %v", err)
	}
	var res *http.Response
	for _, data := range etagServerData {
		if data.path == path {
			res = SimpleResponse(r, data.status, make(http.Header), string(data.body))
			res.Header.Set("Etag", data.etag)
			return req, res
		}
	}

	panic(fmt.Sprintf("req: %v, res: %v", req, res))
	return req, nil
}

func TestEtagFilter(t *testing.T) {
	filter := new(EtagFilter)
	for _, test := range etagTestData {
		req, res := getEtagResponse(t, test.path, test.etag)

		filter.FilterResponse(req, res)

		if st := res.StatusCode; st != test.status {
			t.Errorf("%v StatusCode mismatch. Expecting: %v Got: %v", test.name, test.status, st)
		}
		if res.StatusCode == 200 {
			bodyBuf := new(bytes.Buffer)
			io.Copy(bodyBuf, res.Body)
			body := bodyBuf.Bytes()
			if !bytes.Equal(body, test.body) {
				t.Errorf("%v Body mismatch.\n\tExpecting:\n\t%v\n\tGot:\n\t%v", test.name, test.body, body)
			}
		}
	}
}
