//go:build mcp

package mcpserver

import (
	"context"
	"encoding/json"
	"errors"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wilhg/orch/pkg/agent"
)

type Server struct {
	srv *mcp.Server
}

type Option func(*Server)

func New(ctx context.Context, _ ...Option) (*Server, error) {
	return &Server{srv: mcp.NewServer(&mcp.Implementation{Name: "orch", Version: "dev"}, nil)}, nil
}

// RegisterFromRegistry exports local tools to the MCP server.
func (s *Server) RegisterFromRegistry(allowed map[string]bool, validate agent.ValidateFunc) error {
	agent.RangeTools(func(name string, t agent.Tool) {
		desc := t.Describe()
		var inSch jsonschema.Schema
		_ = json.Unmarshal(desc.InputSchema, &inSch)
		mcp.AddTool(s.srv, &mcp.Tool{
			Name:        desc.Name,
			Description: desc.Description,
			InputSchema: &inSch,
		}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, map[string]any, error) {
			out, err := agent.SafeInvoke(ctx, t, args, allowed, validate)
			if err != nil {
				return nil, nil, err
			}
			return nil, out, nil
		})
	})
	return nil
}

// Serve is not implemented for raw net.Conn in this minimal wrapper.
func (s *Server) Serve(_ context.Context, _ string) error {
	return errors.New("Serve not implemented; use mcp.Server with a concrete Transport")
}

// ServeConn is not implemented for raw net.Conn in this minimal wrapper.
func (s *Server) ServeConn(_ context.Context, _ any) error {
	return errors.New("ServeConn not implemented; use mcp.Server with a concrete Transport")
}
