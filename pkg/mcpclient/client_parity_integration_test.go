//go:build mcp

package mcpclient

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wilhg/orch/pkg/agent"
)

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

func setupLoopback(t *testing.T, allowed map[string]bool) (*mcp.ClientSession, func()) {
	t.Helper()
	_ = agent.RegisterTool(sumTool{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Build server
	s := mcp.NewServer(&mcp.Implementation{Name: "orch-test", Version: "dev"}, nil)
	agent.RangeTools(func(name string, tl agent.Tool) {
		d := tl.Describe()
		var inSch jsonschema.Schema
		_ = json.Unmarshal(d.InputSchema, &inSch)
		mcp.AddTool(s, &mcp.Tool{Name: d.Name, Description: d.Description, InputSchema: &inSch}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, map[string]any, error) {
			out, err := agent.SafeInvoke(ctx, tl, args, allowed, agent.JSONSchemaValidator)
			if err != nil {
				// Return as tool error result
				return &mcp.CallToolResult{IsError: true}, nil, nil
			}
			return nil, out, nil
		})
	})

	srvT, cliT := mcp.NewInMemoryTransports()
	go func() { _ = s.Run(ctx, srvT) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	cleanup := func() {
		_ = session.Close()
		cancel()
	}
	return session, cleanup
}

func TestMCPClient_Handshake_List_Call_Success_And_InvalidInput(t *testing.T) {
	sess, cleanup := setupLoopback(t, map[string]bool{"cpu": true})
	defer cleanup()

	ctx := context.Background()
	tools, err := sess.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("no tools listed")
	}

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "sum", Arguments: map[string]any{"a": 1.0, "b": 2.0}})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	m, _ := res.StructuredContent.(map[string]any)
	if _, ok := m["sum"]; !ok {
		t.Fatalf("missing sum in %v", res.StructuredContent)
	}

	// Invalid input is rejected by SDK validation against InputSchema and surfaces as an error
	if _, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "sum", Arguments: map[string]any{"a": "x", "b": 2.0}}); err == nil {
		t.Fatalf("expected validation error from CallTool")
	}
}

func TestMCPClient_CallTool_ForbiddenPermission(t *testing.T) {
	sess, cleanup := setupLoopback(t, map[string]bool{})
	defer cleanup()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: "sum", Arguments: map[string]any{"a": 1.0, "b": 2.0}})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected permission error result")
	}
}
