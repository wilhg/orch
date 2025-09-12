package tools

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/wilhg/orch/pkg/agent"
)

type HTTPGetTool struct{}

func (HTTPGetTool) Describe() agent.ToolDescriptor {
	in := []byte(`{"type":"object","properties":{"url":{"type":"string","format":"uri"},"timeout_ms":{"type":"integer","minimum":1,"maximum":60000}},"required":["url"],"additionalProperties":false}`)
	out := []byte(`{"type":"object","properties":{"status":{"type":"integer"},"body":{"type":"string"}},"required":["status","body"],"additionalProperties":false}`)
	return agent.ToolDescriptor{
		Name:         "http.get",
		Description:  "Performs an HTTP GET request",
		InputSchema:  in,
		OutputSchema: out,
		Permissions:  []agent.ToolPermission{{Name: "network:outbound"}},
	}
}

func (HTTPGetTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	url, _ := args["url"].(string)
	to := 10000
	if v, ok := args["timeout_ms"].(int); ok && v > 0 {
		to = v
	} else if v, ok := args["timeout_ms"].(float64); ok && v > 0 {
		to = int(v)
	}
	client := &http.Client{Timeout: time.Duration(to) * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return map[string]any{"status": res.StatusCode, "body": string(b)}, nil
}
