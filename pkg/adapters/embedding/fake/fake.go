package fake

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"sort"

	"github.com/wilhg/orch/pkg/adapters/embedding"
)

// Embedder is a deterministic hash-based embedder suitable for unit tests.
// It produces fixed-size vectors with values derived from SHA-256 of the input string.
type Embedder struct {
	dim  int
	name string
}

// New returns a new fake embedder with the given dimension (>= 4).
func New(dim int) *Embedder {
	if dim < 4 {
		dim = 4
	}
	return &Embedder{dim: dim, name: "fake"}
}

func (e *Embedder) Name() string { return e.name }

func (e *Embedder) Embed(ctx context.Context, inputs []string, opts map[string]any) ([]embedding.Vector, error) {
	// Keep output stable regardless of map iteration order in opts
	// by folding opts keys into an extra seed, sorted by key.
	var optSeed uint64
	if len(opts) > 0 {
		keys := make([]string, 0, len(opts))
		for k := range opts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h := sha256.Sum256([]byte(k))
			optSeed ^= binary.LittleEndian.Uint64(h[:8])
		}
	}

	out := make([]embedding.Vector, len(inputs))
	for i, s := range inputs {
		vec := make(embedding.Vector, e.dim)
		h := sha256.Sum256([]byte(s))
		// Fill in chunks of 4 bytes into float32s; simple deterministic pattern.
		for j := 0; j < e.dim; j++ {
			off := (j * 4) % len(h)
			u := binary.LittleEndian.Uint32(h[off : off+4])
			u ^= uint32(optSeed)
			// Scale to [0,1) then shift to [-0.5, 0.5)
			vec[j] = (float32(u&0x7FFFFFFF) / float32(1<<31)) - 0.5
		}
		out[i] = vec
	}
	return out, nil
}
