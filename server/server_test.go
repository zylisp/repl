package server

import (
	"testing"
)

func TestServerBasicEval(t *testing.T) {
	server := NewServer()

	tests := []struct {
		input    string
		expected string
	}{
		{"42", "42"},
		{"(+ 1 2)", "3"},
		{"(* 2 3)", "6"},
		{`"hello"`, `"hello"`},
		{"true", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := server.Eval(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestServerDefine(t *testing.T) {
	server := NewServer()

	// Define a variable
	_, err := server.Eval("(define x 42)")
	if err != nil {
		t.Fatalf("define error: %v", err)
	}

	// Use the variable
	result, err := server.Eval("x")
	if err != nil {
		t.Fatalf("lookup error: %v", err)
	}

	if result != "42" {
		t.Errorf("got %q, want \"42\"", result)
	}
}

func TestServerLambda(t *testing.T) {
	server := NewServer()

	// Define a function
	_, err := server.Eval("(define square (lambda (x) (* x x)))")
	if err != nil {
		t.Fatalf("define error: %v", err)
	}

	// Call the function
	result, err := server.Eval("(square 5)")
	if err != nil {
		t.Fatalf("call error: %v", err)
	}

	if result != "25" {
		t.Errorf("got %q, want \"25\"", result)
	}
}

func TestServerReset(t *testing.T) {
	server := NewServer()

	// Define a variable
	server.Eval("(define x 42)")

	// Reset the server
	server.Reset()

	// Variable should be undefined now
	_, err := server.Eval("x")
	if err == nil {
		t.Error("expected error after reset, got nil")
	}
}

func TestServerErrors(t *testing.T) {
	server := NewServer()

	tests := []string{
		"(+",      // unclosed paren
		"(+ 1 x)", // undefined variable
		"(1 2 3)", // not a function
		"(/ 1 0)", // division by zero
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := server.Eval(input)
			if err == nil {
				t.Errorf("expected error for %q, got nil", input)
			}
		})
	}
}
