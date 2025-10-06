package protocol

// Message represents a protocol message exchanged between client and server.
// Messages use a simple map-based structure that can be encoded in multiple formats.
type Message struct {
	// Op is the operation name (e.g., "eval", "load-file", "describe")
	Op string `json:"op"`

	// ID is a unique message identifier for correlating requests and responses
	ID string `json:"id"`

	// Session is the session ID (reserved for future explicit session support)
	Session string `json:"session,omitempty"`

	// Code is the code to evaluate (for eval and load-file operations)
	Code string `json:"code,omitempty"`

	// Status contains status flags: "done", "error", "interrupted", etc.
	Status []string `json:"status,omitempty"`

	// Value contains the evaluation result, including Zylisp error-as-data results
	// This is interface{} to support arbitrary Zylisp values
	Value interface{} `json:"value,omitempty"`

	// Output contains captured stdout/stderr from evaluation
	Output string `json:"output,omitempty"`

	// ProtocolError contains protocol-level errors only (not Zylisp evaluation errors)
	// Examples: malformed messages, connection issues, unknown operations
	ProtocolError string `json:"protocol_error,omitempty"`

	// Data contains additional operation-specific data
	Data map[string]interface{} `json:"data,omitempty"`
}
