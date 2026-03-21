package openai

import (
	"context"
	"encoding/json"
	"fmt"

	gogpt "github.com/sashabaranov/go-openai"

	"github.com/iSundram/OweCode/internal/ai"
)

const defaultModel = "gpt-4o"

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
		limit:  128000,
	}
}

func (c *Client) Name() string         { return "openai" }
func (c *Client) ContextLimit() int    { return c.limit }

func (c *Client) Models(ctx context.Context) ([]ai.Model, error) {
	resp, err := c.client.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	models := make([]ai.Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, ai.Model{ID: m.ID, Name: m.ID})
	}
	return models, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return ai.ApproximateTokenCount(messages), nil
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	msgs := convertMessages(req)
	if req.System != "" {
		sys := gogpt.ChatCompletionMessage{Role: "system", Content: req.System}
		msgs = append([]gogpt.ChatCompletionMessage{sys}, msgs...)
	}

	tools := convertTools(req.Tools)

	gptReq := gogpt.ChatCompletionRequest{
		Model:       c.model,
		Messages:    msgs,
		Tools:       tools,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
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

func (c *Client) streamComplete(ctx context.Context, req gogpt.ChatCompletionRequest) (ai.CompletionResponse, error) {
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan ai.Chunk, 64)
	go func() {
		defer close(ch)
		defer stream.Close()
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					ch <- ai.Chunk{Error: err, Done: true}
				} else {
					ch <- ai.Chunk{Done: true}
				}
				return
			}
			if len(resp.Choices) == 0 {
				continue
			}
			delta := resp.Choices[0].Delta
			ch <- ai.Chunk{Text: delta.Content}
		}
	}()
	return ai.NewChannelResponse(ch, ai.StopReasonEnd, ai.Usage{}), nil
}

func convertMessages(req ai.CompletionRequest) []gogpt.ChatCompletionMessage {
	var msgs []gogpt.ChatCompletionMessage
	for _, m := range req.Messages {
		role := string(m.Role)
		content := m.TextContent()
		msgs = append(msgs, gogpt.ChatCompletionMessage{Role: role, Content: content})
	}
	return msgs
}

func convertTools(schemas []ai.ToolSchema) []gogpt.Tool {
	if len(schemas) == 0 {
		return nil
	}
	tools := make([]gogpt.Tool, 0, len(schemas))
	for _, s := range schemas {
		params, _ := json.Marshal(s.Parameters)
		var def gogpt.FunctionDefinition
		def.Name = s.Name
		def.Description = s.Description
		_ = json.Unmarshal(params, &def.Parameters)
		tools = append(tools, gogpt.Tool{Type: gogpt.ToolTypeFunction, Function: &def})
	}
	return tools
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
