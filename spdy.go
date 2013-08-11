package falcore

import (
	"net"
	"sync"
	"time"

	"code.google.com/p/go.net/spdy"
)

// Goroutines:
//   1 for network reading & session control
//   1 for network writing (to not block reader while sending)
//   1 per active stream

type spdyConnection struct {
	conn               net.Conn
	framer             *spdy.Framer
	streams            map[uint32]chan spdy.Frame
	sendChan           chan spdy.Frame
	closeChan          chan int
	closeOnce          *sync.Once
	goaway             bool
	lastAcceptedStream uint32
	lock               *sync.RWMutex
}

// Handle a new connection.  Sets everything up, starts writer routine, and becomes reader routine.
// Blocks until the session is complete.
func (srv *Server) spdyHandler(c net.Conn) {
	// Create a framer
	var err error
	var session = new(spdyConnection)

	// Use buffered readers and writers
	var rdrpe = srv.bufferPool.Take(c)
	var wtrpe = srv.writeBufferPool.Take(c)
	defer srv.bufferPool.Give(rdrpe)
	defer srv.bufferPool.Give(wtrpe)

	if session.framer, err = spdy.NewFramer(rdrpe.Br, wtrpe.Bw); err != nil {
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
	defer func() {
		srv.spdyCloseSession(session)
		srv.connectionFinished(c, nil)
	}()

	// Become reader goroutine.  
	srv.spdyReader(session)
}

// send using select incase writer is already shutdown
func (session *spdyConnection) send(frame spdy.Frame) bool {
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

func (session *spdyConnection) lookupStream(streamId uint32) chan spdy.Frame {
	session.lock.RLock()
	defer session.lock.RUnlock()
	return session.streams[streamId]
}

func (session *spdyConnection) deregisterStream(streamId uint32) {
	session.lock.Lock()
	defer session.lock.Unlock()
	delete(session.streams, streamId)
}

// Main read loop.  Also manages session state.
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
			// TODO: Error performing framer operation
		}
		if session.goaway && len(session.streams) == 0 {
			keepalive = false
		}
	}
}

// Main write loop.
func (srv *Server) spdyWriter(session *spdyConnection) {
	for {
		select {
		case <-srv.stopAccepting:
			// TODO: send goaway frame, etc
			session.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		case frame := <-session.sendChan:
			// Attempt to write frame to the wire
			if err := session.framer.WriteFrame(frame); err != nil {
				srv.spdyCloseSession(session)
			}
		case <-session.closeChan:
			// aaand we're done.  reader goroutine will clean up
			return
		}
	}
}

// Mark the session as closed. This will trigger the reader and writers to shut down
func (srv *Server) spdyCloseSession(session *spdyConnection) {
	// close the close chan to signal we're done
	session.closeOnce.Do(func() {
		close(session.closeChan)
	})
}

// Handle an individual stream
func (srv *Server) spdyHandleStream(session *spdyConnection, streamId uint32, fchan chan spdy.Frame) {
	defer session.deregisterStream(streamId)
	for {
		_, ok := <-fchan
		if ok {

		} else {

		}
	}
}
