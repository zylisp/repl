package tcp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockEvaluator is a simple evaluator for testing
func mockEvaluator(code string) (interface{}, string, error) {
	switch code {
	case "(+ 1 2)":
		return float64(3), "", nil
	case "(println \"hello\")":
		return nil, "hello\n", nil
	default:
		return code, "", nil
	}
}

func TestTCPServerClient(t *testing.T) {
	// Create server on localhost with random port
	server := NewServer(":0", "json", mockEvaluator)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual address the server is listening on
	addr := server.Addr()

	// Create client
	client := NewClient("json")
	if err := client.Connect(context.Background(), addr, "json"); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Test basic eval
	t.Run("basic eval", func(t *testing.T) {
		result, err := client.Eval(context.Background(), "(+ 1 2)")
		if err != nil {
			t.Fatalf("Eval failed: %v", err)
		}

		if result.Value != float64(3) {
			t.Errorf("Expected value 3, got %v", result.Value)
		}

		if len(result.Status) == 0 || result.Status[0] != "done" {
			t.Errorf("Expected status 'done', got %v", result.Status)
		}
	})

	// Test eval with output
	t.Run("eval with output", func(t *testing.T) {
		result, err := client.Eval(context.Background(), "(println \"hello\")")
		if err != nil {
			t.Fatalf("Eval failed: %v", err)
		}

		if result.Output != "hello\n" {
			t.Errorf("Expected output 'hello\\n', got %q", result.Output)
		}

		if len(result.Status) == 0 || result.Status[0] != "done" {
			t.Errorf("Expected status 'done', got %v", result.Status)
		}
	})

	// Test server shutdown
	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()

	if err := server.Stop(stopCtx); err != nil && err != context.Canceled {
		t.Errorf("Server stop failed: %v", err)
	}
}

func TestTCPMultipleClients(t *testing.T) {
	// Create server
	server := NewServer(":0", "json", mockEvaluator)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Get server address
	addr := server.Addr()

	// Create multiple clients
	numClients := 5
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		client := NewClient("json")
		if err := client.Connect(context.Background(), addr, "json"); err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()
	}

	// Send requests from all clients concurrently
	results := make(chan *Result, numClients)
	errors := make(chan error, numClients)

	for i, client := range clients {
		go func(i int, c *Client) {
			result, err := c.Eval(context.Background(), "(+ 1 2)")
			if err != nil {
				errors <- fmt.Errorf("client %d: %w", i, err)
				return
			}
			results <- result
		}(i, client)
	}

	// Collect results
	for i := 0; i < numClients; i++ {
		select {
		case result := <-results:
			if result.Value != float64(3) {
				t.Errorf("Expected value 3, got %v", result.Value)
			}
		case err := <-errors:
			t.Errorf("Eval failed: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatalf("Timeout waiting for result from client %d", i)
		}
	}
}
