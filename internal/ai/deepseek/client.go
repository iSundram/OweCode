package deepseek

import (
	"context"

	"github.com/iSundram/OweCode/internal/ai"
	openaiProvider "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "https://api.deepseek.com/v1"

// Client implements ai.Provider using DeepSeek's OpenAI-compatible API.
type Client struct {
	inner *openaiProvider.Client
}

func New(cfg ai.ProviderConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "deepseek-chat"
	}
	return &Client{inner: openaiProvider.New(cfg)}
}

func (c *Client) Name() string      { return "deepseek" }
func (c *Client) ContextLimit() int { return 128000 }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "deepseek-chat", Name: "DeepSeek Chat (V3.2)", ContextLimit: 128000},
		{ID: "deepseek-reasoner", Name: "DeepSeek Reasoner (V3.2)", ContextLimit: 128000},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return c.inner.TokenCount(messages)
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return c.inner.Complete(ctx, req)
}
