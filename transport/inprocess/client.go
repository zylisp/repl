package inprocess

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zylisp/repl/protocol"
)

var clientIDCounter uint64

// Client implements an in-process REPL client.
type Client struct {
	server    *Server
	responses chan *protocol.Message
	clientID  string
	mu        sync.Mutex
	msgID     uint64
}

// NewClient creates a new in-process client.
func NewClient() *Client {
	id := atomic.AddUint64(&clientIDCounter, 1)
	return &Client{
		clientID: fmt.Sprintf("client-%d", id),
	}
}

// Connect connects the client to an in-process server.
// The addr parameter should be a *Server instance or "in-process".
func (c *Client) Connect(ctx context.Context, addr interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Handle different address types
	switch v := addr.(type) {
	case *Server:
		c.server = v
	case string:
		// For universal client compatibility, "in-process" means we need
		// the server to be set separately
		if v != "in-process" && v != "" {
			return fmt.Errorf("invalid in-process address: %q", v)
		}
		if c.server == nil {
			return fmt.Errorf("in-process client requires server to be set")
		}
	default:
		return fmt.Errorf("invalid address type for in-process client: %T", addr)
	}

	// Register with the server
	c.responses = c.server.registerClient(c.clientID)
	return nil
}

// SetServer sets the server for this client (used by the factory).
func (c *Client) SetServer(server *Server) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.server = server
}

// Eval sends code to be evaluated and returns the result.
func (c *Client) Eval(ctx context.Context, code string) (*Result, error) {
	c.mu.Lock()
	msgID := atomic.AddUint64(&c.msgID, 1)
	c.mu.Unlock()

	// Create request message
	req := &protocol.Message{
		Op:      "eval",
		ID:      fmt.Sprintf("%d", msgID),
		Session: c.clientID, // Use Session field to identify client
		Code:    code,
	}

	// Send request
	if err := c.server.sendRequest(req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-c.responses:
		return messageToResult(resp), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.server != nil {
		c.server.unregisterClient(c.clientID)
		c.server = nil
	}
	return nil
}

// Result represents the outcome of a REPL operation.
type Result struct {
	ID     string
	Value  interface{}
	Output string
	Status []string
}

// messageToResult converts a protocol.Message to a Result.
func messageToResult(msg *protocol.Message) *Result {
	return &Result{
		ID:     msg.ID,
		Value:  msg.Value,
		Output: msg.Output,
		Status: msg.Status,
	}
}
