package openrouter

import (
	"context"

	"github.com/iSundram/OweCode/internal/ai"
	openaiProvider "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

// Client implements ai.Provider using the OpenRouter API (OpenAI-compatible).
type Client struct {
	inner *openaiProvider.Client
}

func New(cfg ai.ProviderConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "openai/gpt-4o"
	}
	return &Client{inner: openaiProvider.New(cfg)}
}

func (c *Client) Name() string      { return "openrouter" }
func (c *Client) ContextLimit() int { return c.inner.ContextLimit() }

func (c *Client) Models(ctx context.Context) ([]ai.Model, error) {
	return c.inner.Models(ctx)
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return c.inner.TokenCount(messages)
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return c.inner.Complete(ctx, req)
}
