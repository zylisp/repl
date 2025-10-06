package protocol

import (
	"bytes"
	"io"
	"testing"
)

// mockReadWriteCloser is a simple wrapper around bytes.Buffer for testing
type mockReadWriteCloser struct {
	*bytes.Buffer
}

func (m *mockReadWriteCloser) Close() error {
	return nil
}

func newMockReadWriteCloser() *mockReadWriteCloser {
	return &mockReadWriteCloser{Buffer: &bytes.Buffer{}}
}

func TestJSONCodec_EncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "eval request",
			msg: &Message{
				Op:   "eval",
				ID:   "1",
				Code: "(+ 1 2)",
			},
		},
		{
			name: "eval response with value",
			msg: &Message{
				ID:     "1",
				Value:  float64(3), // JSON numbers decode as float64
				Status: []string{"done"},
			},
		},
		{
			name: "error response",
			msg: &Message{
				ID:            "2",
				ProtocolError: "unknown operation",
				Status:        []string{"error"},
			},
		},
		{
			name: "response with output",
			msg: &Message{
				ID:     "3",
				Value:  nil,
				Output: "hello\n",
				Status: []string{"done"},
			},
		},
		{
			name: "describe response with data",
			msg: &Message{
				ID:     "4",
				Status: []string{"done"},
				Data: map[string]interface{}{
					"versions": map[string]interface{}{
						"zylisp":   "0.1.0",
						"protocol": "0.1.0",
					},
					"ops": []interface{}{"eval", "load-file", "describe"},
				},
			},
		},
		{
			name: "message with all fields",
			msg: &Message{
				Op:      "eval",
				ID:      "5",
				Session: "session-123",
				Code:    "(println \"test\")",
				Status:  []string{"done"},
				Value:   nil,
				Output:  "test\n",
				Data: map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock ReadWriteCloser
			buf := newMockReadWriteCloser()

			// Create codec and encode message
			encoder := NewJSONCodec(buf)
			if err := encoder.Encode(tt.msg); err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			// Create a new codec for decoding (simulating network transmission)
			decoder := NewJSONCodec(buf)
			decoded := &Message{}
			if err := decoder.Decode(decoded); err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Compare key fields
			if decoded.Op != tt.msg.Op {
				t.Errorf("Op mismatch: got %q, want %q", decoded.Op, tt.msg.Op)
			}
			if decoded.ID != tt.msg.ID {
				t.Errorf("ID mismatch: got %q, want %q", decoded.ID, tt.msg.ID)
			}
			if decoded.Code != tt.msg.Code {
				t.Errorf("Code mismatch: got %q, want %q", decoded.Code, tt.msg.Code)
			}
			if decoded.Output != tt.msg.Output {
				t.Errorf("Output mismatch: got %q, want %q", decoded.Output, tt.msg.Output)
			}
			if decoded.ProtocolError != tt.msg.ProtocolError {
				t.Errorf("ProtocolError mismatch: got %q, want %q", decoded.ProtocolError, tt.msg.ProtocolError)
			}

			// Compare Status slice
			if len(decoded.Status) != len(tt.msg.Status) {
				t.Errorf("Status length mismatch: got %d, want %d", len(decoded.Status), len(tt.msg.Status))
			} else {
				for i, s := range tt.msg.Status {
					if decoded.Status[i] != s {
						t.Errorf("Status[%d] mismatch: got %q, want %q", i, decoded.Status[i], s)
					}
				}
			}
		})
	}
}

func TestJSONCodec_MultipleMessages(t *testing.T) {
	buf := newMockReadWriteCloser()
	codec := NewJSONCodec(buf)

	// Encode multiple messages
	messages := []*Message{
		{Op: "eval", ID: "1", Code: "(+ 1 2)"},
		{ID: "1", Value: float64(3), Status: []string{"done"}},
		{Op: "eval", ID: "2", Code: "(* 3 4)"},
		{ID: "2", Value: float64(12), Status: []string{"done"}},
	}

	for _, msg := range messages {
		if err := codec.Encode(msg); err != nil {
			t.Fatalf("Failed to encode message: %v", err)
		}
	}

	// Decode the same messages
	for i, expected := range messages {
		decoded := &Message{}
		if err := codec.Decode(decoded); err != nil {
			t.Fatalf("Failed to decode message %d: %v", i, err)
		}

		if decoded.Op != expected.Op {
			t.Errorf("Message %d Op mismatch: got %q, want %q", i, decoded.Op, expected.Op)
		}
		if decoded.ID != expected.ID {
			t.Errorf("Message %d ID mismatch: got %q, want %q", i, decoded.ID, expected.ID)
		}
	}
}

func TestJSONCodec_DecodeError(t *testing.T) {
	// Create a buffer with invalid JSON
	buf := &mockReadWriteCloser{Buffer: bytes.NewBufferString("{invalid json\n")}
	codec := NewJSONCodec(buf)

	msg := &Message{}
	err := codec.Decode(msg)
	if err == nil {
		t.Fatal("Expected decode error for invalid JSON, got nil")
	}
}

func TestJSONCodec_DecodeEOF(t *testing.T) {
	// Create an empty buffer
	buf := newMockReadWriteCloser()
	codec := NewJSONCodec(buf)

	msg := &Message{}
	err := codec.Decode(msg)
	if err != io.EOF {
		t.Fatalf("Expected EOF error, got: %v", err)
	}
}

func TestJSONCodec_Close(t *testing.T) {
	buf := newMockReadWriteCloser()
	codec := NewJSONCodec(buf)

	if err := codec.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
