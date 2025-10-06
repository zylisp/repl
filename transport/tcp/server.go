package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/zylisp/repl/operations"
	"github.com/zylisp/repl/protocol"
)

// Server implements a TCP REPL server.
type Server struct {
	addr     string
	codec    string
	handler  *operations.Handler
	listener net.Listener
	conns    map[net.Conn]bool
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewServer creates a new TCP REPL server.
func NewServer(addr string, codec string, evaluator operations.EvaluatorFunc) *Server {
	return &Server{
		addr:    addr,
		codec:   codec,
		handler: operations.NewHandler(evaluator),
		conns:   make(map[net.Conn]bool),
	}
}

// Start begins listening for connections on the TCP port.
func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create listener
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on tcp: %w", err)
	}
	s.listener = listener

	// Accept connections in the background
	s.wg.Add(1)
	go s.acceptLoop()

	// Wait for context cancellation
	<-s.ctx.Done()
	return s.ctx.Err()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	// Close the listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all connections
	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.conns = make(map[net.Conn]bool)
	s.mu.Unlock()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addr returns the TCP address.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				// Log error but continue accepting
				continue
			}
		}

		// Track connection
		s.mu.Lock()
		s.conns[conn] = true
		s.mu.Unlock()

		// Handle connection in a goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection processes requests from a single connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	// Create codec for this connection
	codec, err := protocol.NewCodec(s.codec, conn)
	if err != nil {
		return
	}

	// Process messages
	for {
		// Read request
		req := &protocol.Message{}
		if err := codec.Decode(req); err != nil {
			return
		}

		// Handle request
		resp := s.handler.Handle(req)

		// Send response
		if err := codec.Encode(resp); err != nil {
			return
		}
	}
}
