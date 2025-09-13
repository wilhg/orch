package llm

import (
	"context"
	"fmt"
	"sync"
)

// Message represents a chat message with a role and content.
type Message struct {
	Role    string
	Content string
}

// GenerateResult contains the model's text output and token usage if available.
type GenerateResult struct {
	Text         string
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	Model        string
}

// LLM defines a minimal chat/text generation interface.
type LLM interface {
	// Name returns provider name (e.g., "openai").
	Name() string
	// Generate creates a completion from a list of messages. Implementations may ignore messages except the latest user if they are pure-completion models.
	Generate(ctx context.Context, messages []Message, opts map[string]any) (GenerateResult, error)
}

// Factory constructs an LLM from provider-specific config.
type Factory func(ctx context.Context, cfg map[string]any) (LLM, error)

var (
	regMu     sync.RWMutex
	factories = map[string]Factory{}
)

// Register registers an LLM factory under a provider name.
func Register(name string, f Factory) error {
	if name == "" {
		return fmt.Errorf("llm: empty provider name")
	}
	if f == nil {
		return fmt.Errorf("llm: nil factory for %q", name)
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := factories[name]; exists {
		return fmt.Errorf("llm: provider %q already registered", name)
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
