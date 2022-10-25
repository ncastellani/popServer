package popserver

import "io"

// interface of functions that must be implemented for server usage
type Backend interface {
	Login(remoteAddr, username, password string, writer io.Writer) (User, error)
}
