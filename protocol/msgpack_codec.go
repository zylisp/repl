package protocol

import (
	"io"
)

// MessagePackCodec implements the Codec interface using MessagePack encoding.
// This is a placeholder implementation for future binary efficiency optimization.
// When implemented, it will use github.com/vmihailenco/msgpack/v5.
type MessagePackCodec struct {
	rw io.ReadWriteCloser
}

// NewMessagePackCodec creates a new MessagePack codec.
// This is currently a placeholder and will panic if used.
func NewMessagePackCodec(rw io.ReadWriteCloser) *MessagePackCodec {
	return &MessagePackCodec{
		rw: rw,
	}
}

// Encode is not yet implemented.
func (c *MessagePackCodec) Encode(msg *Message) error {
	panic("MessagePack codec not yet implemented")
}

// Decode is not yet implemented.
func (c *MessagePackCodec) Decode(msg *Message) error {
	panic("MessagePack codec not yet implemented")
}

// Close closes the underlying ReadWriteCloser.
func (c *MessagePackCodec) Close() error {
	return c.rw.Close()
}
