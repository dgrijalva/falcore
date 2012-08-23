package falcore

import (
	"fmt"
	"github.com/ngmoco/falcore/filter"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"testing"
	"time"
)

var (
	srv       *Server
	testReqs  []*http.Request
	client    *http.Client
	testFiles = map[string]string{
		"100":  "100.html",
		"512":  "512.html",
		"1024": "1024.html",
		"4k":   "4k.html",
		"20k":  "20k.html",
		"1MB":  "1MB.html",
	}
	port      int
	KeepAlive bool = true
)

func init() {
	if srv == nil {
		port = 8080
		stop := make(chan int)
		srv := NewServer(port, NewPipeline())
		runServer(srv, stop)
		<-srv.AcceptReady
	}
}

func BenchmarkServer(b *testing.B) {

	KeepAlive = false
	benchmarkFileServer(b, "")
}

func BenchmarkServerFile100bytes(b *testing.B) {
	KeepAlive = false
	benchmarkFileServer(b, testFiles["100"])
}
func BenchmarkServerFile4k(b *testing.B) {
	KeepAlive = false
	benchmarkFileServer(b, testFiles["4k"])
}
func BenchmarkServerFile20k(b *testing.B) {
	KeepAlive = false
	benchmarkFileServer(b, testFiles["20k"])
}
func BenchmarkServerFile1MB(b *testing.B) {
	KeepAlive = false
	benchmarkFileServer(b, testFiles["1MB"])
}

func BenchmarkServerKA(b *testing.B) {
	benchmarkFileServer(b, "")
}

func BenchmarkServerFile100bytesKA(b *testing.B) {
	benchmarkFileServer(b, testFiles["100"])
}
func BenchmarkServerFile4kKA(b *testing.B) {
	benchmarkFileServer(b, testFiles["4k"])
}
func BenchmarkServerFile20kKA(b *testing.B) {
	benchmarkFileServer(b, testFiles["20k"])
}
func BenchmarkServerFile1MBKA(b *testing.B) {
	benchmarkFileServer(b, testFiles["1MB"])
}

func runServer(srv *Server, stop chan int) {
/*	f := filter.NewRequestFilter(func(req *filter.Request) *http.Response {
		return filter.SimpleResponse(req.HttpRequest, 200, nil, "hello world!")
	})
	srv.Pipeline.Upstream.PushFront(f)*/

	srv.Pipeline.Upstream.PushBack(&filter.FileFilter{
		PathPrefix: "/",
		BasePath:   "./test/",
	})

	srv.Addr = fmt.Sprintf("127.0.0.1:%d", port)
	li, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(fmt.Sprintf("could not listen to address: %v", err))
	}
	l, ok := li.(*net.TCPListener)
	if !ok {
		panic("can't create listener")
	}
	l.SetDeadline(time.Now().Add(3e9))
	srv.listener = l

	go srv.serve()
}

func connect(b *testing.B) *httputil.ClientConn {
	con, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	//defer con.Close()
	if err != nil {
		b.Fatalf("could not open connection: %v", err)
	}
	c := httputil.NewClientConn(con, nil)
	return c
}

func benchmarkFileServer(b *testing.B, file string) {
	b.StopTimer()
	path := "/"
	if file != "" {
		path = fmt.Sprintf("/bench/%s", file)
	}
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		b.Fatalf("could not create request: $v", err)
	}
	if KeepAlive {
		req.Header.Add("Connection", "keep-alive")
	}

	var failedCon int = 0
	var successCon int = 0
	b.StartTimer()

	var wg sync.WaitGroup
	// Don't really want to use many OS threads here. We're reallly more
	// interested in how fast the server loop can process the requests so better
	// not to create too much contention. You could crank up the value of X if
	// you have a really beefy box.
	x := 1
	wg.Add(x)
	for j := 0; j < x; j++ {
		go func() {
			c := connect(b)
			for i := 0; i < b.N; i++ {
				if !KeepAlive {
					c = connect(b)
				}
				res, err := c.Do(req)

				switch {
				case (err == httputil.ErrPersistEOF) || (err == httputil.ErrClosed):
					failedCon++
					c.Close()
					continue
				case err != nil:
					Debug("Unexpected error: %v", err)
					c.Close()
					failedCon++
					continue
				default:
					_, err := ioutil.ReadAll(res.Body)
					if err != nil {
						b.Logf("ReadAll: %v", err)
						continue
					}
				/*	body := string(all)
					if body != "hello world!" {
						panic("Got body: " + body)
					}*/
				}
				successCon++
				if !KeepAlive {
					c.Close()
				}
			}
			wg.Done()
			if failedCon > 0 {
				b.Logf("Failed connections: %d", failedCon)
			}
		}()
	}
	wg.Wait()
}
