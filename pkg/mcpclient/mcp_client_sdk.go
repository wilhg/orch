//go:build mcp

package mcpclient

import (
	"context"
	"net/url"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type sdkClient struct {
	c *mcp.Client
}

// New creates an MCP client using the official SDK. addr is a ws:// or wss:// URL.
func New(ctx context.Context, addr string, _ ...Option) (Client, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	c, err := mcp.Dial(ctx, u.String())
	if err != nil {
		return nil, err
	}
	return &sdkClient{c: c}, nil
}

func (s *sdkClient) Handshake(ctx context.Context) error { return s.c.Handshake(ctx) }

func (s *sdkClient) ListTools(ctx context.Context) ([]ToolDescriptor, error) {
	ts, err := s.c.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ToolDescriptor, 0, len(ts))
	for _, t := range ts {
		out = append(out, ToolDescriptor{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema, OutputSchema: t.OutputSchema})
	}
	return out, nil
}

func (s *sdkClient) CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return s.c.CallTool(ctx, name, args)
}

func (s *sdkClient) ListResources(ctx context.Context) ([]ResourceDescriptor, error) {
	rs, err := s.c.ListResources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ResourceDescriptor, 0, len(rs))
	for _, r := range rs {
		out = append(out, ResourceDescriptor{URI: r.URI, Description: r.Description})
	}
	return out, nil
}

func (s *sdkClient) Close() error { return s.c.Close() }
