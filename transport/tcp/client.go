package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/zylisp/repl/protocol"
)

// Client implements a TCP REPL client.
type Client struct {
	conn  net.Conn
	codec protocol.Codec
	mu    sync.Mutex
	msgID uint64
}

// NewClient creates a new TCP client.
func NewClient(codecFormat string) *Client {
	return &Client{}
}

// Connect establishes a connection to a TCP server.
func (c *Client) Connect(ctx context.Context, addr string, codecFormat string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Dial the TCP server
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to tcp server: %w", err)
	}

	c.conn = conn

	// Create codec
	codec, err := protocol.NewCodec(codecFormat, conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create codec: %w", err)
	}
	c.codec = codec

	return nil
}

// Eval sends code to be evaluated and returns the result.
// This is a synchronous request-response operation.
func (c *Client) Eval(ctx context.Context, code string) (*Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate message ID
	msgID := atomic.AddUint64(&c.msgID, 1)

	// Create request
	req := &protocol.Message{
		Op:   "eval",
		ID:   fmt.Sprintf("%d", msgID),
		Code: code,
	}

	// Send request
	if err := c.codec.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	resp := &protocol.Message{}
	if err := c.codec.Decode(resp); err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Convert to Result
	return messageToResult(resp), nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.codec != nil {
		c.codec.Close()
		c.codec = nil
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
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
