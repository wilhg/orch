//go:build !mcp

package mcpclient

import (
	"context"
	"errors"
)

// Client defines minimal MCP client capabilities we need.
type Client interface {
	Handshake(ctx context.Context) error
	ListTools(ctx context.Context) ([]ToolDescriptor, error)
	CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error)
	ListResources(ctx context.Context) ([]ResourceDescriptor, error)
	Close() error
}

// ToolDescriptor is a subset of MCP tool schema.
type ToolDescriptor struct {
	Name         string
	Description  string
	InputSchema  []byte
	OutputSchema []byte
}

// ResourceDescriptor describes an MCP resource.
type ResourceDescriptor struct {
	URI         string
	Description string
}

// New returns a stub client which reports not supported unless built with mcp tag.
func New(_ context.Context, _ string, _ ...Option) (Client, error) {
	return &noopClient{}, nil
}

type Option func(*config)

type config struct{}

type noopClient struct{}

func (noopClient) Handshake(context.Context) error {
	return errors.New("mcp not enabled in this build")
}
func (noopClient) ListTools(context.Context) ([]ToolDescriptor, error) {
	return nil, errors.New("mcp not enabled in this build")
}
func (noopClient) CallTool(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, errors.New("mcp not enabled in this build")
}
func (noopClient) ListResources(context.Context) ([]ResourceDescriptor, error) {
	return nil, errors.New("mcp not enabled in this build")
}
func (noopClient) Close() error { return nil }
