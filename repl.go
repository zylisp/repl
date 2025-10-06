package repl

import (
	"context"
	"fmt"

	"github.com/zylisp/repl/transport/inprocess"
	"github.com/zylisp/repl/transport/tcp"
	"github.com/zylisp/repl/transport/unix"
)

// Result represents the outcome of a REPL operation.
type Result struct {
	// ID is the message ID that correlates with the original request
	ID string

	// Value is the Zylisp evaluation result (success or error-as-data)
	Value interface{}

	// Output contains captured stdout/stderr from the evaluation
	Output string

	// Status contains operation status flags (e.g., "done", "error", "interrupted")
	Status []string
}

// Server defines the REPL server interface.
// A server listens for client connections and evaluates Zylisp code.
type Server interface {
	// Start begins listening for connections.
	// It blocks until the context is cancelled or an error occurs.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the server.
	// It waits for active connections to complete within the context deadline.
	Stop(ctx context.Context) error

	// Addr returns the address the server is listening on.
	// The format depends on the transport type.
	Addr() string
}

// Client defines the REPL client interface.
// A client connects to a REPL server and sends evaluation requests.
type Client interface {
	// Connect establishes a connection to a REPL server at the given address.
	// The transport is auto-detected from the address format.
	Connect(ctx context.Context, addr string) error

	// Eval sends code to be evaluated and returns the result.
	// The returned error is for protocol/transport errors only.
	// Zylisp evaluation errors are returned in Result.Value as error-as-data.
	Eval(ctx context.Context, code string) (*Result, error)

	// Close closes the client connection.
	Close() error
}

// ServerConfig provides configuration for creating a REPL server.
type ServerConfig struct {
	// Transport specifies the transport type: "in-process", "unix", or "tcp"
	Transport string

	// Addr is the address to bind to.
	// Format depends on transport:
	//   - in-process: ignored (use "" or "in-process")
	//   - unix: path to socket file (e.g., "/tmp/zylisp.sock")
	//   - tcp: host:port (e.g., "localhost:5555" or ":5555")
	Addr string

	// Codec specifies the message encoding: "json" or "msgpack"
	// Only used for unix and tcp transports (in-process uses direct Go values)
	Codec string

	// Evaluator is the function that evaluates Zylisp code.
	// It returns:
	//   - result: the evaluation result (including error-as-data)
	//   - output: captured stdout/stderr
	//   - error: only for catastrophic failures (should be rare)
	Evaluator func(code string) (result interface{}, output string, err error)
}

// NewServer creates a new REPL server with the given configuration.
func NewServer(config ServerConfig) (Server, error) {
	// Default codec to "json"
	if config.Codec == "" {
		config.Codec = "json"
	}

	// Create server based on transport type
	switch config.Transport {
	case "in-process", "":
		return inprocess.NewServer(config.Evaluator), nil
	case "unix":
		if config.Addr == "" {
			return nil, fmt.Errorf("unix transport requires Addr")
		}
		return unix.NewServer(config.Addr, config.Codec, config.Evaluator), nil
	case "tcp":
		if config.Addr == "" {
			return nil, fmt.Errorf("tcp transport requires Addr")
		}
		return tcp.NewServer(config.Addr, config.Codec, config.Evaluator), nil
	default:
		return nil, fmt.Errorf("unknown transport: %s", config.Transport)
	}
}

// NewClient creates a new REPL client.
// The transport will be auto-detected when Connect is called.
func NewClient() Client {
	return &UniversalClient{}
}

// UniversalClient is a client that auto-detects the transport from the address.
type UniversalClient struct {
	transport string
	impl      interface{} // Actual transport-specific client
}

// Connect establishes a connection to a REPL server, auto-detecting the transport.
func (c *UniversalClient) Connect(ctx context.Context, addr string) error {
	transport, codec := detectTransport(addr)
	c.transport = transport

	switch transport {
	case "in-process":
		// In-process requires special handling - not supported via universal client yet
		return fmt.Errorf("in-process transport not supported via universal client")
	case "unix":
		client := unix.NewClient(codec)
		if err := client.Connect(ctx, addr, codec); err != nil {
			return err
		}
		c.impl = client
		return nil
	case "tcp":
		// Clean up address if it has tcp:// prefix
		if len(addr) > 6 && addr[:6] == "tcp://" {
			addr = addr[6:]
		}
		client := tcp.NewClient(codec)
		if err := client.Connect(ctx, addr, codec); err != nil {
			return err
		}
		c.impl = client
		return nil
	default:
		return fmt.Errorf("unknown transport: %s", transport)
	}
}

// Eval sends code to be evaluated.
func (c *UniversalClient) Eval(ctx context.Context, code string) (*Result, error) {
	switch c.transport {
	case "unix":
		client := c.impl.(*unix.Client)
		result, err := client.Eval(ctx, code)
		if err != nil {
			return nil, err
		}
		return &Result{
			ID:     result.ID,
			Value:  result.Value,
			Output: result.Output,
			Status: result.Status,
		}, nil
	case "tcp":
		client := c.impl.(*tcp.Client)
		result, err := client.Eval(ctx, code)
		if err != nil {
			return nil, err
		}
		return &Result{
			ID:     result.ID,
			Value:  result.Value,
			Output: result.Output,
			Status: result.Status,
		}, nil
	default:
		return nil, fmt.Errorf("not connected")
	}
}

// Close closes the client connection.
func (c *UniversalClient) Close() error {
	switch c.transport {
	case "unix":
		return c.impl.(*unix.Client).Close()
	case "tcp":
		return c.impl.(*tcp.Client).Close()
	default:
		return nil
	}
}

// detectTransport detects the transport type and codec from an address string.
func detectTransport(addr string) (transport, codec string) {
	codec = "json" // default codec

	// Check for explicit transport prefix
	if len(addr) >= 7 && addr[:7] == "unix://" {
		return "unix", codec
	}
	if len(addr) >= 6 && addr[:6] == "tcp://" {
		return "tcp", codec
	}

	// Empty or "in-process" means in-process
	if addr == "" || addr == "in-process" {
		return "in-process", ""
	}

	// Path starting with / or . means unix
	if len(addr) > 0 && (addr[0] == '/' || addr[0] == '.') {
		return "unix", codec
	}

	// Default to TCP for host:port format
	return "tcp", codec
}
