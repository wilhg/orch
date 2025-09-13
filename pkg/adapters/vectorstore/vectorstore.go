package vectorstore

import (
	"context"
	"fmt"
	"sync"
)

// Vector is a single dense embedding vector.
type Vector []float32

// Item represents a vectorized document chunk with metadata for filtering and citation.
type Item struct {
	// ID is provider-assigned or caller-provided unique identifier for the item.
	ID string
	// Namespace groups items logically (e.g., by dataset, tenant, or collection).
	Namespace string
	// Vector is the dense embedding.
	Vector Vector
	// Metadata carries arbitrary attributes for filtering (e.g., source, doc_id, tags).
	Metadata map[string]any
}

// Match is a search result with similarity score and original item.
type Match struct {
	Item  Item
	Score float32 // higher is more similar
}

// VectorStore defines upsert and similarity query operations.
type VectorStore interface {
	// Upsert inserts or replaces items by ID within a namespace.
	Upsert(ctx context.Context, items []Item) error
	// Query returns top-k most similar items to the query vector, optionally filtered by namespace and metadata.
	Query(ctx context.Context, query Vector, k int, filter Filter) ([]Match, error)
}

// Filter constrains query results.
type Filter struct {
	Namespace string
	// Equals matches exact key/value pairs in metadata (AND semantics across keys).
	Equals map[string]any
}

// Factory constructs a VectorStore instance from a provider-specific configuration.
type Factory func(ctx context.Context, cfg map[string]any) (VectorStore, error)

var (
	regMu     sync.RWMutex
	factories = map[string]Factory{}
)

// Register registers a VectorStore factory.
func Register(name string, f Factory) error {
	if name == "" {
		return fmt.Errorf("vectorstore: empty provider name")
	}
	if f == nil {
		return fmt.Errorf("vectorstore: nil factory for %q", name)
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := factories[name]; exists {
		return fmt.Errorf("vectorstore: provider %q already registered", name)
	}
	factories[name] = f
	return nil
}

// Resolve gets a registered factory by name.
func Resolve(name string) (Factory, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// Range iterates all registered factories.
func Range(fn func(name string, f Factory)) {
	regMu.RLock()
	defer regMu.RUnlock()
	for n, f := range factories {
		fn(n, f)
	}
}
