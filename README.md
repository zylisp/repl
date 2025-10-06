# Zylisp Remote REPL Protocol

A remote REPL protocol implementation for Zylisp, supporting multiple transport mechanisms (in-process, Unix domain sockets, and TCP) with a unified API.

## Features

- **Multiple Transports**: In-process, Unix domain sockets, and TCP
- **Auto-Detection**: Client automatically detects transport from address format
- **JSON Codec**: Human-readable newline-delimited JSON protocol
- **Errors as Data**: Follows Zylisp's philosophy - evaluation errors are data, not exceptions
- **Simple Session Model**: Connection = session (no explicit session management)
- **Interface-First Design**: Extensible architecture for future enhancements

## Quick Start

### Server

```go
package main

import (
    "context"
    "github.com/zylisp/repl"
)

func main() {
    // Create a TCP server
    server, _ := repl.NewServer(repl.ServerConfig{
        Transport: "tcp",
        Addr:      ":5555",
        Codec:     "json",
        Evaluator: myZylispEval,
    })

    // Start the server
    server.Start(context.Background())
}

func myZylispEval(code string) (interface{}, string, error) {
    // Your Zylisp evaluator implementation
    return nil, "", nil
}
```

### Client

```go
package main

import (
    "context"
    "fmt"
    "github.com/zylisp/repl"
)

func main() {
    // Create client (transport auto-detected)
    client := repl.NewClient()

    // Connect to server
    client.Connect(context.Background(), "localhost:5555")
    defer client.Close()

    // Evaluate code
    result, err := client.Eval(context.Background(), "(+ 1 2)")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Result: %v\n", result.Value)
}
```

## Architecture

### Protocol Layers

1. **Message Format** (`protocol/message.go`): Core message structure
2. **Codec** (`protocol/codec.go`): Message encoding/decoding (JSON, MessagePack)
3. **Operations** (`operations/operations.go`): Operation handlers (eval, load-file, describe)
4. **Transports** (`transport/*/`): Connection mechanisms
5. **Unified API** (`repl.go`): High-level Server/Client interfaces

### Supported Transports

#### In-Process
- Zero-overhead communication using Go channels
- Perfect for testing and embedded use cases
- Address: `"in-process"` or `""`

```go
server, _ := repl.NewServer(repl.ServerConfig{
    Transport: "in-process",
    Evaluator: myEval,
})
```

#### Unix Domain Sockets
- High-performance local IPC
- Ideal for development tools
- Address: `/path/to/socket` or `unix:///path/to/socket`

```go
server, _ := repl.NewServer(repl.ServerConfig{
    Transport: "unix",
    Addr:      "/tmp/zylisp.sock",
    Codec:     "json",
    Evaluator: myEval,
})
```

#### TCP
- Remote REPL access across network
- Address: `host:port` or `tcp://host:port`

```go
server, _ := repl.NewServer(repl.ServerConfig{
    Transport: "tcp",
    Addr:      ":5555",
    Codec:     "json",
    Evaluator: myEval,
})
```

## Protocol Specification

### Message Format

Messages are JSON objects with the following fields:

```json
{
  "op": "eval",
  "id": "1",
  "code": "(+ 1 2)",
  "status": ["done"],
  "value": 3,
  "output": "",
  "protocol_error": "",
  "data": {}
}
```

**Fields:**
- `op`: Operation name (e.g., "eval", "load-file", "describe")
- `id`: Unique message identifier for request/response correlation
- `code`: Code to evaluate (for eval operations)
- `status`: Status flags (`["done"]`, `["error"]`, `["interrupted"]`)
- `value`: Evaluation result (including Zylisp error-as-data)
- `output`: Captured stdout/stderr
- `protocol_error`: Protocol-level errors only (not Zylisp errors)
- `data`: Additional operation-specific data

### Operations

#### eval
Evaluate Zylisp code.

**Request:**
```json
{"op": "eval", "id": "1", "code": "(+ 1 2)"}
```

**Response:**
```json
{"id": "1", "value": 3, "status": ["done"]}
```

#### load-file
Load and evaluate a file.

**Request:**
```json
{
  "op": "load-file",
  "id": "2",
  "data": {"file": "/path/to/file.zylisp"}
}
```

**Response:**
```json
{"id": "2", "value": "...", "status": ["done"]}
```

#### describe
Get server capabilities.

**Request:**
```json
{"op": "describe", "id": "3"}
```

**Response:**
```json
{
  "id": "3",
  "status": ["done"],
  "data": {
    "versions": {"zylisp": "0.1.0", "protocol": "0.1.0"},
    "ops": ["eval", "load-file", "describe", "interrupt"],
    "transports": ["in-process", "unix", "tcp"]
  }
}
```

