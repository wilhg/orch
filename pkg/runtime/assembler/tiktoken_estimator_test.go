package assembler

import "testing"

func TestNewTikTokenEstimator(t *testing.T) {
	est, err := NewTikTokenEstimator("gpt-4")
	if err != nil {
		t.Skipf("tiktoken not available for model: %v", err)
	}
	// Simple sanity: token count should be > 0 for a non-empty string
	if got := est("hello world"); got <= 0 {
		t.Fatalf("got %d tokens, want > 0", got)
	}
}
