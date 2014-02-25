package responder

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

func pipeBench(b *testing.B, f func() (io.WriteCloser, *http.Response), size int) {
	data := make([]byte, 1024*1024)
	for i := 0; i < b.N; i++ {
		wr, res := f()
		go func() {
			for j := 0; j < len(data); {
				jj, _ := wr.Write(data[:size])
				j += jj
			}
			wr.Close()
		}()
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}
}

var pipeBenchRequest, _ = http.NewRequest("GET", "/foo", nil)

func pipeRes() (io.WriteCloser, *http.Response) {
	return PipeResponse(pipeBenchRequest, 200, nil)
}

func buffPipeRes() (io.WriteCloser, *http.Response) {
	return BufferedPipeResponse(pipeBenchRequest, 200, nil)
}

func BenchmarkPipeResponse128(b *testing.B) {
	pipeBench(b, pipeRes, 128)
}

func BenchmarkPipeResponse512(b *testing.B) {
	pipeBench(b, pipeRes, 512)
}

func BenchmarkPipeResponse1024(b *testing.B) {
	pipeBench(b, pipeRes, 1024)
}

func BenchmarkPipeResponse3K(b *testing.B) {
	pipeBench(b, pipeRes, 1024*3)
}

func BenchmarkPipeResponse500K(b *testing.B) {
	pipeBench(b, pipeRes, 1024*512)
}

func BenchmarkPipeResponse1MB(b *testing.B) {
	pipeBench(b, pipeRes, 1024*1024)
}

func BenchmarkBufferedPipeResponse128(b *testing.B) {
	pipeBench(b, buffPipeRes, 128)
}

func BenchmarkBufferedPipeResponse512(b *testing.B) {
	pipeBench(b, buffPipeRes, 512)
}

func BenchmarkBufferedPipeResponse1024(b *testing.B) {
	pipeBench(b, buffPipeRes, 1024)
}

func BenchmarkBufferedPipeResponse3K(b *testing.B) {
	pipeBench(b, buffPipeRes, 1024*3)
}

func BenchmarkBufferedPipeResponse500K(b *testing.B) {
	pipeBench(b, buffPipeRes, 1024*512)
}

func BenchmarkBufferedPipeResponse1MB(b *testing.B) {
	pipeBench(b, buffPipeRes, 1024*1024)
}

func TestPipeResponse(t *testing.T) {
	req, _ := http.NewRequest("GET", "/foo", nil)
	data := make([]byte, 1e7)
	wr, res := PipeResponse(req, 200, nil)
	go func() {
		io.Copy(wr, bytes.NewBuffer(data))
		wr.Close()
	}()
	i, _ := io.Copy(ioutil.Discard, res.Body)
	if len(data) != int(i) {
		t.Errorf("Content length doesn't match")
	}
	res.Body.Close()
}

func TestBufferedPipeResponse(t *testing.T) {
	req, _ := http.NewRequest("GET", "/foo", nil)
	data := make([]byte, 1e7)
	wr, res := BufferedPipeResponse(req, 200, nil)
	go func() {
		io.Copy(wr, bytes.NewBuffer(data))
		wr.Close()
	}()
	i, _ := io.Copy(ioutil.Discard, res.Body)
	if len(data) != int(i) {
		t.Errorf("Content length doesn't match")
	}
	res.Body.Close()
}
