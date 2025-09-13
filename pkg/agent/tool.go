package agent

import (
	"context"
)

// ToolPermission describes a capability a tool requires.
// Example: network:outbound, fs:read, model:generate, secret:aws
type ToolPermission struct {
	// Name is a stable, lower_snake identifier of the permission.
	Name string `json:"name"`
	// Description explains what the permission allows.
	Description string `json:"description,omitempty"`
}

// ToolDescriptor declares the static interface of a tool.
// InputSchema and OutputSchema are JSON Schemas (draft 2020-12) in UTF-8 bytes.
type ToolDescriptor struct {
	Name         string           `json:"name"`
	Description  string           `json:"description,omitempty"`
	InputSchema  []byte           `json:"input_schema"`
	OutputSchema []byte           `json:"output_schema"`
	Permissions  []ToolPermission `json:"permissions,omitempty"`
}

// Tool defines a callable unit with schema-validated inputs/outputs and a permission model.
// Implementations should be deterministic for identical inputs or declare side-effect semantics clearly.
type Tool interface {
	// Describe returns the public descriptor (schemas, permissions).
	Describe() ToolDescriptor
	// Invoke executes the tool with validated args. The args MUST conform to InputSchema.
	// The returned map MUST conform to OutputSchema.
	Invoke(ctx context.Context, args map[string]any) (map[string]any, error)
}

// DescribeTool is a helper to get a ToolDescriptor from a Tool (nil-safe).
func DescribeTool(t Tool) ToolDescriptor {
	if t == nil {
		return ToolDescriptor{}
	}
	return t.Describe()
}
