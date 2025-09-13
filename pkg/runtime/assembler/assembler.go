package assembler

import (
	"sort"
)

// Item represents a retrievable chunk of context.
// Ordering and deduplication are based on (Source, ChunkID).
type Item struct {
	Source  string
	ChunkID string
	Text    string
}

// Pinned identifies an item that must be considered first.
type Pinned struct {
	Source  string
	ChunkID string
}

// AssemblyLog summarizes the assembly decision.
type AssemblyLog struct {
	TotalTokens    int // total tokens of included items
	IncludedTokens int // same as TotalTokens (kept for explicitness)
	DroppedCount   int // number of items excluded due to budget (duplicates are not counted)
}

// TokenEstimator estimates token usage of text content.
type TokenEstimator func(text string) int

// Assembler deterministically assembles context respecting pins, dedup, and token budget.
type Assembler struct {
	estimate  TokenEstimator
	maxTokens int
}

// Option configures the Assembler.
type Option func(*Assembler)

// WithTokenEstimator sets the token estimator. Defaults to rune length.
func WithTokenEstimator(est TokenEstimator) Option {
	return func(a *Assembler) {
		if est != nil {
			a.estimate = est
		}
	}
}

// WithMaxTokens sets the maximum token budget. Defaults to a large value (1e9).
func WithMaxTokens(n int) Option {
	return func(a *Assembler) {
		if n > 0 {
			a.maxTokens = n
		}
	}
}

// New creates a new Assembler.
func New(opts ...Option) *Assembler {
	a := &Assembler{
		estimate:  func(s string) int { return len([]rune(s)) },
		maxTokens: 1_000_000_000,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Assemble returns a deterministic selection of items given pins and token budget.
// Behavior:
// - Deduplicate by (source, chunkID).
// - Include pinned items first (sorted by source, chunkID), subject to budget.
// - Then include remaining items in a deterministic order: by source, then chunkID.
// - Never exceed the max token budget.
func (a *Assembler) Assemble(items []Item, pins []Pinned) ([]Item, AssemblyLog) {
	// Build maps for dedup and pin lookup
	type key struct{ s, c string }
	seen := make(map[key]Item)
	budgetDropped := 0
	for _, it := range items {
		k := key{it.Source, it.ChunkID}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = it
	}
	isPinned := make(map[key]bool)
	for _, p := range pins {
		isPinned[key{p.Source, p.ChunkID}] = true
	}

	// Split into pinned and others
	pinnedItems := make([]Item, 0, len(pins))
	otherItems := make([]Item, 0, len(seen))
	for k, it := range seen {
		if isPinned[k] {
			pinnedItems = append(pinnedItems, it)
		} else {
			otherItems = append(otherItems, it)
		}
	}

	// Deterministic sort: by Source, then ChunkID
	less := func(a, b Item) bool {
		if a.Source < b.Source {
			return true
		}
		if a.Source > b.Source {
			return false
		}
		return a.ChunkID < b.ChunkID
	}
	sort.Slice(pinnedItems, func(i, j int) bool { return less(pinnedItems[i], pinnedItems[j]) })
	sort.Slice(otherItems, func(i, j int) bool { return less(otherItems[i], otherItems[j]) })

	// Accumulate under budget
	budget := a.maxTokens
	result := make([]Item, 0, len(items))
	includedTokens := 0
	take := func(it Item) bool {
		cost := a.estimate(it.Text)
		if cost <= budget {
			budget -= cost
			includedTokens += cost
			result = append(result, it)
			return true
		}
		return false
	}
	for _, it := range pinnedItems {
		if !take(it) {
			budgetDropped++
		}
	}
	for _, it := range otherItems {
		if !take(it) {
			budgetDropped++
		}
	}

	log := AssemblyLog{TotalTokens: includedTokens, IncludedTokens: includedTokens, DroppedCount: budgetDropped}
	return result, log
}
