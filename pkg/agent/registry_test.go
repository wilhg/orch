package agent

import (
	"context"
	"testing"
)

type testTool struct{}

func (testTool) Describe() ToolDescriptor {
	return ToolDescriptor{
		Name:         "sum",
		InputSchema:  []byte(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"],"additionalProperties":false}`),
		OutputSchema: []byte(`{"type":"object","properties":{"sum":{"type":"number"}},"required":["sum"],"additionalProperties":false}`),
		Permissions:  []ToolPermission{{Name: "cpu"}},
		SideEffects:  []SideEffectCategory{SideEffectProcess},
	}
}

func (testTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	return map[string]any{"sum": a + b}, nil
}

func TestRegistryAndSafeInvoke(t *testing.T) {
	// register
	if err := RegisterTool(testTool{}); err != nil && err.Error() != "tool \"sum\" already registered" {
		t.Fatal(err)
	}
	tl, ok := ResolveTool("sum")
	if !ok || tl == nil {
		t.Fatal("tool not resolved")
	}
	// missing permission
	_, err := SafeInvoke(context.Background(), tl, map[string]any{"a": 1.0, "b": 2.0}, map[string]bool{}, JSONSchemaValidator)
	if err == nil {
		t.Fatal("expected permission error")
	}
	// ok
	out, err := SafeInvoke(context.Background(), tl, map[string]any{"a": 1.0, "b": 2.0}, map[string]bool{"cpu": true}, JSONSchemaValidator)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["sum"]; !ok {
		t.Fatalf("missing sum in %v", out)
	}
	// bad input
	if _, err := SafeInvoke(context.Background(), tl, map[string]any{"a": "x", "b": 2.0}, map[string]bool{"cpu": true}, JSONSchemaValidator); err == nil {
		t.Fatal("expected validation error")
	}
}
