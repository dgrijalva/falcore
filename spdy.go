package falcore

import (
	"net"
	"code.google.com/p/go.net/spdy"
	"time"
)

func (srv *Server) spdyHandler(c net.Conn){
	// Create a framer
	var framer *spdy.Framer
	var err error

	var closeSentinelChan = make(chan int)
	go srv.spdySentinel(c, closeSentinelChan)
	defer srv.connectionFinished(c, closeSentinelChan)

	// FIXME: should we be using buffered reader/writers?
	if framer, err = spdy.NewFramer(c, c); err != nil {
		Error("Couldn't get spdy.Framer for conn: %v", err)
		srv.connectionFinished(c, nil)
		return
	}
	
	var frame spdy.Frame
	keepalive := true
	for err == nil && keepalive {
		if frame, err = framer.ReadFrame(); err == nil {
			switch frame.(type) {
			case *spdy.GoAwayFrame:
				keepalive = false
			}
		}
	}
	
}

func (srv *Server) spdyHandleStream(f *spdy.Framer, frame spdy.Frame) {
	
}

func (srv *Server) spdySentinel(c net.Conn, connClosed chan int) {
	select {
	case <-srv.stopAccepting:
		// TODO: send goaway frame, etc
		c.SetReadDeadline(time.Now().Add(3 * time.Second))
	case <-connClosed:
	}
}