package client

import (
	"testing"

	"github.com/zylisp/repl/server"
)

func TestClientSend(t *testing.T) {
	srv := server.NewServer()
	client := NewClient(srv)

	result, err := client.Send("(+ 1 2)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "3" {
		t.Errorf("got %q, want \"3\"", result)
	}
}

func TestClientReset(t *testing.T) {
	srv := server.NewServer()
	client := NewClient(srv)

	// Define a variable
	client.Send("(define x 42)")

	// Reset
	client.Reset()

	// Variable should be undefined
	_, err := client.Send("x")
	if err == nil {
		t.Error("expected error after reset, got nil")
	}
}
