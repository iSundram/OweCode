package kimi

import (
	"context"

	"github.com/iSundram/OweCode/internal/ai"
	openaiProvider "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "https://api.moonshot.cn/v1"

// Client implements ai.Provider using Kimi/Moonshot's OpenAI-compatible API.
type Client struct {
	inner *openaiProvider.Client
}

func New(cfg ai.ProviderConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "kimi-k2.5"
	}
	return &Client{inner: openaiProvider.New(cfg)}
}

func (c *Client) Name() string      { return "kimi" }
func (c *Client) ContextLimit() int { return 256000 }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "kimi-k2.5", Name: "Kimi K2.5", ContextLimit: 256000},
		{ID: "kimi-k2-0905-preview", Name: "Kimi K2 Preview", ContextLimit: 256000},
		{ID: "kimi-k2-turbo-preview", Name: "Kimi K2 Turbo Preview", ContextLimit: 256000},
		{ID: "kimi-k2-thinking", Name: "Kimi K2 Thinking", ContextLimit: 256000},
		{ID: "kimi-k2-thinking-turbo", Name: "Kimi K2 Thinking Turbo", ContextLimit: 256000},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return c.inner.TokenCount(messages)
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return c.inner.Complete(ctx, req)
}
