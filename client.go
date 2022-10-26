package popserver

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// contains the data for a connection session
type Client struct {
	conn         net.Conn              // connection accepted by the TCP listener
	isAlive      bool                  // if the connection is still available
	lastCommand  string                // last executed command on the session
	currentState int                   // current state of the session (AUTH/TRANSACTION/UPDATE)
	commands     map[string]Executable // map POP comamnds into its execution handler
	writer       io.Writer             // writer for logging
	timeout      time.Duration         // timeout until closing an open connection

	backend Backend // backend operations source
	user    User    // user (email account) operations source

	// authentication data
	username string
	password string
}

func newClient(conn net.Conn, back Backend, timeout time.Duration) *Client {
	s := Client{
		conn:    conn,
		backend: back,
		writer:  io.Discard,
		timeout: timeout,
	}

	commands := make(map[string]Executable)
	commands["QUIT"] = QuitCommand{}
	commands["USER"] = UserCommand{}
	commands["PASS"] = PassCommand{}
	commands["STAT"] = StatCommand{}
	commands["LIST"] = ListCommand{}
	commands["RETR"] = RetrCommand{}
	commands["DELE"] = DeleCommand{}
	commands["NOOP"] = NoopCommand{}
	commands["RSET"] = RsetCommand{}
	commands["UIDL"] = UidlCommand{}
	commands["CAPA"] = CapaCommand{}
	s.commands = commands

	return &s
}

// write an +OK response with message on wire
func (c *Client) writeOk(msg string, args ...interface{}) {
	fmt.Fprintf(c.conn, "+OK %s\r\n", fmt.Sprintf(msg, args...))
}

// write an -ERR response with the error on wire
func (c *Client) writeErr(msg string, args ...interface{}) {
	fmt.Fprintf(c.conn, "-ERR %s\r\n", fmt.Sprintf(msg, args...))
}

// send a multi line response on the wire
func (s *Client) writeMulti(msgs []string, stripEmpty bool) {
	for _, line := range msgs {
		if line == "" && stripEmpty {
			continue
		}

		line := strings.Trim(line, "\r")
		if strings.HasPrefix(line, ".") {
			fmt.Fprintf(s.conn, ".%s\r\n", line)
		} else {
			fmt.Fprintf(s.conn, "%s\r\n", line)
		}
	}

	fmt.Fprint(s.conn, ".\r\n")
}

// parse the passed input data into a manageable tuple
func (s *Client) parseInput(input string) (string, []string) {
	input = strings.Trim(input, "\r \n")
	cmd := strings.Split(input, " ")
	return strings.ToUpper(cmd[0]), cmd[1:]
}

// handle an inbound connection
func (s *Client) handle() error {
	defer s.conn.Close()
	s.conn.SetDeadline(time.Now().Add(s.timeout))
	s.isAlive = true
	s.currentState = STATE_AUTHORIZATION

	// handle inbound data contents
	reader := bufio.NewReader(s.conn)
	for s.isAlive {

		// according to RFC commands are terminated by CRLF, but we are removing \r in parseInput
		input, err := reader.ReadString('\n')
		if err != nil {

			// handle read input failure
			if err != io.EOF {
				return err
			}

			// unlock the user
			if len(s.username) > 0 {
				s.user.Unlock()
			}

			break
		}

		// parse the input command and check if its valid
		cmd, args := s.parseInput(input)
		exec, ok := s.commands[cmd]
		if !ok {
			s.writeErr("invalid command [cmd: %v]", cmd)
			continue
		}

		// run the command executor
		state, err := exec.Run(s, args)
		if err != nil {
			s.writeErr("error executing command [err: %v] [cmd: %v]", cmd, err)
			return err
		}

		s.lastCommand = cmd
		s.currentState = state
	}

	return nil
}
