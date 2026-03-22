package openai

import (
	"testing"

	gogpt "github.com/sashabaranov/go-openai"

	"github.com/iSundram/OweCode/internal/ai"
)

func TestChatMessagesFromRequestToolTurn(t *testing.T) {
	req := ai.CompletionRequest{
		Messages: []ai.Message{
			ai.NewTextMessage(ai.RoleUser, "read foo"),
			{
				Role: ai.RoleAssistant,
				Content: []ai.ContentPart{
					{Type: ai.ContentTypeToolCall, ToolCall: &ai.ToolCall{
						ID: "call_1", Name: "read_file", Args: map[string]any{"path": "a.go"},
					}},
				},
			},
			{
				Role: ai.RoleTool,
				Content: []ai.ContentPart{{
					Type: ai.ContentTypeToolResult,
					ToolResult: &ai.ToolResult{ToolCallID: "call_1", Content: "package main"},
				}},
			},
		},
	}
	msgs := ChatMessagesFromRequest(req)
	if len(msgs) != 3 {
		t.Fatalf("len=%d", len(msgs))
	}
	if msgs[1].Role != "assistant" || len(msgs[1].ToolCalls) != 1 {
		t.Fatalf("assistant tool_calls: %+v", msgs[1])
	}
	if msgs[1].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("name: %q", msgs[1].ToolCalls[0].Function.Name)
	}
	if msgs[2].Role != "tool" || msgs[2].ToolCallID != "call_1" || msgs[2].Content != "package main" {
		t.Fatalf("tool message: %+v", msgs[2])
	}
}

func TestToolsFromSchemas(t *testing.T) {
	tools := ToolsFromSchemas([]ai.ToolSchema{{
		Name:        "x",
		Description: "d",
		Parameters:  map[string]any{"type": "object"},
	}})
	if len(tools) != 1 || tools[0].Type != gogpt.ToolTypeFunction {
		t.Fatalf("%+v", tools)
	}
}
