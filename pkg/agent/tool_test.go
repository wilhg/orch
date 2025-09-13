package agent

import (
	"context"
	"encoding/json"
	"testing"
)

type echoTool struct{}

func (echoTool) Describe() ToolDescriptor {
	in := []byte(`{"type":"object","properties":{"msg":{"type":"string"}},"required":["msg"],"additionalProperties":false}`)
	out := []byte(`{"type":"object","properties":{"echo":{"type":"string"}},"required":["echo"],"additionalProperties":false}`)
	return ToolDescriptor{
		Name:         "echo",
		Description:  "echoes a message",
		InputSchema:  in,
		OutputSchema: out,
		Permissions:  []ToolPermission{{Name: "cpu", Description: "local compute"}},
		SideEffects:  []SideEffectCategory{SideEffectProcess},
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

func TestDescribeTool(t *testing.T) {
	d := DescribeTool(echoTool{})
	if d.Name != "echo" {
		t.Fatalf("name=%s", d.Name)
	}
	if len(d.InputSchema) == 0 || len(d.OutputSchema) == 0 {
		t.Fatal("schemas missing")
	}
	if len(d.Permissions) != 1 || d.Permissions[0].Name != "cpu" {
		t.Fatalf("permissions=%v", d.Permissions)
	}
	if len(d.SideEffects) != 1 || d.SideEffects[0] != SideEffectProcess {
		t.Fatalf("side_effects=%v", d.SideEffects)
	}
}
