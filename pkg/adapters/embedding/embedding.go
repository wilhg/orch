package embedding

import (
	"context"
	"fmt"
	"sync"
)

// Vector represents a single embedding vector.
type Vector []float32

// Embedder produces embedding vectors from text inputs.
//
// Implementations should be deterministic for the same input unless options specify
// non-deterministic behavior. All network or I/O operations must honor ctx.
type Embedder interface {
	// Name returns a short provider name (e.g., "openai", "voyage").
	Name() string
	// Embed returns one vector per input string, in order.
	Embed(ctx context.Context, inputs []string, opts map[string]any) ([]Vector, error)
}

// Factory constructs an Embedder from a provider-specific configuration map.
type Factory func(ctx context.Context, cfg map[string]any) (Embedder, error)

var (
	regMu     sync.RWMutex
	factories = map[string]Factory{}
)

// Register registers an Embedder factory under a provider name.
func Register(name string, f Factory) error {
	if name == "" {
		return fmt.Errorf("embedding: empty provider name")
	}
	if f == nil {
		return fmt.Errorf("embedding: nil factory for %q", name)
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := factories[name]; exists {
		return fmt.Errorf("embedding: provider %q already registered", name)
	}
	factories[name] = f
	return nil
}

// Resolve retrieves a registered factory by name.
func Resolve(name string) (Factory, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// Range calls fn for each registered provider name and factory.
func Range(fn func(name string, f Factory)) {
	regMu.RLock()
	defer regMu.RUnlock()
	for n, f := range factories {
		fn(n, f)
	}
}
