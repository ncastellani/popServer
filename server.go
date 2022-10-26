package popserver

import (
	"crypto/tls"
	"log"
	"net"
	"time"
)

// required values for starting up a new POP server
type Server struct {
	Greeting  string        // welcome message sent on inbound connection
	Address   string        // host and port to expose the TCP server
	Backend   Backend       // backend implemented by the user to handle data
	Timeout   time.Duration // time until close an opened connection
	TLSConfig *tls.Config   // config for handling TLS connections
	Logger    *log.Logger   // used to print-out debug data

	listener net.Listener // net package server listener
}

func NewServer(addr string, back Backend) *Server {
	return &Server{
		Greeting: "POP3 server ready",
		Address:  addr,
		Backend:  back,
		Logger:   log.Default(),
	}
}

// start up the POP server without TLS
func (s *Server) ListenAndServe() error {
	var err error

	s.listener, err = net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.Logger.Printf("error while accepting new inbound connection [err: %v]", err)
			continue
		}

		go s.serve(conn)
	}

}

// start up the POP server with TLS termination
func (s *Server) ListenAndServeTLS() error {
	var err error

	s.listener, err = tls.Listen("tcp", s.Address, s.TLSConfig)
	if err != nil {
		return err
	}

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.Logger.Printf("error while accepting new inbound connection [err: %v]", err)
			continue
		}

		go s.serve(conn)
	}

}

// close the listener opened by the start up proccess
func (s *Server) Close() error {
	return s.listener.Close()
}

// return the server greeting and setup a Client to handle the connection
func (s *Server) serve(conn net.Conn) {
	client := newClient(conn, s.Backend, s.Timeout)
	client.writer = s.Logger.Writer() // set the logger as the default io writer for logging
	client.writeOk(s.Greeting)        // respond with the server GREETING
	client.handle()
}
