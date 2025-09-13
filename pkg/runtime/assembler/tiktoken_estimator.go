package assembler

import (
	tiktoken "github.com/pkoukk/tiktoken-go"
)

// NewTikTokenEstimator returns a TokenEstimator backed by tiktoken-go for the given model.
// Common models: "gpt-4", "gpt-3.5-turbo", "gpt-4o". See tiktoken-go docs for support.
// If the model is unknown, EncodingForModel returns an error.
func NewTikTokenEstimator(model string) (TokenEstimator, error) {
	enc, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return nil, err
	}
	return func(text string) int {
		// Encode returns token ids; we count them.
		return len(enc.Encode(text, nil, nil))
	}, nil
}
