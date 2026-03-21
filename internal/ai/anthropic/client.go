package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/iSundram/OweCode/internal/ai"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	defaultModel   = "claude-3-5-sonnet-20241022"
	anthropicVer   = "2023-06-01"
)

// Client implements ai.Provider for Anthropic Claude.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	model      string
	limit      int
}

func New(cfg ai.ProviderConfig) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	model := cfg.DefaultModel
	if model == "" {
		model = defaultModel
	}
	return &Client{
		httpClient: &http.Client{},
		apiKey:     cfg.APIKey,
		baseURL:    base,
		model:      model,
		limit:      200000,
	}
}

func (c *Client) Name() string      { return "anthropic" }
func (c *Client) ContextLimit() int { return c.limit }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", ContextLimit: 200000},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", ContextLimit: 200000},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", ContextLimit: 200000},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	// Approximate: 1 token ≈ 4 characters.
	total := 0
	for _, m := range messages {
		total += len(m.TextContent()) / 4
	}
	return total, nil
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	msgs := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == ai.RoleSystem {
			continue
		}
		msgs = append(msgs, anthropicMessage{
			Role:    string(m.Role),
			Content: []anthropicContent{{Type: "text", Text: m.TextContent()}},
		})
	}

	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = 4096
	}

	body, err := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTok,
		System:    req.System,
		Messages:  msgs,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVer)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, data)
	}

	var ar anthropicResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, err
	}

	text := ""
	if len(ar.Content) > 0 {
		text = ar.Content[0].Text
	}
	usage := ai.Usage{
		InputTokens:  ar.Usage.InputTokens,
		OutputTokens: ar.Usage.OutputTokens,
		TotalTokens:  ar.Usage.InputTokens + ar.Usage.OutputTokens,
	}
	return ai.NewStaticResponse(text, nil, ai.StopReasonEnd, usage), nil
}
