package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/iSundram/OweCode/internal/ai"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	defaultModel   = "claude-sonnet-4-6"
	anthropicVer   = "2023-06-01"
	maxRetries     = 7
	maxBackoff     = 60 * time.Second
)

// apiError holds a parsed HTTP error from the Anthropic API.
type apiError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *apiError) Error() string {
	return fmt.Sprintf("anthropic: status %d: %s", e.StatusCode, e.Message)
}

// isRetryable reports whether this error should trigger a retry.
func isRetryable(err error) bool {
	ae, ok := err.(*apiError)
	if !ok {
		return false
	}
	switch ae.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// addJitter adds up to 25% random jitter to a duration.
func addJitter(d time.Duration) time.Duration {
	jitter := time.Duration(rand.Int63n(int64(d / 4)))
	return d + jitter
}

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
		limit:      modelContextLimit(model),
	}
}

func (c *Client) Name() string      { return "anthropic" }
func (c *Client) ContextLimit() int { return c.limit }

func (c *Client) Models(_ context.Context) ([]ai.Model, error) {
	return []ai.Model{
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", ContextLimit: 1000000},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", ContextLimit: 1000000},
		{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", ContextLimit: 200000},
	}, nil
}

func (c *Client) TokenCount(messages []ai.Message) (int, error) {
	return ai.ApproximateTokenCount(messages), nil
}

// anthropicTool describes a tool for the Anthropic API.
type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// anthropicToolUse is the tool_use content block in a response.
type anthropicToolUse struct {
	Type  string         `json:"type"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// anthropicToolResult is a tool_result block sent back to the model.
type anthropicToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

type anthropicContentBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
	IsError   bool           `json:"is_error,omitempty"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// buildMessages converts ai.Messages to Anthropic format, handling tool results.
func buildMessages(msgs []ai.Message) []anthropicMessage {
	var result []anthropicMessage
	for _, m := range msgs {
		if m.Role == ai.RoleSystem {
			continue
		}
		var blocks []anthropicContentBlock
		for _, p := range m.Content {
			switch p.Type {
			case ai.ContentTypeText:
				if p.Text != "" {
					blocks = append(blocks, anthropicContentBlock{Type: "text", Text: p.Text})
				}
			case ai.ContentTypeToolCall:
				if p.ToolCall != nil {
					inputJSON, _ := json.Marshal(p.ToolCall.Args)
					var inputMap map[string]any
					_ = json.Unmarshal(inputJSON, &inputMap)
					blocks = append(blocks, anthropicContentBlock{
						Type:  "tool_use",
						ID:    p.ToolCall.ID,
						Name:  p.ToolCall.Name,
						Input: inputMap,
					})
				}
			case ai.ContentTypeToolResult:
				if p.ToolResult != nil {
					blocks = append(blocks, anthropicContentBlock{
						Type:      "tool_result",
						ToolUseID: p.ToolResult.ToolCallID,
						Content:   p.ToolResult.Content,
						IsError:   p.ToolResult.IsError,
					})
				}
			}
		}
		if len(blocks) == 0 {
			continue
		}
		// Tool results must be in a "user" role message
		role := string(m.Role)
		if m.Role == ai.RoleTool {
			role = "user"
		}
		result = append(result, anthropicMessage{Role: role, Content: blocks})
	}
	return result
}

func (c *Client) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	msgs := buildMessages(req.Messages)

	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = 8192
	}

	// Build tools list
	var tools []anthropicTool
	for _, t := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	areqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTok,
		System:    req.System,
		Messages:  msgs,
		Tools:     tools,
		Stream:    req.Stream,
	}

	body, err := json.Marshal(areqBody)
	if err != nil {
		return nil, err
	}

	backoff := time.Second
	for attempt := 0; attempt < maxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("x-api-key", c.apiKey)
		httpReq.Header.Set("anthropic-version", anthropicVer)
		httpReq.Header.Set("content-type", "application/json")
		if req.Stream {
			httpReq.Header.Set("accept", "text/event-stream")
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			ae := &apiError{StatusCode: resp.StatusCode, Message: string(data)}
			// Parse Retry-After header if present
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					ae.RetryAfter = time.Duration(secs) * time.Second
				}
			}

			if !isRetryable(ae) || attempt == maxRetries-1 {
				return nil, ae
			}

			wait := backoff
			if ae.RetryAfter > 0 {
				wait = ae.RetryAfter
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(addJitter(wait)):
			}
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		if req.Stream {
			return c.parseStream(resp.Body), nil
		}

		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var ar anthropicResponse
		if err := json.Unmarshal(data, &ar); err != nil {
			return nil, err
		}

		return parseNonStreamResponse(&ar), nil
	}
	return nil, fmt.Errorf("anthropic: max retries (%d) exceeded", maxRetries)
}

