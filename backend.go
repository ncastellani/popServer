package popserver

// interface of functions that must be implemented for server usage
type Backend interface {
	Login(remoteAddr, username, password string) (User, error)
}
