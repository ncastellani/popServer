package popserver

import (
	"fmt"
	"strconv"
	"strings"
)

type Executable interface {
	Run(c *Client, args []string) (int, error)
}

// QUIT

type QuitCommand struct{}

func (cmd QuitCommand) Run(c *Client, args []string) (int, error) {
	newState := c.currentState
	if c.currentState == STATE_TRANSACTION {

		// call the update (expunge)
		err := c.user.Update()
		if err != nil {
			c.writeErr("update operation failed")
			return 0, err
		}

		// unlock the grant
		err = c.user.Unlock()
		if err != nil {
			c.writeErr(ErrUnableToUnlockUser.Error())
			return 0, err
		}

		newState = STATE_UPDATE
	}

	c.isAlive = false
	c.writeOk("goodbye. closing connection")

	return newState, nil
}

// USER

type UserCommand struct{}

func (cmd UserCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_AUTHORIZATION {
		return 0, ErrInvalidState
	}
	if len(args) != 1 {
		return 0, ErrInvalidArgsCount
	}

	c.username = args[0]
	c.writeOk("user set [set: %v]", c.username)

	return STATE_AUTHORIZATION, nil
}

// PASS

type PassCommand struct{}

func (cmd PassCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_AUTHORIZATION {
		return 0, ErrInvalidState
	}
	if c.lastCommand != "USER" {
		c.writeErr("PASS can be executed only directly after USER command")
		return STATE_AUTHORIZATION, nil
	}
	if len(args) != 1 {
		return 0, ErrInvalidArgsCount
	}

	c.password = args[0]

	// call the login backend
	user, err := c.backend.Login(c.conn.RemoteAddr().String(), c.username, c.password, c.writer)
	if err != nil {
		c.writeErr(err.Error())
		return STATE_AUTHORIZATION, nil
	}

	c.user = user

	// grant lock on this user
	err = c.user.Lock()
	if err != nil {
		c.writeErr(ErrUnableToLockUser.Error())
		return 0, fmt.Errorf("error granting user lock [user: %v] [err: %v]", c.username, err)
	}

	c.writeOk("login sucessfully performed [user: %v]", c.username)

	return STATE_TRANSACTION, nil
}

// STAT

type StatCommand struct{}

func (cmd StatCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}

	messages, octets, err := c.user.Stat()
	if err != nil {
		return 0, fmt.Errorf("error calling STAT [user: %v] [err: %v]", c.username, err)
	}

	c.writeOk("%d %d", messages, octets)

	return STATE_TRANSACTION, nil
}

// LIST

type ListCommand struct{}

func (cmd ListCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}

	if len(args) > 0 {

		msgId, err := strconv.Atoi(args[0])
		if err != nil {
			c.writeErr("invalid argument [arg: %v]", args[0])
			return 0, nil
		}

		exists, octets, err := c.user.ListMessage(msgId)
		if err != nil {
			return 0, fmt.Errorf("error calling 'LIST %v' [user: %v] [err: %v]", msgId, c.username, err)
		}

		if !exists {
			c.writeErr("no such message")
			return STATE_TRANSACTION, nil
		}

		c.writeOk("%d %d", msgId, octets)

	} else {

		octets, msgCount, err := c.user.List()
		if err != nil {
			return 0, fmt.Errorf("error calling LIST [user: %v] [err: %v]", c.username, err)
		}

		c.writeOk("%d messages out of %v", len(octets), msgCount)

		messagesList := make([]string, msgCount+1)
		for i, octet := range octets {
			messagesList[i] = fmt.Sprintf("%d %d", i, octet)
		}

		c.writeMulti(messagesList, true)
	}

	return STATE_TRANSACTION, nil
}

// RETR

type RetrCommand struct{}

func (cmd RetrCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}
	if len(args) == 0 {
		c.writeErr("missing argument for RETR command")
		return 0, fmt.Errorf("missing argument for RETR [user: %v]", c.username)
	}

	msgId, err := strconv.Atoi(args[0])
	if err != nil {
		c.writeErr("invalid argument [arg: %v]", args[0])
		return 0, fmt.Errorf("invalid argument for RETR [user: %v] [err: %v]", c.username, err)
	}

	message, err := c.user.Retr(msgId)
	if err != nil {
		c.writeErr(err.Error())
		return STATE_TRANSACTION, nil
	}

	lines := strings.Split(message, "\n")
	c.writeOk("")
	c.writeMulti(lines, false)

	return STATE_TRANSACTION, nil
}

// DELE

type DeleCommand struct{}

func (cmd DeleCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}
	if len(args) == 0 {
		c.writeErr("missing argument for DELE command")
		return 0, fmt.Errorf("missing argument for DELE [user: %v]", c.username)
	}

	msgId, err := strconv.Atoi(args[0])
	if err != nil {
		c.writeErr("invalid argument [arg: %v]", args[0])
		return 0, fmt.Errorf("invalid argument for DELE [user: %v] [err: %v]", c.username, err)
	}

	err = c.user.Dele(msgId)
	if err != nil {
		return 0, fmt.Errorf("error calling 'DELE %v' [user: %v] [err: %v]", msgId, c.username, err)
	}

	c.writeOk("message %d deleted sucessfully", msgId)

	return STATE_TRANSACTION, nil
}

// NOOP

type NoopCommand struct{}

func (cmd NoopCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}

	c.writeOk("")
	return STATE_TRANSACTION, nil
}

// RSET

type RsetCommand struct{}

func (cmd RsetCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}

	err := c.user.Rset()
	if err != nil {
		return 0, fmt.Errorf("error calling RSET [user: %v] [err: %v]", c.username, err)
	}

	c.writeOk("")

	return STATE_TRANSACTION, nil
}

// UIDL

type UidlCommand struct{}

func (cmd UidlCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}

	if len(args) > 0 {

		msgId, err := strconv.Atoi(args[0])
		if err != nil {
			c.writeErr("invalid argument [arg: %v]", args[0])
			return 0, fmt.Errorf("invalid argument for UIDL [user: %v] [err: %v]", c.username, err)
		}

		exists, uid, err := c.user.UidlMessage(msgId)
		if err != nil {
			return 0, fmt.Errorf("error calling 'UIDL %v' [user: %v] [err: %v]", msgId, c.username, err)
		}

		if !exists {
			c.writeErr("no such message")
			return STATE_TRANSACTION, nil
		}

		c.writeOk("%d %s", msgId, uid)

	} else {

		uids, msgCount, err := c.user.Uidl()
		if err != nil {
			return 0, fmt.Errorf("Error calling UIDL for user %s: %v", c.user, err)
		}

		c.writeOk("%d messages out of %v", len(uids), msgCount)

		uidsList := make([]string, msgCount+1)
		for i, uid := range uids {
			uidsList[i] = fmt.Sprintf("%d %s", i, uid)
		}

		c.writeMulti(uidsList, true)

	}

	return STATE_TRANSACTION, nil
}

// CAPA

type CapaCommand struct{}

func (cmd CapaCommand) Run(c *Client, args []string) (int, error) {
	commands := []string{"USER", "UIDL"}

	c.writeOk("")
	c.writeMulti(commands, true)

	return c.currentState, nil
}
