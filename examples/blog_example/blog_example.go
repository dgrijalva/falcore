package main

import (
	"fmt"
	"github.com/ngmoco/falcore"
	"github.com/ngmoco/falcore/filter"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	// create pipeline
	pipeline := falcore.NewPipeline()

	// set filter logger
	filter.SetLogger(falcore.GetLogger())

	// add upstream pipeline stages
	var filter1 delayFilter
	pipeline.Upstream.PushBack(filter1)
	var filter2 helloFilter
	pipeline.Upstream.PushBack(filter2)

	// add request done callback stage
	pipeline.RequestDoneCallback = reqCB

	// create server on port 8000
	server := falcore.NewServer(8000, pipeline)

	// start the server
	// this is normally blocking forever unless you send lifecycle commands 
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Could not start server:", err)
	}
}

// Example filter to show timing features
type delayFilter int

func (f delayFilter) FilterRequest(req *filter.Request) *http.Response {
	status := rand.Intn(2) // random status 0 or 1
	if status == 0 {
		time.Sleep(time.Duration(rand.Int63n(100e6))) // random sleep between 0 and 100 ms
	}
	req.CurrentStage.Status = byte(status) // set the status to produce a unique signature
	return nil
}

// Example filter that returns a Response
type helloFilter int

func (f helloFilter) FilterRequest(req *filter.Request) *http.Response {
	return filter.SimpleResponse(req.HttpRequest, 200, nil, "hello world!\n")
}

var reqCB = filter.NewRequestFilter(func(req *filter.Request) *http.Response {
	req.Trace() // Prints detailed stats about the request to the log
	return nil
})
