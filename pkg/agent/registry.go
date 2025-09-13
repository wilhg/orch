package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/wilhg/orch/pkg/errmodel"
)

// toolRegistry keeps tools by name.
var (
	toolsMu sync.RWMutex
	tools   = map[string]Tool{}
)

// RegisterTool registers a Tool by its descriptor name.
func RegisterTool(t Tool) error {
	if t == nil {
		return fmt.Errorf("tool is nil")
	}
	d := t.Describe()
	if d.Name == "" {
		return fmt.Errorf("tool name is empty")
	}
	// Validate schemas compile and side-effect categories are recognized.
	if err := validateSchemaOnly(d.InputSchema); err != nil {
		return fmt.Errorf("invalid input schema for %s: %w", d.Name, err)
	}
	if err := validateSchemaOnly(d.OutputSchema); err != nil {
		return fmt.Errorf("invalid output schema for %s: %w", d.Name, err)
	}
	for _, c := range d.SideEffects {
		switch c {
		case SideEffectNetwork, SideEffectFilesystem, SideEffectProcess, SideEffectModel, SideEffectDatabase, SideEffectCache, SideEffectSecret:
			// ok
		default:
			return fmt.Errorf("unknown side effect category %q for tool %s", c, d.Name)
		}
	}
	toolsMu.Lock()
	defer toolsMu.Unlock()
	if _, exists := tools[d.Name]; exists {
		return fmt.Errorf("tool %q already registered", d.Name)
	}
	tools[d.Name] = t
	return nil
}

// ResolveTool returns a Tool by name.
func ResolveTool(name string) (Tool, bool) {
	toolsMu.RLock()
	defer toolsMu.RUnlock()
	t, ok := tools[name]
	return t, ok
}

// SafeInvoke validates input against the tool's schema, invokes it, and validates output.
// Permission checks are passed in by the caller via allowed set; missing permissions cause a policy error.
func SafeInvoke(ctx context.Context, t Tool, args map[string]any, allowed map[string]bool, validate ValidateFunc) (map[string]any, error) {
	if t == nil {
		return nil, errmodel.Validation("bad_tool", "tool is nil", nil)
	}
	d := t.Describe()
	// permissions
	for _, p := range d.Permissions {
		if !allowed[p.Name] {
			return nil, errmodel.Policy("forbidden", "permission denied for tool", map[string]any{"permission": p.Name, "tool": d.Name})
		}
	}
	if err := validate(d.InputSchema, args); err != nil {
		return nil, errmodel.Validation("invalid_input", "tool input validation failed", map[string]any{"tool": d.Name, "error": err.Error()})
	}
	out, err := t.Invoke(ctx, args)
	if err != nil {
		return nil, err
	}
	if err := validate(d.OutputSchema, out); err != nil {
		return nil, errmodel.Validation("invalid_output", "tool output validation failed", map[string]any{"tool": d.Name, "error": err.Error()})
	}
	return out, nil
}

// RangeTools iterates over registered tools.
func RangeTools(fn func(name string, t Tool)) {
	toolsMu.RLock()
	defer toolsMu.RUnlock()
	for n, t := range tools {
		fn(n, t)
	}
}

// validateSchemaOnly compiles a JSON schema to ensure it is well-formed.
func validateSchemaOnly(schema []byte) error {
	if len(schema) == 0 {
		return nil
	}
	return CompileJSONSchema(schema)
}
