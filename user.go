package popserver

// interface of functions to perform user (email account) operations
type User interface {
	Update() error
	Lock() error
	Unlock() error

	Stat() (messages, octets int, err error)
	List() (octets map[int]int, err error)
	ListMessage(msgID int) (exists bool, octets int, err error)
	Retr(msgID int) (message string, err error)
	Dele(msgID int) error
	Rset() error
	Uidl() (uids map[int]string, err error)
	UidlMessage(msgID int) (exists bool, uid string, err error)
}
