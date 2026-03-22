package minimax

import (
	"context"

	"github.com/iSundram/OweCode/internal/ai"
	openaiProvider "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "https://api.minimaxi.com/v1"

// Client implements ai.Provider using MiniMax's OpenAI-compatible API.
type Client struct {
	inner *openaiProvider.Client
}

func New(cfg ai.ProviderConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "MiniMax-M2.7"
	}
	return &Client{inner: openaiProvider.New(cfg)}
}

func (c *Client) Name() string      { return "minimax" }
func (c *Client) ContextLimit() int { return 204800 }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "MiniMax-M2.7", Name: "MiniMax M2.7", ContextLimit: 204800},
		{ID: "MiniMax-M2.7-highspeed", Name: "MiniMax M2.7 Highspeed", ContextLimit: 204800},
		{ID: "MiniMax-M2.5", Name: "MiniMax M2.5", ContextLimit: 204800},
		{ID: "MiniMax-M2.5-highspeed", Name: "MiniMax M2.5 Highspeed", ContextLimit: 204800},
		{ID: "MiniMax-M2.1", Name: "MiniMax M2.1", ContextLimit: 204800},
		{ID: "MiniMax-M2.1-highspeed", Name: "MiniMax M2.1 Highspeed", ContextLimit: 204800},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return c.inner.TokenCount(messages)
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return c.inner.Complete(ctx, req)
}
