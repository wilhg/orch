//go:build mcp

package mcpserver

import (
	"context"
	"net"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wilhg/orch/pkg/agent"
)

type echoTool struct{}

func (echoTool) Describe() agent.ToolDescriptor {
	return agent.ToolDescriptor{
		Name:         "echo",
		Description:  "echo tool",
		InputSchema:  []byte(`{"type":"object","properties":{"msg":{"type":"string"}},"required":["msg"],"additionalProperties":false}`),
		OutputSchema: []byte(`{"type":"object","properties":{"echo":{"type":"string"}},"required":["echo"],"additionalProperties":false}`),
	}
}

func (echoTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	v, _ := args["msg"].(string)
	return map[string]any{"echo": v}, nil
}

func TestMCPServer_ClientHandshakeAndCall(t *testing.T) {
	// register tool
	_ = agent.RegisterTool(echoTool{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// start server on an in-memory pipe
	srv, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	c1, c2 := net.Pipe()
	go func() { _ = srv.ServeConn(ctx, c1) }()

	// client side using SDK directly for the test
	cli := mcp.NewClient()
	go func() { _ = cli.Serve(ctx, c2) }()

	if err := cli.Handshake(ctx); err != nil {
		t.Fatal(err)
	}
	tools, err := cli.ListTools(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) == 0 {
		t.Fatal("no tools listed")
	}

	out, err := cli.CallTool(ctx, "echo", map[string]any{"msg": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out["echo"] != "hi" {
		t.Fatalf("unexpected echo: %v", out)
	}
}
