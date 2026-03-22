package xai

import (
	"context"

	"github.com/iSundram/OweCode/internal/ai"
	openaiProvider "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "https://api.x.ai/v1"

// Client implements ai.Provider using xAI's OpenAI-compatible API.
type Client struct {
	inner *openaiProvider.Client
	model string
}

func New(cfg ai.ProviderConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "grok-4.20-reasoning"
	}
	return &Client{
		inner: openaiProvider.New(cfg),
		model: cfg.DefaultModel,
	}
}

func (c *Client) Name() string      { return "xai" }
func (c *Client) ContextLimit() int { return modelContextLimit(c.model) }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "grok-4.20", Name: "Grok 4.20", ContextLimit: 2000000},
		{ID: "grok-4.20-reasoning", Name: "Grok 4.20 Reasoning", ContextLimit: 2000000},
		{ID: "grok-4.20-multi-agent", Name: "Grok 4.20 Multi-Agent (Beta)", ContextLimit: 2000000},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return c.inner.TokenCount(messages)
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return c.inner.Complete(ctx, req)
}

func modelContextLimit(model string) int {
	switch model {
	case "grok-4.20", "grok-4.20-reasoning", "grok-4.20-multi-agent":
		return 2000000
	default:
		return 128000
	}
}
