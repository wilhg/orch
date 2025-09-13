package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/wilhg/orch/pkg/adapters/llm"
	genai "google.golang.org/genai"
)

const defaultModel = "gemini-2.5-flash-lite"

type clientWrapper struct {
	client *genai.Client
	model  string
}

func (c *clientWrapper) Name() string { return "gemini" }

func (c *clientWrapper) Generate(ctx context.Context, messages []llm.Message, opts map[string]any) (llm.GenerateResult, error) {
	model := c.model
	if v, ok := opts["model"].(string); ok && v != "" {
		model = v
	}
	// Build a single turn from concatenated text for simplicity
	var text string
	for _, m := range messages {
		if m.Content != "" {
			text += m.Content + "\n"
		}
	}
	parts := []*genai.Part{{Text: text}}
	res, err := c.client.Models.GenerateContent(ctx, model, []*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return llm.GenerateResult{}, err
	}
	out := res.Text()
	return llm.GenerateResult{Text: out, Model: model}, nil
}

// Factory creates a Gemini LLM client using GOOGLE_API_KEY by default.
func Factory(ctx context.Context, cfg map[string]any) (llm.LLM, error) { // nolint: revive
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if v, ok := cfg["api_key"].(string); ok && v != "" {
		apiKey = v
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: missing API key; set GOOGLE_API_KEY or cfg.api_key")
	}
	// Prefer Gemini API backend
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, err
	}
	model := defaultModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}
	return &clientWrapper{client: client, model: model}, nil
}

func init() {
	_ = llm.Register("gemini", Factory)
}
