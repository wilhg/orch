package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/wilhg/orch/pkg/adapters/embedding"
	genai "google.golang.org/genai"
)

const defaultEmbeddingModel = "gemini-embedding-001"

type embedClient struct {
	client *genai.Client
	model  string
}

func (e *embedClient) Name() string { return "gemini" }

func (e *embedClient) Embed(ctx context.Context, inputs []string, opts map[string]any) ([]embedding.Vector, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	model := e.model
	if v, ok := opts["model"].(string); ok && v != "" {
		model = v
	}
	// Build genai content list for embedding
	contents := make([]*genai.Content, 0, len(inputs))
	for _, s := range inputs {
		contents = append(contents, &genai.Content{Parts: []*genai.Part{{Text: s}}})
	}
	res, err := e.client.Models.EmbedContent(ctx, model, contents, nil)
	if err != nil {
		return nil, err
	}
	out := make([]embedding.Vector, 0, len(res.Embeddings))
	for _, emb := range res.Embeddings {
		vec := make([]float32, len(emb.Values))
		for i := range emb.Values {
			vec[i] = float32(emb.Values[i])
		}
		out = append(out, embedding.Vector(vec))
	}
	return out, nil
}

// Factory creates a Gemini embedder using GOOGLE_API_KEY by default.
func Factory(ctx context.Context, cfg map[string]any) (embedding.Embedder, error) { // nolint: revive
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if v, ok := cfg["api_key"].(string); ok && v != "" {
		apiKey = v
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: missing API key; set GOOGLE_API_KEY or cfg.api_key")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, err
	}
	model := defaultEmbeddingModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}
	return &embedClient{client: client, model: model}, nil
}

func init() {
	_ = embedding.Register("gemini", Factory)
}
