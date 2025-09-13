package memory

import (
	"context"
	"testing"

	"github.com/wilhg/orch/pkg/adapters/vectorstore"
)

func TestUpsertAndQuery(t *testing.T) {
	ctx := context.Background()
	s := New()

	// Upsert a few items across namespaces.
	items := []vectorstore.Item{
		{ID: "a1", Namespace: "ns1", Vector: vectorstore.Vector{1, 0}, Metadata: map[string]any{"doc": "1", "tag": "x"}},
		{ID: "a2", Namespace: "ns1", Vector: vectorstore.Vector{0.8, 0.2}, Metadata: map[string]any{"doc": "2", "tag": "y"}},
		{ID: "b1", Namespace: "ns2", Vector: vectorstore.Vector{0, 1}, Metadata: map[string]any{"doc": "3", "tag": "x"}},
	}
	if err := s.Upsert(ctx, items); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Query in ns1 near [1,0]
	matches, err := s.Query(ctx, vectorstore.Vector{1, 0}, 2, vectorstore.Filter{Namespace: "ns1"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("len=%d want 2", len(matches))
	}
	if matches[0].Item.ID != "a1" {
		t.Fatalf("top match=%s want a1", matches[0].Item.ID)
	}

	// Filter by metadata
	matches, err = s.Query(ctx, vectorstore.Vector{1, 0}, 2, vectorstore.Filter{Namespace: "ns1", Equals: map[string]any{"tag": "y"}})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(matches) != 1 || matches[0].Item.ID != "a2" {
		t.Fatalf("filtered result unexpected: %+v", matches)
	}

	// Ensure namespace isolation
	matches, err = s.Query(ctx, vectorstore.Vector{0, 1}, 10, vectorstore.Filter{Namespace: "ns2"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(matches) != 1 || matches[0].Item.ID != "b1" {
		t.Fatalf("ns2 query unexpected: %+v", matches)
	}
}
