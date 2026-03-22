package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	gogpt "github.com/sashabaranov/go-openai"

	"github.com/iSundram/OweCode/internal/ai"
)

// openaiStreamResponse implements ai.CompletionResponse for OpenAI streaming.
// Tool calls and usage are finalized when the SSE stream ends (before Done chunk).
type openaiStreamResponse struct {
	ch         <-chan ai.Chunk
	toolCalls  *[]ai.ToolCall
	stopReason *ai.StopReason
	usage      *ai.Usage
}

func (r *openaiStreamResponse) Stream() <-chan ai.Chunk { return r.ch }
func (r *openaiStreamResponse) ToolCalls() []ai.ToolCall {
	if r.toolCalls == nil {
		return nil
	}
	return *r.toolCalls
}
func (r *openaiStreamResponse) StopReason() ai.StopReason { return *r.stopReason }
func (r *openaiStreamResponse) Usage() ai.Usage {
	if r.usage == nil {
		return ai.Usage{}
	}
	return *r.usage
}

type toolCallAcc struct {
	id   string
	name string
	args strings.Builder
}

type toolCallAccumulator struct {
	byIndex map[int]*toolCallAcc
}

func (a *toolCallAccumulator) addDelta(delta gogpt.ChatCompletionStreamChoiceDelta) {
	if delta.FunctionCall != nil {
		if a.byIndex == nil {
			a.byIndex = make(map[int]*toolCallAcc)
		}
		t := a.byIndex[0]
		if t == nil {
			t = &toolCallAcc{}
			a.byIndex[0] = t
		}
		if delta.FunctionCall.Name != "" {
			t.name = delta.FunctionCall.Name
		}
		t.args.WriteString(delta.FunctionCall.Arguments)
	}
	for _, tc := range delta.ToolCalls {
		if a.byIndex == nil {
			a.byIndex = make(map[int]*toolCallAcc)
		}
		idx := 0
		if tc.Index != nil {
			idx = *tc.Index
		}
		t := a.byIndex[idx]
		if t == nil {
			t = &toolCallAcc{}
			a.byIndex[idx] = t
		}
		if tc.ID != "" {
			t.id = tc.ID
		}
		if tc.Function.Name != "" {
			t.name = tc.Function.Name
		}
		t.args.WriteString(tc.Function.Arguments)
	}
}

func (a *toolCallAccumulator) build() []ai.ToolCall {
	if len(a.byIndex) == 0 {
		return nil
	}
	indices := make([]int, 0, len(a.byIndex))
	for i := range a.byIndex {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	out := make([]ai.ToolCall, 0, len(indices))
	for _, i := range indices {
		t := a.byIndex[i]
		var args map[string]any
		if s := strings.TrimSpace(t.args.String()); s != "" {
			_ = json.Unmarshal([]byte(s), &args)
		}
		if args == nil {
			args = map[string]any{}
		}
		out = append(out, ai.ToolCall{
			ID:   t.id,
			Name: t.name,
			Args: args,
		})
	}
	return out
}

func mapUsageFromGPT(u *gogpt.Usage) ai.Usage {
	if u == nil {
		return ai.Usage{}
	}
	return ai.Usage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
}

func (c *Client) streamComplete(ctx context.Context, req gogpt.ChatCompletionRequest) (ai.CompletionResponse, error) {
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan ai.Chunk, 64)
	toolCalls := make([]ai.ToolCall, 0)
	stopReason := ai.StopReasonEnd
	usage := ai.Usage{}

	toolCallsPtr := &toolCalls
	stopPtr := &stopReason
	usagePtr := &usage

	go func() {
		defer close(ch)
		defer stream.Close()

		var acc toolCallAccumulator
		for {
			resp, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					*toolCallsPtr = acc.build()
					if len(*toolCallsPtr) > 0 && *stopPtr == ai.StopReasonEnd {
						*stopPtr = ai.StopReasonTools
					}
					ch <- ai.Chunk{Done: true}
					return
				}
				ch <- ai.Chunk{Error: err, Done: true}
				return
			}

			if resp.Usage != nil {
				*usagePtr = mapUsageFromGPT(resp.Usage)
			}
			if len(resp.Choices) == 0 {
				continue
			}

			choice := resp.Choices[0]
			if choice.FinishReason != "" {
				*stopPtr = mapStopReason(string(choice.FinishReason))
			}

			d := choice.Delta
			acc.addDelta(d)

			if d.Content != "" {
				ch <- ai.Chunk{Text: d.Content}
			}
			if d.Refusal != "" {
				ch <- ai.Chunk{Text: d.Refusal}
			}
		}
	}()

	return &openaiStreamResponse{
		ch:         ch,
		toolCalls:  toolCallsPtr,
		stopReason: stopPtr,
		usage:      usagePtr,
	}, nil
}
