package falcore

import (
	"net"
	"fmt"
)

func (srv *Server) spdyHandler(c net.Conn){
	fmt.Println("I am SPDY!")
	c.Close()
}