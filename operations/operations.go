package operations

import (
	"fmt"
	"os"

	"github.com/zylisp/repl/protocol"
)

// EvaluatorFunc is the function signature for a Zylisp code evaluator.
// It returns:
//   - result: the evaluation result (including error-as-data)
//   - output: captured stdout/stderr
//   - error: only for catastrophic failures (should be rare)
type EvaluatorFunc func(code string) (result interface{}, output string, err error)

// Handler processes a request message and returns a response message.
type Handler struct {
	evaluator EvaluatorFunc
}

// NewHandler creates a new operation handler with the given evaluator.
func NewHandler(evaluator EvaluatorFunc) *Handler {
	return &Handler{
		evaluator: evaluator,
	}
}

// Handle processes a request message and returns a response message.
// It dispatches to the appropriate operation handler based on the Op field.
func (h *Handler) Handle(req *protocol.Message) *protocol.Message {
	// Create base response with the same ID
	resp := &protocol.Message{
		ID: req.ID,
	}

	// Dispatch to operation handler
	switch req.Op {
	case "eval":
		return h.handleEval(req, resp)
	case "load-file":
		return h.handleLoadFile(req, resp)
	case "describe":
		return h.handleDescribe(req, resp)
	case "interrupt":
		return h.handleInterrupt(req, resp)
	case "complete", "info", "eldoc", "lookup", "stdin", "ls-sessions", "clone", "close":
		// Future operations - return not implemented
		resp.Status = []string{"error"}
		resp.ProtocolError = fmt.Sprintf("operation %q not yet implemented", req.Op)
		return resp
	default:
		resp.Status = []string{"error"}
		resp.ProtocolError = fmt.Sprintf("unknown operation: %q", req.Op)
		return resp
	}
}

// handleEval processes the "eval" operation.
func (h *Handler) handleEval(req *protocol.Message, resp *protocol.Message) *protocol.Message {
	if req.Code == "" {
		resp.Status = []string{"error"}
		resp.ProtocolError = "eval operation requires 'code' field"
		return resp
	}

	// Evaluate the code
	result, output, err := h.evaluator(req.Code)
	if err != nil {
		// Catastrophic error (not a Zylisp error-as-data)
		resp.Status = []string{"error"}
		resp.ProtocolError = fmt.Sprintf("evaluator error: %v", err)
		return resp
	}

	// Success - even if result is a Zylisp error, it's in the value field
	resp.Value = result
	resp.Output = output
	resp.Status = []string{"done"}
	return resp
}

// handleLoadFile processes the "load-file" operation.
func (h *Handler) handleLoadFile(req *protocol.Message, resp *protocol.Message) *protocol.Message {
	// Get file path from either 'file' or 'file-path' field
	var filePath string
	if req.Data != nil {
		if fp, ok := req.Data["file"].(string); ok {
			filePath = fp
		} else if fp, ok := req.Data["file-path"].(string); ok {
			filePath = fp
		}
	}

	if filePath == "" {
		resp.Status = []string{"error"}
		resp.ProtocolError = "load-file operation requires 'file' or 'file-path' in data field"
		return resp
	}

	// Read the file
	code, err := os.ReadFile(filePath)
	if err != nil {
		resp.Status = []string{"error"}
		resp.ProtocolError = fmt.Sprintf("failed to read file: %v", err)
		return resp
	}

	// Evaluate the file contents
	result, output, err := h.evaluator(string(code))
	if err != nil {
		// Catastrophic error
		resp.Status = []string{"error"}
		resp.ProtocolError = fmt.Sprintf("evaluator error: %v", err)
		return resp
	}

	// Success
	resp.Value = result
	resp.Output = output
	resp.Status = []string{"done"}
	return resp
}

// handleDescribe processes the "describe" operation.
// It returns information about the server's capabilities.
func (h *Handler) handleDescribe(req *protocol.Message, resp *protocol.Message) *protocol.Message {
	resp.Status = []string{"done"}
	resp.Data = map[string]interface{}{
		"versions": map[string]interface{}{
			"zylisp":   "0.1.0",
			"protocol": "0.1.0",
		},
		"ops": []string{
			"eval",
			"load-file",
			"describe",
			"interrupt",
		},
		"transports": []string{
			"in-process",
			"unix",
			"tcp",
		},
	}
	return resp
}

// handleInterrupt processes the "interrupt" operation.
// This is a stub for now - full implementation requires context cancellation.
func (h *Handler) handleInterrupt(req *protocol.Message, resp *protocol.Message) *protocol.Message {
	resp.Status = []string{"error"}
	resp.ProtocolError = "interrupt operation not yet fully implemented"
	return resp
}
