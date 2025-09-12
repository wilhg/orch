package agent

import (
	"context"

	"github.com/wilhg/orch/pkg/errmodel"
)

// ToolEffectHandler routes intents with Name "tool" to registered tools.
// Args must contain {"name": string, "args": map[string]any}.
type ToolEffectHandler struct {
	AllowedPermissions map[string]bool
	Validate           ValidateFunc
}

func (h ToolEffectHandler) CanHandle(intent Intent) bool { return intent.Name == "tool" }

func (h ToolEffectHandler) Handle(ctx context.Context, s State, intent Intent) ([]Event, error) {
	nameAny, ok := intent.Args["name"]
	if !ok {
		return nil, errMissing("name")
	}
	name, _ := nameAny.(string)
	tool, ok := ResolveTool(name)
	if !ok || tool == nil {
		return nil, errUnknownTool(name)
	}
	var targs map[string]any
	if v, ok := intent.Args["args"]; ok {
		if m, ok := v.(map[string]any); ok {
			targs = m
		}
	}
	out, err := SafeInvoke(ctx, tool, targs, h.AllowedPermissions, h.Validate)
	if err != nil {
		return nil, err
	}
	ev := Event{Type: "tool_result", Payload: map[string]any{"tool": name, "output": out}}
	return []Event{ev}, nil
}

func errMissing(k string) error {
	return errmodel.Validation("missing_fields", k+" required", map[string]any{"fields": []string{k}})
}
func errUnknownTool(n string) error {
	return errmodel.Validation("not_found", "tool not found", map[string]any{"tool": n})
}
