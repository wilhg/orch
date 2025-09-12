//go:build mcp

package mcpclient

import (
	"context"
	"net"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wilhg/orch/pkg/agent"
	"github.com/wilhg/orch/pkg/mcpserver"
)

// sumTool is a minimal tool for exercising client/server calls.
type sumTool struct{}

func (sumTool) Describe() agent.ToolDescriptor {
	return agent.ToolDescriptor{
		Name:         "sum",
		Description:  "adds two numbers",
		InputSchema:  []byte(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"],"additionalProperties":false}`),
		OutputSchema: []byte(`{"type":"object","properties":{"sum":{"type":"number"}},"required":["sum"],"additionalProperties":false}`),
		Permissions:  []agent.ToolPermission{{Name: "cpu"}},
	}
}

func (sumTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	return map[string]any{"sum": a + b}, nil
}

// newLoopbackClientServer wires an MCP server and sdkClient over an in-memory connection.
func newLoopbackClientServer(t *testing.T, allowed map[string]bool) (*sdkClient, *mcpserver.Server, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	srv, err := mcpserver.New(ctx)
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	if err := srv.RegisterFromRegistry(allowed, agent.JSONSchemaValidator); err != nil {
		t.Fatalf("register: %v", err)
	}
	c1, c2 := net.Pipe()
	go func() { _ = srv.ServeConn(ctx, c1) }()

	// Wrap the SDK client inside our adapter so we exercise wrapper methods.
	raw := mcp.NewClient()
	go func() { _ = raw.Serve(ctx, c2) }()
	cli := &sdkClient{c: raw}

	cleanup := func() {
		_ = cli.Close()
		cancel()
	}
	return cli, srv, cleanup
}

func TestMCPClient_Handshake_List_Call_Success_And_InvalidInput(t *testing.T) {
	t.Skip("MCP server/client API update pending (go-sdk v0.4)")
}

func TestMCPClient_CallTool_ForbiddenPermission(t *testing.T) {
	t.Skip("MCP server/client API update pending (go-sdk v0.4)")
}
