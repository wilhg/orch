//go:build mcp

package mcpserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
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
		Permissions:  []agent.ToolPermission{{Name: "cpu"}},
	}
}

func (echoTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	b, _ := json.Marshal(args)
	var m struct {
		Msg string `json:"msg"`
	}
	_ = json.Unmarshal(b, &m)
	return map[string]any{"echo": m.Msg}, nil
}

func TestMCPServer_ClientHandshakeAndCall(t *testing.T) {
	_ = agent.RegisterTool(echoTool{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s := mcp.NewServer(&mcp.Implementation{Name: "orch-test", Version: "dev"}, nil)
	agent.RangeTools(func(name string, tl agent.Tool) {
		d := tl.Describe()
		var inSch jsonschema.Schema
		_ = json.Unmarshal(d.InputSchema, &inSch)
		mcp.AddTool(s, &mcp.Tool{Name: d.Name, Description: d.Description, InputSchema: &inSch}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, map[string]any, error) {
			out, err := agent.SafeInvoke(ctx, tl, args, map[string]bool{"cpu": true}, agent.JSONSchemaValidator)
			if err != nil {
				return nil, nil, err
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
	defer func() { _ = session.Close() }()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("no tools listed")
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"msg": "hi"}})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	m, _ := res.StructuredContent.(map[string]any)
	if m["echo"] != "hi" {
		t.Fatalf("unexpected content: %#v", res.StructuredContent)
	}
}
