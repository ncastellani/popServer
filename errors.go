package popserver

import "fmt"

var (
	ErrInvalidState       = fmt.Errorf("command not supported on this state")
	ErrInvalidArgsCount   = fmt.Errorf("args count is not sufficient for this command")
	ErrUnableToLockUser   = fmt.Errorf("server is unable to grant a user lock")
	ErrUnableToUnlockUser = fmt.Errorf("server is unable to unlock user")
)
