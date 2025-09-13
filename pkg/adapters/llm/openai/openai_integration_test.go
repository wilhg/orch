//go:build integration

package openai

import (
	"context"
	"os"
	"testing"

	"github.com/wilhg/orch/pkg/adapters/llm"
)

func TestOpenAIChatGenerate(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	ctx := context.Background()
	// default model gpt-5-nano per repo rule; allow override via env
	m, err := Factory(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	msgs := []llm.Message{{Role: "user", Content: "Say 'pong'"}}
	res, err := m.Generate(ctx, msgs, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if res.Text == "" {
		t.Fatalf("empty response text")
	}
}
