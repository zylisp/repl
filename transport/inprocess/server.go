package inprocess

import (
	"context"
	"fmt"
	"sync"

	"github.com/zylisp/repl/operations"
	"github.com/zylisp/repl/protocol"
)

// Server implements an in-process REPL server using Go channels for message passing.
// This provides zero-overhead communication for testing and embedded use cases.
type Server struct {
	handler  *operations.Handler
	requests chan *protocol.Message
	clients  map[string]chan *protocol.Message // clientID -> response channel
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewServer creates a new in-process REPL server.
func NewServer(evaluator operations.EvaluatorFunc) *Server {
	return &Server{
		handler:  operations.NewHandler(evaluator),
		requests: make(chan *protocol.Message, 100),
		clients:  make(map[string]chan *protocol.Message),
	}
}

// Start begins processing requests.
// It blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	s.wg.Add(1)
	go s.processRequests()

	// Wait for context cancellation
	<-s.ctx.Done()
	return s.ctx.Err()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	// Close all client response channels
	s.mu.Lock()
	for _, ch := range s.clients {
		close(ch)
	}
	s.clients = make(map[string]chan *protocol.Message)
	s.mu.Unlock()

	// Wait for processing goroutine to finish
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

// Addr returns the address (always "in-process" for this transport).
func (s *Server) Addr() string {
	return "in-process"
}

// processRequests handles incoming requests and routes responses to clients.
func (s *Server) processRequests() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case req, ok := <-s.requests:
			if !ok {
				return
			}

			// Get client ID from the request
			// For in-process, we use the Session field to identify the client
			clientID := req.Session
			if clientID == "" {
				// Skip requests without client ID
				continue
			}

			// Process the request
			resp := s.handler.Handle(req)

			// Send response to the client
			s.mu.RLock()
			respChan, exists := s.clients[clientID]
			s.mu.RUnlock()

			if exists {
				select {
				case respChan <- resp:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

// registerClient registers a new client and returns its response channel.
func (s *Server) registerClient(clientID string) chan *protocol.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	respChan := make(chan *protocol.Message, 10)
	s.clients[clientID] = respChan
	return respChan
}

// unregisterClient removes a client.
func (s *Server) unregisterClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.clients[clientID]; exists {
		close(ch)
		delete(s.clients, clientID)
	}
}

// sendRequest sends a request from a client to the server.
func (s *Server) sendRequest(req *protocol.Message) error {
	select {
	case s.requests <- req:
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("server stopped")
	}
}