func parseNonStreamResponse(ar *anthropicResponse) ai.CompletionResponse {
	var text string
	var toolCalls []ai.ToolCall
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			text += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, ai.ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: block.Input,
			})
		}
	}

	stop := ai.StopReasonEnd
	if ar.StopReason == "tool_use" {
		stop = ai.StopReasonTools
	} else if ar.StopReason == "max_tokens" {
		stop = ai.StopReasonLength
	}

	usage := ai.Usage{
		InputTokens:  ar.Usage.InputTokens,
		OutputTokens: ar.Usage.OutputTokens,
		TotalTokens:  ar.Usage.InputTokens + ar.Usage.OutputTokens,
	}
	return ai.NewStaticResponse(text, toolCalls, stop, usage)
}

// SSE event types from Anthropic streaming API.
type sseEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type streamDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type streamContentBlockDelta struct {
	Type  string      `json:"type"`
	Index int         `json:"index"`
	Delta streamDelta `json:"delta"`
}

type streamContentBlockStart struct {
	Type         string                `json:"type"`
	Index        int                   `json:"index"`
	ContentBlock anthropicContentBlock `json:"content_block"`
}

type streamMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type streamMessageStart struct {
	Type    string `json:"type"`
	Message struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// toolUseBlock accumulates a streaming tool_use content block.
type toolUseBlock struct {
	id    string
	name  string
	input strings.Builder
}

// streamState holds mutable state for SSE stream parsing.
type streamState struct {
	ch         chan<- ai.Chunk
	toolBlocks map[int]*toolUseBlock
	toolCalls  *[]ai.ToolCall
	stopReason *ai.StopReason
	usage      *ai.Usage
}

// parseStream reads SSE from the response body and returns a streaming response.
func (c *Client) parseStream(body io.ReadCloser) ai.CompletionResponse {
	ch := make(chan ai.Chunk, 64)
	toolCalls := make([]ai.ToolCall, 0)
	stopReason := ai.StopReasonEnd
	usage := ai.Usage{}

	state := &streamState{
		ch:         ch,
		toolBlocks: map[int]*toolUseBlock{},
		toolCalls:  &toolCalls,
		stopReason: &stopReason,
		usage:      &usage,
	}

	go func() {
		defer body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		var eventType string
		var dataLines []string

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if eventType != "" && len(dataLines) > 0 {
					data := strings.Join(dataLines, "\n")
					processSSEEvent(eventType, data, state)
				}
				eventType = ""
				dataLines = nil
				continue
			}
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			}
		}

		// Finalize any tool_use blocks accumulated during streaming
		for _, tb := range state.toolBlocks {
			var args map[string]any
			if err := json.Unmarshal([]byte(tb.input.String()), &args); err != nil {
				args = map[string]any{}
			}
			*state.toolCalls = append(*state.toolCalls, ai.ToolCall{
				ID:   tb.id,
				Name: tb.name,
				Args: args,
			})
		}

		ch <- ai.Chunk{Done: true}
	}()

	return &streamingResponse{
		ch:         ch,
		toolCalls:  &toolCalls,
		stopReason: &stopReason,
		usage:      &usage,
	}
}

func processSSEEvent(eventType, data string, state *streamState) {
	switch eventType {
	case "content_block_start":
		var ev streamContentBlockStart
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			if ev.ContentBlock.Type == "tool_use" {
				state.toolBlocks[ev.Index] = &toolUseBlock{
					id:   ev.ContentBlock.ID,
					name: ev.ContentBlock.Name,
				}
			}
		}
	case "content_block_delta":
		var ev streamContentBlockDelta
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			switch ev.Delta.Type {
			case "text_delta":
				if ev.Delta.Text != "" {
					state.ch <- ai.Chunk{Text: ev.Delta.Text}
				}
			case "input_json_delta":
				if tb, ok := state.toolBlocks[ev.Index]; ok {
					tb.input.WriteString(ev.Delta.Text)
				}
			}
		}
	case "message_delta":
		var ev streamMessageDelta
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			state.usage.OutputTokens = ev.Usage.OutputTokens
			switch ev.Delta.StopReason {
			case "tool_use":
				*state.stopReason = ai.StopReasonTools
			case "max_tokens":
				*state.stopReason = ai.StopReasonLength
			default:
				*state.stopReason = ai.StopReasonEnd
			}
		}
	case "message_start":
		var ev streamMessageStart
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			state.usage.InputTokens = ev.Message.Usage.InputTokens
		}
	}
}

// streamingResponse implements ai.CompletionResponse for SSE streams.
// It collects all chunks first, then provides the full picture to the agent.
type streamingResponse struct {
	ch         <-chan ai.Chunk
	toolCalls  *[]ai.ToolCall
	stopReason *ai.StopReason
	usage      *ai.Usage
}

func (r *streamingResponse) Stream() <-chan ai.Chunk   { return r.ch }
func (r *streamingResponse) ToolCalls() []ai.ToolCall  { return *r.toolCalls }
func (r *streamingResponse) StopReason() ai.StopReason { return *r.stopReason }
func (r *streamingResponse) Usage() ai.Usage           { return *r.usage }

func modelContextLimit(model string) int {
	switch model {
	case "claude-opus-4-6", "claude-sonnet-4-6":
		return 1000000
	default:
		return 200000
	}
}
