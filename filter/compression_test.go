package filter

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var serverData = []struct {
	path     string
	mime     string
	encoding string
	body     []byte
}{
	{
		"/hello",
		"text/plain",
		"",
		[]byte("hello world"),
	},
	{
		"/hello.gz",
		"text/plain",
		"gzip",
		compress_gzip([]byte("hello world")),
	},
	{
		"/images/face.png",
		"image/png",
		"",
		readfile("../test/images/face.png"),
	},
}

var testData = []struct {
	name string
	// input
	path   string
	accept string
	// output
	encoding     string
	encoded_body []byte
}{
	{
		"no compression",
		"/hello",
		"",
		"",
		[]byte("hello world"),
	},
	{
		"gzip",
		"/hello",
		"gzip",
		"gzip",
		compress_gzip([]byte("hello world")),
	},
	{
		"deflate",
		"/hello",
		"deflate",
		"deflate",
		compress_deflate([]byte("hello world")),
	},
	{
		"preference",
		"/hello",
		"gzip, deflate",
		"gzip",
		compress_gzip([]byte("hello world")),
	},
	{
		"precompressed",
		"/hello.gz",
		"gzip",
		"gzip",
		compress_gzip([]byte("hello world")),
	},
	{
		"image",
		"/images/face.png",
		"gzip",
		"",
		readfile("../test/images/face.png"),
	},
}

func compress_gzip(body []byte) []byte {
	buf := new(bytes.Buffer)
	comp := gzip.NewWriter(buf)
	comp.Write(body)
	comp.Close()
	b := buf.Bytes()
	// fmt.Println(b)
	return b
}

func compress_deflate(body []byte) []byte {
	buf := new(bytes.Buffer)
	comp, err := flate.NewWriter(buf, -1)
	if err != nil {
		panic(fmt.Sprintf("Error using compress/flate.NewWriter() %v", err))
	}
	comp.Write(body)
	comp.Close()
	b := buf.Bytes()
	// fmt.Println(b)
	return b
}

func readfile(path string) []byte {
	if data, err := ioutil.ReadFile(path); err == nil {
		return data
	} else {
		panic(fmt.Sprintf("Error reading file %v: %v", path, err))
	}
	return nil
}

func getCompressionResponse(t *testing.T, path string, accept string) (*Request, *http.Response) {
	rt := http.NewFileTransport(http.Dir("./"))
	r, err := http.NewRequest("GET", path, nil)
	r.Header.Set("Accept-Encoding", accept)
	req := &Request{
		HttpRequest:  r,
		CurrentStage: new(PipelineStageStat),
	}
	if err != nil {
		t.Errorf("Error creating http.Request: %v", err)
	}
	res, err := rt.RoundTrip(r)
	for _, data := range serverData {
		if data.path == path {

			res.Header.Set("Content-Type", data.mime)
			res.Header.Set("Content-Encoding", data.encoding)
			res.Body = (*fixedResBody)(strings.NewReader(string(data.body)))
		}
	}
	if err != nil {
		t.Errorf("bad round trip: %v", err)
	}
	return req, res
}

func TestCompressionFilter2(t *testing.T) {
	filter := NewCompressionFilter(DefaultTypes)
	for _, test := range testData {
		req, res := getCompressionResponse(t, test.path, test.accept)

		filter.FilterResponse(req, res)
		bodyBuf := new(bytes.Buffer)
		io.Copy(bodyBuf, res.Body)
		body := bodyBuf.Bytes()

		if enc := res.Header.Get("Content-Encoding"); enc != test.encoding {
			t.Errorf("%v Header mismatch. Expecting: %v Got: %v", test.name, test.encoding, enc)
		}
		if !bytes.Equal(body, test.encoded_body) {
			t.Errorf("%v Body mismatch.\n\tExpecting:\n\t%v\n\tGot:\n\t%v", test.name, test.encoded_body, body)
		}
	}
}
