package popgun

import (
	"fmt"
	"strconv"
)

type Executable interface {
	Run(c *Client, args []string) (int, error)
}

type QuitCommand struct{}

func (cmd QuitCommand) Run(c *Client, args []string) (int, error) {
	c.isAlive = false
	c.printer.Ok("Goodbye")
	return STATE_UPDATE, nil
}

type UserCommand struct{}

func (cmd UserCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_AUTHORIZATION {
		return 0, ErrInvalidState
	}
	c.user = args[0]
	c.printer.Ok("")
	return STATE_AUTHORIZATION, nil
}

type PassCommand struct{}

func (cmd PassCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_AUTHORIZATION {
		return 0, ErrInvalidState
	}
	if c.lastCommand != "USER" {
		c.printer.Err("PASS can be executed only directly after USER command")
		return STATE_AUTHORIZATION, nil
	}
	c.pass = args[0]
	if !c.authorizator.Authorize(c.user, c.pass) {
		c.printer.Err("Invalid username or password")
		return STATE_AUTHORIZATION, nil
	}
	c.printer.Ok("User Successfully Logged on")
	return STATE_TRANSACTION, nil
}

type StatCommand struct{}

func (cmd StatCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}
	if !c.authorizator.IsAuthorized() {
		return 0, ErrUnauthorized
	}
	messages, octets, err := c.backend.Stat(c.user)
	if err != nil {
		return 0, fmt.Errorf("Error calling Stat for user %s: %v", c.user, err)
	}
	c.printer.Ok("%d %d", messages, octets)
	return STATE_TRANSACTION, nil
}

type ListCommand struct{}

func (cmd ListCommand) Run(c *Client, args []string) (int, error) {
	if c.currentState != STATE_TRANSACTION {
		return 0, ErrInvalidState
	}
	if !c.authorizator.IsAuthorized() {
		return 0, ErrUnauthorized
	}
	if len(args) > 0 {
		msgId, err := strconv.Atoi(args[0])
		if err != nil {
			c.printer.Err("Invalid argument: %s", args[0])
			return 0, fmt.Errorf("Invalid argument for LIST given by user %s: %v", c.user, err)
		}
		exists, octets, err := c.backend.ListMessage(c.user, msgId)
		if err != nil {
			return 0, fmt.Errorf("Error calling LIST [arg] for user %s: %v", c.user, err)
		}
		if !exists {
			c.printer.Err("no such message")
			return STATE_TRANSACTION, nil
		}
		c.printer.Ok("%d %d", msgId, octets)
	} else {
		messages, err := c.backend.List(c.user)
		if err != nil {
			return 0, fmt.Errorf("Error calling LIST for user %s: %v", c.user, err)
		}
		r := fmt.Sprintf("%d messages", len(messages))
		for _, msg := range messages {
			r = fmt.Sprintf("%s\n%d %d", r, msg[0], msg[1])
		}
		c.printer.Ok(r)
	}

	return STATE_TRANSACTION, nil
}

type RetrCommand struct{}

func (cmd RetrCommand) Run(c *Client, args []string) (int, error) {
	return 0, nil
}

type DeleCommand struct{}

func (cmd DeleCommand) Run(c *Client, args []string) (int, error) {
	return 0, nil
}

type NoopCommand struct{}

func (cmd NoopCommand) Run(c *Client, args []string) (int, error) {
	return 0, nil
}

type RsetCommand struct{}

func (cmd RsetCommand) Run(c *Client, args []string) (int, error) {
	return 0, nil
}