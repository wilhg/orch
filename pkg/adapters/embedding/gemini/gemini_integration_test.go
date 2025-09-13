//go:build integration

package gemini

import (
	"context"
	"os"
	"testing"
)

func TestGeminiEmbeddings(t *testing.T) {
	if os.Getenv("GOOGLE_API_KEY") == "" {
		t.Skip("GOOGLE_API_KEY not set")
	}
	ctx := context.Background()
	e, err := Factory(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	vecs, err := e.Embed(ctx, []string{"hello world", "gemini rocks"}, nil)
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("got %d vectors, want 2", len(vecs))
	}
	if len(vecs[0]) == 0 {
		t.Fatalf("empty vector")
	}
}
