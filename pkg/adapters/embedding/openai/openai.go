package openai

import (
	"context"
	"fmt"
	"os"

	oa "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/wilhg/orch/pkg/adapters/embedding"
)

const defaultEmbeddingModel = "text-embedding-3-small"

type embedClient struct {
	client oa.Client
	model  string
}

func (e *embedClient) Name() string { return "openai" }

func (e *embedClient) Embed(ctx context.Context, inputs []string, opts map[string]any) ([]embedding.Vector, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	// The SDK accepts a slice of strings via the Input union
	resp, err := e.client.Embeddings.New(ctx, oa.EmbeddingNewParams{
		Model: oa.EmbeddingModelTextEmbedding3Small,
		Input: oa.EmbeddingNewParamsInputUnion{OfArrayOfStrings: inputs},
	})
	if err != nil {
		return nil, err
	}
	out := make([]embedding.Vector, 0, len(resp.Data))
	for _, d := range resp.Data {
		// d.Embedding is []float32 or []float64 depending on SDK
		// openai-go v2 returns []float32-compatible via type alias; convert defensively
		vec := make([]float32, len(d.Embedding))
		for i := range d.Embedding {
			vec[i] = float32(d.Embedding[i])
		}
		out = append(out, embedding.Vector(vec))
	}
	return out, nil
}

// Factory registers the OpenAI embedder: cfg keys: api_key, model
func Factory(ctx context.Context, cfg map[string]any) (embedding.Embedder, error) { // nolint: revive
	_ = ctx
	apiKey := os.Getenv("OPENAI_API_KEY")
	if v, ok := cfg["api_key"].(string); ok && v != "" {
		apiKey = v
	}
	if apiKey == "" {
		return nil, fmt.Errorf("openai: missing API key; set OPENAI_API_KEY or cfg.api_key")
	}
	model := defaultEmbeddingModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}
	c := oa.NewClient(option.WithAPIKey(apiKey))
	return &embedClient{client: c, model: model}, nil
}

func init() {
	_ = embedding.Register("openai", Factory)
}
