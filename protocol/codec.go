package protocol

import (
	"fmt"
	"io"
)

// Codec defines the interface for encoding and decoding protocol messages.
// Implementations handle the serialization format (JSON, MessagePack, etc.)
// and message framing over the underlying transport.
type Codec interface {
	// Encode writes a message to the underlying writer
	Encode(msg *Message) error

	// Decode reads a message from the underlying reader
	Decode(msg *Message) error

	// Close closes the codec and its underlying resources
	Close() error
}

// NewCodec creates a codec based on the specified format.
// Supported formats: "json", "msgpack"
// The rw parameter is the underlying transport connection.
func NewCodec(format string, rw io.ReadWriteCloser) (Codec, error) {
	switch format {
	case "json":
		return NewJSONCodec(rw), nil
	case "msgpack":
		return NewMessagePackCodec(rw), nil
	default:
		return nil, fmt.Errorf("unsupported codec format: %s", format)
	}
}
