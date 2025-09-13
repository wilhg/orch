package embedding_test

import (
	"context"
	"testing"

	"github.com/wilhg/orch/pkg/adapters/embedding"
	fakeembed "github.com/wilhg/orch/pkg/adapters/embedding/fake"
)

func TestRegistry(t *testing.T) {
	ctx := context.Background()
	// Register a temporary factory and ensure resolve works; isolate via name.
	name := "test-embedder"
	if _, ok := embedding.Resolve(name); ok {
		t.Fatalf("%s unexpectedly pre-registered", name)
	}
	if err := embedding.Register(name, func(ctx context.Context, cfg map[string]any) (embedding.Embedder, error) {
		return fakeembed.New(8), nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	f, ok := embedding.Resolve(name)
	if !ok {
		t.Fatalf("resolve failed for %s", name)
	}
	e, err := f(ctx, nil)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if e.Name() == "" {
		t.Fatalf("embedder missing name")
	}
	vecs, err := e.Embed(ctx, []string{"a", "b"}, nil)
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("got %d vectors, want 2", len(vecs))
	}
	if len(vecs[0]) != 8 || len(vecs[1]) != 8 {
		t.Fatalf("unexpected dimensions: %d %d", len(vecs[0]), len(vecs[1]))
	}
}
