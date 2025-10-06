package protocol

import (
	"encoding/json"
	"io"
)

// JSONCodec implements the Codec interface using newline-delimited JSON encoding.
// It uses encoding/json's Encoder and Decoder which automatically handle framing.
type JSONCodec struct {
	rw      io.ReadWriteCloser
	encoder *json.Encoder
	decoder *json.Decoder
}

// NewJSONCodec creates a new JSON codec that reads from and writes to the given ReadWriteCloser.
func NewJSONCodec(rw io.ReadWriteCloser) *JSONCodec {
	return &JSONCodec{
		rw:      rw,
		encoder: json.NewEncoder(rw),
		decoder: json.NewDecoder(rw),
	}
}

// Encode encodes a message to JSON and writes it to the underlying writer.
// The encoder automatically adds a newline after each message.
func (c *JSONCodec) Encode(msg *Message) error {
	return c.encoder.Encode(msg)
}

// Decode reads and decodes a JSON message from the underlying reader.
// The decoder automatically handles newline-delimited JSON.
func (c *JSONCodec) Decode(msg *Message) error {
	return c.decoder.Decode(msg)
}

// Close closes the underlying ReadWriteCloser.
func (c *JSONCodec) Close() error {
	return c.rw.Close()
}
