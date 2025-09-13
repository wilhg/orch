//go:build integration

package gemini

import (
	"context"
	"os"
	"testing"

	"github.com/wilhg/orch/pkg/adapters/llm"
)

func TestGeminiChatGenerate(t *testing.T) {
	if os.Getenv("GOOGLE_API_KEY") == "" {
		t.Skip("GOOGLE_API_KEY not set")
	}
	ctx := context.Background()
	m, err := Factory(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	msgs := []llm.Message{{Role: "user", Content: "Say 'hello from gemini'"}}
	res, err := m.Generate(ctx, msgs, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if res.Text == "" {
		t.Fatalf("empty response text")
	}
}
