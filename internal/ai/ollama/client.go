package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	gogpt "github.com/sashabaranov/go-openai"

	"github.com/iSundram/OweCode/internal/ai"
	openaiCompat "github.com/iSundram/OweCode/internal/ai/openai"
)

const defaultBaseURL = "http://localhost:11434"

// Client implements ai.Provider for Ollama (local models).
type Client struct {
	httpClient *http.Client
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
		model = "llama3.2"
	}
	return &Client{
		httpClient: &http.Client{},
		baseURL:    base,
		model:      model,
		limit:      128000,
	}
}

func (c *Client) Name() string      { return "ollama" }
func (c *Client) ContextLimit() int { return c.limit }

type ollamaModel struct {
	Name string `json:"name"`
}

type ollamaModelsResponse struct {
	Models []ollamaModel `json:"models"`
}

func (c *Client) Models(ctx context.Context) ([]ai.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var mr ollamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, err
	}
	models := make([]ai.Model, 0, len(mr.Models))
	for _, m := range mr.Models {
		models = append(models, ai.Model{ID: m.Name, Name: m.Name})
	}
	return models, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return ai.ApproximateTokenCount(messages), nil
}

type ollamaChatRequest struct {
	Model    string                        `json:"model"`
	Messages []gogpt.ChatCompletionMessage `json:"messages"`
	Tools    []gogpt.Tool                  `json:"tools,omitempty"`
	Stream   bool                          `json:"stream"`
}

type ollamaResponse struct {
	Message struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done            bool `json:"done"`
	PromptEvalCount int  `json:"prompt_eval_count"`
	EvalCount       int  `json:"eval_count"`
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	msgs := openaiCompat.ChatMessagesFromRequest(req)
	if req.System != "" {
		msgs = append([]gogpt.ChatCompletionMessage{{Role: "system", Content: req.System}}, msgs...)
	}
	tools := openaiCompat.ToolsFromSchemas(req.Tools)

	body, err := json.Marshal(ollamaChatRequest{
		Model:    c.model,
		Messages: msgs,
		Tools:    tools,
		Stream:   false,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, data)
	}

	var or ollamaResponse
	if err := json.Unmarshal(data, &or); err != nil {
		return nil, err
	}

	var toolCalls []ai.ToolCall
	for _, tc := range or.Message.ToolCalls {
		var args map[string]any
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		if args == nil {
			args = map[string]any{}
		}
		toolCalls = append(toolCalls, ai.ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}

	stop := ai.StopReasonEnd
	if len(toolCalls) > 0 {
		stop = ai.StopReasonTools
	}

	usage := ai.Usage{
		InputTokens:  or.PromptEvalCount,
		OutputTokens: or.EvalCount,
		TotalTokens:  or.PromptEvalCount + or.EvalCount,
	}
	return ai.NewStaticResponse(or.Message.Content, "", toolCalls, stop, usage), nil
}
