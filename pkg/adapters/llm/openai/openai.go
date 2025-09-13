package openai

import (
	"context"
	"fmt"
	"os"

	oa "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
	"github.com/wilhg/orch/pkg/adapters/llm"
)

const (
	defaultModel = "gpt-5-nano"
)

type clientWrapper struct {
	client oa.Client
	model  string
}

func (c *clientWrapper) Name() string { return "openai" }

func (c *clientWrapper) Generate(ctx context.Context, messages []llm.Message, opts map[string]any) (llm.GenerateResult, error) {
	model := c.model
	if v, ok := opts["model"].(string); ok && v != "" {
		model = v
	}

	// Map our messages to SDK union type
	mm := make([]oa.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "user":
			mm = append(mm, oa.UserMessage(m.Content))
		case "system":
			mm = append(mm, oa.SystemMessage(m.Content))
		case "assistant":
			mm = append(mm, oa.AssistantMessage(m.Content))
		default:
			mm = append(mm, oa.UserMessage(m.Content))
		}
	}

	resp, err := c.client.Chat.Completions.New(ctx, oa.ChatCompletionNewParams{
		Model:    shared.ChatModel(model),
		Messages: mm,
	})
	if err != nil {
		return llm.GenerateResult{}, err
	}
	var out string
	if len(resp.Choices) > 0 {
		out = resp.Choices[0].Message.Content
	}
	usage := resp.Usage
	return llm.GenerateResult{
		Text:         out,
		PromptTokens: int(usage.PromptTokens),
		OutputTokens: int(usage.CompletionTokens),
		TotalTokens:  int(usage.TotalTokens),
		Model:        model,
	}, nil
}

// Factory registers the OpenAI LLM provider: cfg keys: api_key, model
func Factory(ctx context.Context, cfg map[string]any) (llm.LLM, error) { // nolint: revive
	_ = ctx
	apiKey := os.Getenv("OPENAI_API_KEY")
	if v, ok := cfg["api_key"].(string); ok && v != "" {
		apiKey = v
	}
	if apiKey == "" {
		return nil, fmt.Errorf("openai: missing API key; set OPENAI_API_KEY or cfg.api_key")
	}
	model := defaultModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}

	c := oa.NewClient(option.WithAPIKey(apiKey))
	return &clientWrapper{client: c, model: model}, nil
}

func init() {
	_ = llm.Register("openai", Factory)
}
