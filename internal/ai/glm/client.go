package glm

import (
	"context"

	"github.com/iSundram/OweCode/internal/ai"
	openaiProvider "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "https://open.bigmodel.cn/api/coding/paas/v4"

// Client implements ai.Provider using GLM/Zhipu's OpenAI-compatible API.
type Client struct {
	inner *openaiProvider.Client
}

func New(cfg ai.ProviderConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "glm-5"
	}
	return &Client{inner: openaiProvider.New(cfg)}
}

func (c *Client) Name() string      { return "glm" }
func (c *Client) ContextLimit() int { return 128000 }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "glm-5", Name: "GLM-5", ContextLimit: 128000},
		{ID: "glm-5-turbo", Name: "GLM-5 Turbo", ContextLimit: 128000},
		{ID: "glm-4.7", Name: "GLM-4.7", ContextLimit: 128000},
		{ID: "glm-4.7-flashx", Name: "GLM-4.7 FlashX", ContextLimit: 128000},
		{ID: "glm-4.6", Name: "GLM-4.6", ContextLimit: 128000},
		{ID: "codegeex-4", Name: "CodeGeeX-4", ContextLimit: 128000},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return c.inner.TokenCount(messages)
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return c.inner.Complete(ctx, req)
}
