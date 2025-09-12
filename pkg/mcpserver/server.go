package mcpserver

import (
	"context"
	"errors"

	"github.com/wilhg/orch/pkg/agent"
)

// Server is a placeholder MCP server when the mcp build tag is not set.
// It allows the rest of the repo to compile without the SDK.
type Server struct{}

type Option func(*Server)

// New creates a new MCP server (no-op without mcp tag).
func New(_ context.Context, _ ...Option) (*Server, error) { return &Server{}, nil }

// RegisterFromRegistry is a no-op that would export tools to the MCP server.
func (s *Server) RegisterFromRegistry(_ map[string]bool, _ agent.ValidateFunc) error { return nil }

// Serve starts the MCP server (no-op without mcp tag).
func (s *Server) Serve(_ context.Context, _ string) error {
	return errors.New("mcp server not enabled in this build")
}
