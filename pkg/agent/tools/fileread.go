package tools

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/wilhg/orch/pkg/agent"
)

// FileReadTool reads a file from a provided fs.FS sandbox.
type FileReadTool struct{ FS fs.FS }

func (t FileReadTool) Describe() agent.ToolDescriptor {
	in := []byte(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}`)
	out := []byte(`{"type":"object","properties":{"content":{"type":"string"}},"required":["content"],"additionalProperties":false}`)
	return agent.ToolDescriptor{
		Name:         "fs.read",
		Description:  "Reads a text file from sandboxed fs",
		InputSchema:  in,
		OutputSchema: out,
		Permissions:  []agent.ToolPermission{{Name: "fs:read"}},
	}
}

func (t FileReadTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	if t.FS == nil {
		return nil, errors.New("no fs configured")
	}
	p, _ := args["path"].(string)
	if p == "" {
		return nil, errors.New("path required")
	}
	if filepath.IsAbs(p) || filepath.Clean(p) != p || strings.Contains(p, "..") {
		return nil, errors.New("invalid path")
	}
	b, err := fs.ReadFile(t.FS, p)
	if err != nil {
		return nil, err
	}
	return map[string]any{"content": string(b)}, nil
}
