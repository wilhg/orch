//go:build integration

package chromadb

import (
	"context"
	"fmt"
	"testing"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	vstore "github.com/wilhg/orch/pkg/adapters/vectorstore"
)

func TestChromaDBUpsertAndQuery(t *testing.T) {
	ctx := context.Background()

	req := tc.ContainerRequest{
		Image:        "ghcr.io/chroma-core/chroma:latest",
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor:   wait.ForHTTP("/api/v1/heartbeat").WithPort("8000/tcp"),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Skipf("skip: cannot start chromadb: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "8000/tcp")
	if err != nil {
		t.Fatal(err)
	}
	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Create store
	vs, err := Factory(ctx, map[string]any{
		"base_url":          baseURL,
		"collection":        "itest",
		"create_if_missing": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Upsert items
	items := []vstore.Item{
		{ID: "a1", Namespace: "ns1", Vector: vstore.Vector{1, 0}, Metadata: map[string]any{"doc": "1", "tag": "x"}},
		{ID: "a2", Namespace: "ns1", Vector: vstore.Vector{0.8, 0.2}, Metadata: map[string]any{"doc": "2", "tag": "y"}},
		{ID: "b1", Namespace: "ns2", Vector: vstore.Vector{0, 1}, Metadata: map[string]any{"doc": "3", "tag": "x"}},
	}
	if err := vs.Upsert(ctx, items); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Query in ns1 near [1,0]
	matches, err := vs.Query(ctx, vstore.Vector{1, 0}, 2, vstore.Filter{Namespace: "ns1"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no matches returned")
	}
	if matches[0].Item.ID != "a1" {
		t.Fatalf("top match=%s want a1", matches[0].Item.ID)
	}
}
