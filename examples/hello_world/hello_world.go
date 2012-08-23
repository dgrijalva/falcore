package main

import (
	"flag"
	"fmt"
	"github.com/ngmoco/falcore"
	"github.com/ngmoco/falcore/filter"
	"net/http"
)

// Command line options
var (
	port = flag.Int("port", 8000, "the port to listen on")
)

func main() {
	// parse command line options
	flag.Parse()

	// setup pipeline
	pipeline := falcore.NewPipeline()

	// set filter logger
	filter.SetLogger(falcore.GetLogger())

	// upstream
	pipeline.Upstream.PushBack(helloFilter)

	// setup server
	server := falcore.NewServer(*port, pipeline)

	// start the server
	// this is normally blocking forever unless you send lifecycle commands 
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Could not start server:", err)
	}
}

var helloFilter = filter.NewRequestFilter(func(req *filter.Request) *http.Response {
	return filter.SimpleResponse(req.HttpRequest, 200, nil, "hello world!")
})
