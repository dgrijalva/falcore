package falcore

import (
	"net"
	"time"

	"code.google.com/p/go.net/spdy"
)

type spdyConnection struct {
	conn net.Conn
	framer *spdy.Framer
	streams map[uint32]chan spdy.Frame
	sendChan chan spdy.Frame
	closeChan chan int
	goaway bool
}

func (srv *Server) spdyHandler(c net.Conn){
	// Create a framer
	var err error
	var session = new(spdyConnection)

	// FIXME: should we be using buffered reader/writers?
	if session.framer, err = spdy.NewFramer(c, c); err != nil {
		Error("Couldn't get spdy.Framer for conn: %v", err)
		srv.connectionFinished(c, nil)
		return
	}

	session.conn = c
	session.streams = make(map[uint32]chan spdy.Frame)
	session.sendChan = make(chan spdy.Frame)
	session.closeChan = make(chan int)
	session.goaway = false
	
	go srv.spdyWriter(session)
	defer srv.connectionFinished(c, session.closeChan)
	
	srv.spdyReader(session)
}

func (srv *Server) spdyHandleStream(f *spdyConnection, fchan chan spdy.Frame) {
	for {
		_, ok := <- fchan
		if ok {
			
		} else {
			
		}
	}
}

func (srv *Server) spdyReader(session *spdyConnection) {
	var frame spdy.Frame
	var err error
	keepalive := true
	for err == nil && keepalive {
		if frame, err = session.framer.ReadFrame(); err == nil {
			switch fr := frame.(type) {
			case *spdy.NoopFrame:
				// Do nothing
			case *spdy.GoAwayFrame:
				session.goaway = true
			case *spdy.PingFrame:
				session.sendChan <- frame
			case *spdy.SynStreamFrame:
				if !session.goaway {
					c := make(chan spdy.Frame)
					session.streams[fr.StreamId] = c
					go srv.spdyHandleStream(session, c)
					c <- frame
				}
			}
		}
		if session.goaway && len(session.streams) == 0 {
			keepalive = false
		}
	}
}

func (srv *Server) spdyWriter(session *spdyConnection) {
	select {
	case <-srv.stopAccepting:
		// TODO: send goaway frame, etc
		session.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	// case <-connClosed:
	}
}