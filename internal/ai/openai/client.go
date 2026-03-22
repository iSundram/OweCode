package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gogpt "github.com/sashabaranov/go-openai"

	"github.com/iSundram/OweCode/internal/ai"
)

const defaultModel = "gpt-5.4"

// Client implements ai.Provider using the OpenAI API.
type Client struct {
	client *gogpt.Client
	model  string
	limit  int
}

// New creates a new OpenAI client.
func New(cfg ai.ProviderConfig) *Client {
	c := gogpt.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	if cfg.OrgID != "" {
		c.OrgID = cfg.OrgID
	}
	model := cfg.DefaultModel
	if model == "" {
		model = defaultModel
	}
	return &Client{
		client: gogpt.NewClientWithConfig(c),
		model:  model,
		limit:  modelContextLimit(model),
	}
}

func (c *Client) Name() string      { return "openai" }
func (c *Client) ContextLimit() int { return c.limit }

func (c *Client) Models(ctx context.Context) ([]ai.Model, error) {
	resp, err := c.client.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	// Map of well-known models to provide better metadata
	metadata := map[string]struct {
		Name  string
		Limit int
	}{
		"gpt-5.4":           {Name: "GPT-5.4", Limit: 1000000},
		"gpt-5.4-pro":       {Name: "GPT-5.4 Pro", Limit: 1000000},
		"gpt-5.4-mini":      {Name: "GPT-5.4 Mini", Limit: 400000},
		"gpt-5.4-nano":      {Name: "GPT-5.4 Nano", Limit: 400000},
		"gpt-5":             {Name: "GPT-5", Limit: 128000},
		"gpt-5-mini":        {Name: "GPT-5 Mini", Limit: 128000},
		"gpt-5-nano":        {Name: "GPT-5 Nano", Limit: 128000},
		"gpt-4.1":           {Name: "GPT-4.1", Limit: 128000},
		"gpt-4o":            {Name: "GPT-4o", Limit: 128000},
		"gpt-4o-mini":       {Name: "GPT-4o Mini", Limit: 128000},
		"o1":                {Name: "o1", Limit: 200000},
		"o1-mini":           {Name: "o1-mini", Limit: 128000},
		"o1-preview":        {Name: "o1-preview", Limit: 128000},
		"o3-mini":           {Name: "o3-mini", Limit: 200000},
		"gpt-4-turbo":       {Name: "GPT-4 Turbo", Limit: 128000},
		"gpt-3.5-turbo":     {Name: "GPT-3.5 Turbo", Limit: 16385},
		"deepseek-chat":     {Name: "DeepSeek V3", Limit: 128000},
		"deepseek-reasoner": {Name: "DeepSeek R1", Limit: 128000},
	}

	models := make([]ai.Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		// Filter for common chat models to reduce noise
		id := m.ID
		if !isChatModel(id) {
			continue
		}

		name := id
		limit := 128000
		if meta, ok := metadata[id]; ok {
			name = meta.Name
			limit = meta.Limit
		}

		models = append(models, ai.Model{
			ID:           id,
			Name:         name,
			ContextLimit: limit,
		})
	}
	return models, nil
}

func isChatModel(id string) bool {
	// Standard OpenAI prefixes
	prefixes := []string{"gpt-5", "gpt-4", "gpt-3.5", "o1", "o3", "chatgpt", "gpt-realtime"}
	for _, p := range prefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	// Permissive for OpenRouter/Local providers (usually have / or are common prefixes)
	if strings.Contains(id, "/") {
		return true
	}
	extraPrefixes := []string{"claude-", "gemini-", "deepseek-", "llama-", "mistral-", "grok-", "glm-", "kimi-", "minimax-"}
	for _, p := range extraPrefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

func modelContextLimit(model string) int {
	switch model {
	case "gpt-5.4", "gpt-5.4-pro":
		return 1000000
	case "gpt-5.4-mini", "gpt-5.4-nano":
		return 400000
	case "o1", "o3-mini":
		return 200000
	default:
		return 128000
	}
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return ai.ApproximateTokenCount(messages), nil
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	msgs := ChatMessagesFromRequest(req)

	// Handle reasoning models which might not support system messages in the same way
	// or require specific parameters.
	isReasoning := strings.HasPrefix(c.model, "o1") || strings.HasPrefix(c.model, "o3")

	if req.System != "" {
		role := "system"
		if isReasoning {
			// Some reasoning models prefer system instructions in the first user message
			// or as a 'developer' role in newer APIs. For now, keep as system but be aware.
			role = "system"
		}
		sys := gogpt.ChatCompletionMessage{Role: role, Content: req.System}
		msgs = append([]gogpt.ChatCompletionMessage{sys}, msgs...)
	}

	tools := ToolsFromSchemas(req.Tools)

	gptReq := gogpt.ChatCompletionRequest{
		Model:       c.model,
		Messages:    msgs,
		Tools:       tools,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}
	if req.Stream {
		gptReq.StreamOptions = &gogpt.StreamOptions{IncludeUsage: true}
	}

	// Reasoning models often don't support temperature
	if isReasoning {
		gptReq.Temperature = 0
	}

	if req.Stream {
		return c.streamComplete(ctx, gptReq)
	}
	return c.syncComplete(ctx, gptReq)
}

func (c *Client) syncComplete(ctx context.Context, req gogpt.ChatCompletionRequest) (ai.CompletionResponse, error) {
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}
	choice := resp.Choices[0]
	text := choice.Message.Content
	var toolCalls []ai.ToolCall
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		toolCalls = append(toolCalls, ai.ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}
	stop := mapStopReason(string(choice.FinishReason))
	usage := ai.Usage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}
	return ai.NewStaticResponse(text, toolCalls, stop, usage), nil
}

func mapStopReason(r string) ai.StopReason {
	switch r {
	case "tool_calls":
		return ai.StopReasonTools
	case "length":
		return ai.StopReasonLength
	case "stop":
		return ai.StopReasonEnd
	default:
		return ai.StopReasonStopped
	}
}
