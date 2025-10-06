package inprocess

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
	case "(error \"test error\")":
		// Return error as data (Zylisp errors-as-data)
		return map[string]interface{}{
			"error": "test error",
			"type":  "user-error",
		}, "", nil
	case "(catastrophic)":
		// Return actual Go error (catastrophic failure)
		return nil, "", fmt.Errorf("catastrophic failure")
	default:
		return code, "", nil
	}
}

func TestInProcessServerClient(t *testing.T) {
	// Create server
	server := NewServer(mockEvaluator)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Create client
	client := NewClient()
	client.SetServer(server)

	// Connect client
	if err := client.Connect(context.Background(), server); err != nil {
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

	// Test error-as-data
	t.Run("error as data", func(t *testing.T) {
		result, err := client.Eval(context.Background(), "(error \"test error\")")
		if err != nil {
			t.Fatalf("Eval failed: %v", err)
		}

		// The error should be in the Value field, not as a Go error
		errMap, ok := result.Value.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected error value to be map, got %T", result.Value)
		}

		if errMap["error"] != "test error" {
			t.Errorf("Expected error message 'test error', got %v", errMap["error"])
		}

		// Status should still be "done" because the protocol succeeded
		if len(result.Status) == 0 || result.Status[0] != "done" {
			t.Errorf("Expected status 'done', got %v", result.Status)
		}
	})
}

func TestMultipleClients(t *testing.T) {
	// Create server
	server := NewServer(mockEvaluator)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Create multiple clients
	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		client := NewClient()
		client.SetServer(server)
		if err := client.Connect(context.Background(), server); err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		clients[i] = client
		defer client.Close()
	}

	// Send requests from all clients concurrently
	results := make(chan *Result, 5)
	errors := make(chan error, 5)

	for i, client := range clients {
		go func(i int, c *Client) {
			result, err := c.Eval(context.Background(), "(+ 1 2)")
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i, client)
	}

	// Collect results
	for i := 0; i < 5; i++ {
		select {
		case result := <-results:
			if result.Value != float64(3) {
				t.Errorf("Client %d: expected value 3, got %v", i, result.Value)
			}
		case err := <-errors:
			t.Errorf("Client %d: eval failed: %v", i, err)
		case <-time.After(time.Second):
			t.Fatalf("Timeout waiting for result from client %d", i)
		}
	}
}

func TestServerShutdown(t *testing.T) {
	// Create server
	server := NewServer(mockEvaluator)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Create client
	client := NewClient()
	client.SetServer(server)
	if err := client.Connect(context.Background(), server); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Send a request
	result, err := client.Eval(context.Background(), "(+ 1 2)")
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	if result.Value != float64(3) {
		t.Errorf("Expected value 3, got %v", result.Value)
	}

	// Shut down server
	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()

	if err := server.Stop(stopCtx); err != nil && err != context.Canceled {
		t.Errorf("Server stop failed: %v", err)
	}
}

func TestClientContextCancellation(t *testing.T) {
	// Create server that takes a long time to respond
	slowEvaluator := func(code string) (interface{}, string, error) {
		time.Sleep(2 * time.Second)
		return "slow", "", nil
	}

	server := NewServer(slowEvaluator)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Start(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	// Create client
	client := NewClient()
	client.SetServer(server)
	if err := client.Connect(context.Background(), server); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Create a context with short timeout
	evalCtx, evalCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer evalCancel()

	// Try to eval - should timeout
	_, err := client.Eval(evalCtx, "(+ 1 2)")
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}
