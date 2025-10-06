package client

import (
	"github.com/zylisp/repl/server"
)

// Client represents a REPL client
type Client struct {
	server *server.Server
}

// NewClient creates a new REPL client
func NewClient(srv *server.Server) *Client {
	return &Client{server: srv}
}

// Send sends an expression to the server and returns the result
func (c *Client) Send(expr string) (string, error) {
	return c.server.Eval(expr)
}

// Reset resets the server environment
func (c *Client) Reset() {
	c.server.Reset()
}
