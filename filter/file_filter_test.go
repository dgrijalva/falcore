package filter

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// var srv *falcore.Server

func init() {
	// Silence log output
	log.SetOutput(nil)

	// setup mime
	mime.AddExtensionType(".foo", "foo/bar")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".txt", "text/plain")
	mime.AddExtensionType(".png", "image/png")

}

func get(p string) (r *http.Response, err error) {
	req, _ := http.NewRequest("GET", p, nil)
	rt := http.NewFileTransport(http.Dir("/"))
	return rt.RoundTrip(req)
}

var fourOhFourTests = []struct {
	name string
	url  string
}{
	{
		name: "basic invalid path",
		url:  "/this/path/doesnt/exist",
	},
	{
		name: "realtive pathing out of sandbox",
		url:  "/../README.md",
	},
	{
		name: "directory",
		url:  "/hello",
	},
}

func TestFourOhFour(t *testing.T) {
	for _, test := range fourOhFourTests {
		r, err := get(test.url)
		if err != nil {
			t.Errorf("%v Error getting file:", test.name, err)
			continue
		}
		if r.StatusCode != 404 {
			t.Errorf("%v Expected status 404, got %v", test.name, r.StatusCode)
		}
	}
}

var basicTests = []struct {
	name string
	path string
	mime string
	data []byte
	file string
	url  string
}{
	{
		name: "small text file",
		mime: "text/plain",
		path: "fsbase_test/hello/world.txt",
		data: []byte("Hello world!"),
		url:  "/hello/world.txt",
	},
	{
		name: "json file",
		mime: "application/json",
		path: "fsbase_test/foo.json",
		file: "../test/foo.json",
		url:  "/foo.json",
	},
	{
		name: "png file",
		mime: "image/png",
		path: "fsbase_test/images/face.png",
		file: "../test/images/face.png",
		url:  "/images/face.png",
	},
	{
		name: "relative paths",
		mime: "application/json",
		path: "fsbase_test/foo.json",
		file: "../test/foo.json",
		url:  "/images/../foo.json",
	},
	{
		name: "custom mime type",
		mime: "foo/bar",
		path: "fsbase_test/custom_type.foo",
		file: "../test/custom_type.foo",
		url:  "/custom_type.foo",
	},
}

func TestBasicFiles(t *testing.T) {
	rbody := new(bytes.Buffer)
	path := filepath.SplitList(os.Getenv("GOPATH"))
	p := filepath.Join(path[0], "/src/github.com/ngmoco/falcore/test")
	filter := &FileFilter{p, ""}
	for _, test := range basicTests {
		// read in test file data
		if test.file != "" {
			test.data, _ = ioutil.ReadFile(test.file)
		}

		req, err := http.NewRequest("GET", test.url, nil)
		if err != nil {
			t.Errorf("error creating request: %v", err)
			continue
		}
		re := new(Request)
		re.HttpRequest = req
		r := filter.FilterRequest(re)
		if r.StatusCode != 200 {
			t.Errorf("%v Expected status 200, got %v", test.name, r.StatusCode)
			continue
		}
		if strings.Split(r.Header.Get("Content-Type"), ";")[0] != test.mime {
			t.Errorf("%v Expected Content-Type: %v, got '%v'", test.name, test.mime, r.Header.Get("Content-Type"))
		}
		rbody.Reset()
		io.Copy(rbody, r.Body)
		if rbytes := rbody.Bytes(); !bytes.Equal(test.data, rbytes) {
			t.Errorf("%v Body doesn't match.\n\tExpected:\n\t%v\n\tReceived:\n\t%v", test.name, test.data, rbytes)
		}
	}
}
