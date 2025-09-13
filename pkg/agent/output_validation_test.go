package agent

import (
	"context"
	"testing"

	"github.com/wilhg/orch/pkg/errmodel"
)

// badOutputTool declares an output schema that the Invoke implementation violates.
type badOutputTool struct{}

func (badOutputTool) Describe() ToolDescriptor {
	// Output requires {"ok": boolean}
	return ToolDescriptor{
		Name:         "bad_output_tool",
		Description:  "returns output that violates its OutputSchema",
		InputSchema:  []byte(`{"type":"object","properties":{},"additionalProperties":false}`),
		OutputSchema: []byte(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`),
		Permissions:  []ToolPermission{{Name: "cpu"}},
		SideEffects:  []SideEffectCategory{SideEffectProcess},
	}
}

func (badOutputTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Violate schema by returning string for ok and missing required boolean type
	return map[string]any{"ok": "yes"}, nil
}

func TestSafeInvoke_InvalidOutput_YieldsParseError(t *testing.T) {
	t.Helper()

	tool := badOutputTool{}
	// Allow required permission so we reach output validation stage.
	allowed := map[string]bool{"cpu": true}

	out, err := SafeInvoke(context.Background(), tool, map[string]any{}, allowed, JSONSchemaValidator)
	if err == nil {
		t.Fatalf("expected error, got output=%v", out)
	}
	ce := errmodel.From(err)
	if ce == nil {
		t.Fatalf("expected compact error, got %T: %v", err, err)
	}
	if ce.Category != errmodel.CategoryValidation || ce.Code != "invalid_output" {
		t.Fatalf("unexpected error category/code: %+v", ce)
	}
	if ce.Context == nil || ce.Context["tool"] != "bad_output_tool" {
		t.Fatalf("expected context to include tool name, got %+v", ce.Context)
	}
}
