package falcore

import (
	"net"
	"time"
	"sync"

	"code.google.com/p/go.net/spdy"
)

// Goroutines:
//   1 for network reading & session control
//   1 for network writing (to not block reader while sending)
//   1 per active session

type spdyConnection struct {
	conn net.Conn
	framer *spdy.Framer
	streams map[uint32]chan spdy.Frame
	sendChan chan spdy.Frame
	closeChan chan int
	closeOnce *sync.Once
	goaway bool
	lastAcceptedStream uint32
	lock *sync.RWMutex
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
	session.closeOnce = new(sync.Once)
	session.goaway = false
	session.lock = new(sync.RWMutex)
	
	go srv.spdyWriter(session)
	defer func(){
		srv.connectionFinished(c, nil)
		session.closeOnce.Do(func(){
			close(session.closeChan)
		})
	}()
	
	srv.spdyReader(session)
}

func (srv *Server) spdyHandleStream(session *spdyConnection, streamId uint32, fchan chan spdy.Frame) {
	defer session.deregisterStream(streamId)
	for {
		_, ok := <- fchan
		if ok {
			
		} else {
			
		}
	}
}

// send using select incase writer is already shutdown
func (session *spdyConnection) send(frame spdy.Frame)bool {
	select {
	case session.sendChan <- frame:
		return true
	case <-session.closeChan:
	}
	return false
}

func (session *spdyConnection) registerStream(streamId uint32, fchan chan spdy.Frame) {
	session.lock.Lock()
	defer session.lock.Unlock()

	session.streams[streamId] = fchan
	if streamId > session.lastAcceptedStream {
		session.lastAcceptedStream = streamId
	}
}

func (session *spdyConnection) lookupStream(streamId uint32)(chan spdy.Frame) {
	session.lock.RLock()
	defer session.lock.RUnlock()
	return session.streams[streamId]
}

func (session *spdyConnection) deregisterStream(streamId uint32) {
	session.lock.Lock()
	defer session.lock.Unlock()
	delete(session.streams, streamId)
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
				session.send(frame)
			case *spdy.SynStreamFrame:
				if !session.goaway {
					c := make(chan spdy.Frame)
					session.registerStream(fr.StreamId, c)
					go srv.spdyHandleStream(session, fr.StreamId, c)
					c <- frame
				}
			}
		} else {
			// Error performing framer operation
		}
		if session.goaway && len(session.streams) == 0 {
			keepalive = false
		}
	}
}

func (srv *Server) spdyWriter(session *spdyConnection) {
	for {
		select {
		case <-srv.stopAccepting:
			// TODO: send goaway frame, etc
			session.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		case frame := <-session.sendChan:
			// Attempt to write frame to the wire
			if err := session.framer.WriteFrame(frame); err != nil {
				// close the close chan to signal we're done
				session.closeOnce.Do(func(){
					close(session.closeChan)
				})
			}
		case <-session.closeChan:
			srv.connectionFinished(session.conn, nil)
			// aaand we're done.  reader goroutine will close up
			return
		}
	}
}
