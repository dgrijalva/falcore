package main

import (
	"http"
	"flag"
	"fmt"
	"rand"

	"falcore"
)

// Command line options
var (
	port = flag.Int("port", 8000, "the port to listen on")
)

type Greeting struct {
	ID   int
	Text string
}

var Greetings = []Greeting{
	{0, "hello world!"},
	{1, "how are you gentlement?  all your base are belong to us"},
	{2, "It's a traaaap!"},
	{3, "Bonjour!"},
}

var GreetingRenderer falcore.Renderer


func main() {
	// parse command line options
	flag.Parse()

	// setup pipeline
	pipeline := falcore.NewPipeline()

	// upstream
	pipeline.Upstream.PushBack(greetingFilter)

	// setup server
	server := falcore.NewServer(*port, pipeline)

	// set up the renderer
	htmlRenderer, _ := falcore.NewTemplateRenderer(`<html><body><h1>{{.Text}}</h1></body></html>`, "text/html")
	textRenderer, _ := falcore.NewTemplateRenderer("{{.Text}}", "text/plain")

	GreetingRenderer = falcore.NewFormatSelector(htmlRenderer)
	GreetingRenderer.(*falcore.FormatSelector).Add("text/html", htmlRenderer)
	GreetingRenderer.(*falcore.FormatSelector).Add("text/plain", textRenderer)
	GreetingRenderer.(*falcore.FormatSelector).Add("application/json", &falcore.JSONRenderer{})
	

	// start the server
	// this is normally blocking forever unless you send lifecycle commands 
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Could not start server:", err)
	}
}

var greetingFilter = falcore.NewRequestFilter(func(req *falcore.Request) *http.Response {
	if res, err := falcore.RenderResponse(req, 200, nil, GreetingRenderer, Greetings[rand.Intn(len(Greetings))]); err == nil {
		return res
	}
	return falcore.SimpleResponse(req.HttpRequest, 500, nil, "server error")
})
