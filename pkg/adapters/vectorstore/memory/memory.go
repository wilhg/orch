package memory

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"

	"github.com/wilhg/orch/pkg/adapters/vectorstore"
)

// Store is an in-memory VectorStore implementation intended for tests and examples.
type Store struct {
	mu     sync.RWMutex
	byNSID map[string]map[string]vectorstore.Item // namespace -> id -> item
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{byNSID: make(map[string]map[string]vectorstore.Item)}
}

// Upsert inserts or replaces items.
func (s *Store) Upsert(ctx context.Context, items []vectorstore.Item) error {
	if len(items) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range items {
		if it.ID == "" {
			return errors.New("memory vectorstore: empty id")
		}
		if len(it.Vector) == 0 {
			return errors.New("memory vectorstore: empty vector")
		}
		ns := it.Namespace
		if ns == "" {
			ns = "default"
		}
		bucket, ok := s.byNSID[ns]
		if !ok {
			bucket = make(map[string]vectorstore.Item)
			s.byNSID[ns] = bucket
		}
		bucket[it.ID] = it
	}
	return nil
}

// Query performs cosine similarity search with optional namespace and metadata equality filter.
func (s *Store) Query(ctx context.Context, query vectorstore.Vector, k int, filter vectorstore.Filter) ([]vectorstore.Match, error) {
	s.mu.RLock()
	snapshot := make(map[string]map[string]vectorstore.Item, len(s.byNSID))
	for ns, bucket := range s.byNSID {
		dup := make(map[string]vectorstore.Item, len(bucket))
		for id, it := range bucket {
			dup[id] = it
		}
		snapshot[ns] = dup
	}
	s.mu.RUnlock()

	ns := filter.Namespace
	if ns == "" {
		ns = "default"
	}
	bucket := snapshot[ns]
	if bucket == nil {
		return nil, nil
	}

	// Precompute norm of query
	qnorm := dot(query, query)
	if qnorm == 0 {
		return nil, errors.New("memory vectorstore: zero-norm query vector")
	}
	qnorm = math.Sqrt(qnorm)

	// Gather matches
	matches := make([]vectorstore.Match, 0, len(bucket))
	for _, it := range bucket {
		if !metaEquals(it.Metadata, filter.Equals) {
			continue
		}
		if len(it.Vector) != len(query) {
			// Skip dimension mismatch
			continue
		}
		sim := cosine(query, it.Vector, qnorm)
		matches = append(matches, vectorstore.Match{Item: it, Score: sim})
	}

	// Sort by score desc and truncate to k
	sort.Slice(matches, func(i, j int) bool { return matches[i].Score > matches[j].Score })
	if k > 0 && len(matches) > k {
		matches = matches[:k]
	}
	return matches, nil
}

func metaEquals(have map[string]any, want map[string]any) bool {
	if len(want) == 0 {
		return true
	}
	if have == nil {
		return false
	}
	for k, v := range want {
		if hv, ok := have[k]; !ok || hv != v {
			return false
		}
	}
	return true
}

func cosine(a, b vectorstore.Vector, qnorm float64) float32 {
	denom := qnorm * math.Sqrt(dot(b, b))
	if denom == 0 {
		return 0
	}
	return float32(dot(a, b) / denom)
}

func dot(a, b vectorstore.Vector) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var s float64
	for i := 0; i < n; i++ {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}
