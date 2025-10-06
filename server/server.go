package server

import (
	"fmt"

	"github.com/zylisp/lang/interpreter"
	"github.com/zylisp/lang/parser"
)

// Server represents a REPL server
type Server struct {
	env *interpreter.Env
}

// NewServer creates a new REPL server
func NewServer() *Server {
	env := interpreter.NewEnv(nil)
	interpreter.LoadPrimitives(env)

	return &Server{env: env}
}

// Eval evaluates a Zylisp expression and returns the result as a string
func (s *Server) Eval(source string) (string, error) {
	// Tokenize
	tokens, err := parser.Tokenize(source)
	if err != nil {
		return "", fmt.Errorf("tokenize error: %w", err)
	}

	// Parse
	expr, err := parser.Read(tokens)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	// Evaluate
	result, err := interpreter.Eval(expr, s.env)
	if err != nil {
		return "", fmt.Errorf("eval error: %w", err)
	}

	return result.String(), nil
}

// Reset clears the environment and reloads primitives
func (s *Server) Reset() {
	s.env = interpreter.NewEnv(nil)
	interpreter.LoadPrimitives(s.env)
}