#### interrupt
Interrupt running evaluation (stub for now).

**Request:**
```json
{"op": "interrupt", "id": "4", "interrupt-id": "1"}
```

**Response:**
```json
{"id": "4", "status": ["error"], "protocol_error": "not yet implemented"}
```

### Error Handling

The protocol distinguishes between two types of errors:

#### 1. Protocol Errors
Connection failures, malformed messages, unknown operations. These are returned as Go errors and set the `protocol_error` field.

```go
result, err := client.Eval(ctx, code)
if err != nil {
    // This is a protocol/transport error
    log.Fatal(err)
}
```

#### 2. Zylisp Evaluation Errors
Type errors, runtime errors, etc. These are **not** Go errors - they're successful evaluations that produced error values (errors-as-data).

```go
result, err := client.Eval(ctx, "(/ 1 0)")
if err != nil {
    // This would be a transport error
    log.Fatal(err)
}
// err is nil - the protocol worked fine
// Check result.Value for Zylisp error-as-data
if errorValue, ok := result.Value.(map[string]interface{}); ok {
    if _, isError := errorValue["error"]; isError {
        fmt.Printf("Zylisp error: %v\n", errorValue)
    }
}
```

### Address Formats

| Format | Transport | Example |
|--------|-----------|---------|
| `""` or `"in-process"` | In-process | `"in-process"` |
| Path starting with `/` or `.` | Unix | `"/tmp/zylisp.sock"` |
| `unix://path` | Unix | `"unix:///tmp/zylisp.sock"` |
| `tcp://host:port` | TCP | `"tcp://localhost:5555"` |
| `host:port` | TCP | `"localhost:5555"` |

## Examples

### TCP Server and Client

```go
// server.go
package main

import (
    "context"
    "github.com/zylisp/repl"
)

func evalZylisp(code string) (interface{}, string, error) {
    // Simple mock evaluator
    if code == "(+ 1 2)" {
        return 3, "", nil
    }
    return nil, "", nil
}

func main() {
    server, _ := repl.NewServer(repl.ServerConfig{
        Transport: "tcp",
        Addr:      ":5555",
        Codec:     "json",
        Evaluator: evalZylisp,
    })

    server.Start(context.Background())
}
```

```go
// client.go
package main

import (
    "context"
    "fmt"
    "github.com/zylisp/repl"
)

func main() {
    client := repl.NewClient()
    client.Connect(context.Background(), "localhost:5555")
    defer client.Close()

    result, err := client.Eval(context.Background(), "(+ 1 2)")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Result: %v\n", result.Value) // Output: Result: 3
}
```

### Unix Domain Socket

```go
server, _ := repl.NewServer(repl.ServerConfig{
    Transport: "unix",
    Addr:      "/tmp/zylisp.sock",
    Codec:     "json",
    Evaluator: myEval,
})
go server.Start(context.Background())

client := repl.NewClient()
client.Connect(context.Background(), "/tmp/zylisp.sock")
result, _ := client.Eval(context.Background(), "(+ 1 2)")
```

### Testing with Netcat

Since the protocol uses newline-delimited JSON, you can test it with `netcat`:

```bash
# Start a TCP server on port 5555
# Then connect with netcat:
$ nc localhost 5555

# Send a request (paste this JSON and press Enter):
{"op":"eval","id":"1","code":"(+ 1 2)"}

# You'll receive:
{"id":"1","value":3,"status":["done"]}
```

## Testing

Run all tests:
```bash
go test ./...
```

Run specific transport tests:
```bash
go test ./transport/inprocess/
go test ./transport/unix/
go test ./transport/tcp/
```

## Future Enhancements

These features are planned but not yet implemented:

1. **Explicit Session Management**: Multiple sessions per connection
2. **Streaming Responses**: Multiple response messages per request
3. **MessagePack Codec**: Binary protocol for performance
4. **Advanced Operations**: Code completion, symbol documentation, jump-to-definition
5. **Security**: TLS support, authentication/authorization
6. **Middleware Architecture**: Pluggable cross-cutting concerns

## Implementation Status

- ✅ Protocol message format
- ✅ JSON codec (fully implemented)
- ⏳ MessagePack codec (placeholder only)
- ✅ In-process transport
- ✅ Unix domain socket transport
- ✅ TCP transport
- ✅ Core operations (eval, load-file, describe)
- ⏳ Interrupt operation (stub only)
- ✅ Universal client with transport auto-detection
- ✅ Comprehensive test coverage

## License

See LICENSE file for details.
