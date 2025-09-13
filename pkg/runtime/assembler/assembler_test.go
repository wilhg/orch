package assembler

import (
	"testing"
)

// Test cases cover:
// - Pinning: pinned items always included first until budget exhausted
// - Deduplication: duplicate chunks by ID are included once
// - Token budgeting: respects max tokens using a simple token estimator
// - Deterministic ordering: stable tie-breaker by source then chunk ID

func TestAssemble_Pinning_Dedup_Budget(t *testing.T) {
	est := func(text string) int { return len([]rune(text)) }
	asm := New(WithTokenEstimator(est), WithMaxTokens(10))

	// Inputs with duplicates and pins
	items := []Item{
		{Source: "docA", ChunkID: "1", Text: "abcd"},  // 4 tokens
		{Source: "docA", ChunkID: "1", Text: "abcd"},  // duplicate
		{Source: "docB", ChunkID: "2", Text: "ef"},    // 2 tokens
		{Source: "docC", ChunkID: "3", Text: "ghijk"}, // 5 tokens
	}
	pins := []Pinned{{Source: "docC", ChunkID: "3"}} // must include first

	out, log := asm.Assemble(items, pins)

	// Expect pinned docC:3 first (5), then docA:1 (4) fits, docB:2 (2) would exceed (5+4+2=11) so excluded.
	if len(out) != 2 {
		t.Fatalf("got %d items, want 2", len(out))
	}
	if out[0].Source != "docC" || out[0].ChunkID != "3" {
		t.Fatalf("first not pinned: %+v", out[0])
	}
	if out[1].Source != "docA" || out[1].ChunkID != "1" {
		t.Fatalf("second unexpected: %+v", out[1])
	}
	// Ensure dedup removed the duplicate docA:1
	seen := map[string]bool{}
	for _, it := range out {
		key := it.Source + ":" + it.ChunkID
		if seen[key] {
			t.Fatalf("duplicate present: %s", key)
		}
		seen[key] = true
	}
	if log.TotalTokens != 9 || log.IncludedTokens != 9 || log.DroppedCount != 1 {
		t.Fatalf("log mismatch: %+v", log)
	}
}

func TestAssemble_DeterministicOrder(t *testing.T) {
	est := func(text string) int { return len(text) }
	asm := New(WithTokenEstimator(est), WithMaxTokens(100))
	items := []Item{
		{Source: "b", ChunkID: "2", Text: "x"},
		{Source: "a", ChunkID: "2", Text: "x"},
		{Source: "a", ChunkID: "1", Text: "x"},
	}
	out, _ := asm.Assemble(items, nil)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	// Expect order: a:1, a:2, b:2
	want := []struct{ s, c string }{{"a", "1"}, {"a", "2"}, {"b", "2"}}
	for i, w := range want {
		if out[i].Source != w.s || out[i].ChunkID != w.c {
			t.Fatalf("order[%d]=%s:%s want %s:%s", i, out[i].Source, out[i].ChunkID, w.s, w.c)
		}
	}
}
