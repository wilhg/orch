//go:build mcp

package mcpserver

import (
	"context"
	"net"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wilhg/orch/pkg/agent"
)

type Server struct {
	srv *mcp.Server
}

type Option func(*Server)

func New(ctx context.Context, _ ...Option) (*Server, error) {
	return &Server{srv: mcp.NewServer()}, nil
}

// RegisterFromRegistry exports local tools to the MCP server.
func (s *Server) RegisterFromRegistry(allowed map[string]bool, validate agent.ValidateFunc) error {
	agent.RangeTools(func(name string, t agent.Tool) {
		desc := t.Describe()
		// map agent.Tool to MCP tool
		_ = s.srv.RegisterTool(mcp.Tool{
			Name:         desc.Name,
			Description:  desc.Description,
			InputSchema:  desc.InputSchema,
			OutputSchema: desc.OutputSchema,
			Handler: func(ctx context.Context, args map[string]any) (map[string]any, error) {
				return agent.SafeInvoke(ctx, t, args, allowed, validate)
			},
		})
	})
	return nil
}

// Serve listens on a net.Conn (JSON-RPC over WS/HTTP is handled outside in SDK examples).
// Here we assume conn is already established; for simplicity expose a tcp listen addr and accept one connection.
func (s *Server) Serve(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go func() { _ = s.srv.Serve(ctx, conn) }()
	}
}

// ServeConn serves a single pre-established connection.
func (s *Server) ServeConn(ctx context.Context, conn net.Conn) error {
	return s.srv.Serve(ctx, conn)
}
