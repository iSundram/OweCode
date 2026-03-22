package openai

import (
	"encoding/json"
	"strings"

	gogpt "github.com/sashabaranov/go-openai"

	"github.com/iSundram/OweCode/internal/ai"
)

// ChatMessagesFromRequest converts the agent conversation to OpenAI chat messages,
// including assistant tool_calls and tool role results.
func ChatMessagesFromRequest(req ai.CompletionRequest) []gogpt.ChatCompletionMessage {
	out := make([]gogpt.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		out = append(out, messageToOpenAI(m))
	}
	return out
}

func messageToOpenAI(m ai.Message) gogpt.ChatCompletionMessage {
	if m.Role == ai.RoleTool {
		for _, p := range m.Content {
			if p.Type == ai.ContentTypeToolResult && p.ToolResult != nil {
				return gogpt.ChatCompletionMessage{
					Role:       "tool",
					ToolCallID: p.ToolResult.ToolCallID,
					Content:    p.ToolResult.Content,
				}
			}
		}
		return gogpt.ChatCompletionMessage{Role: "tool", Content: ""}
	}

	var msg gogpt.ChatCompletionMessage
	msg.Role = string(m.Role)
	var textParts []string
	var calls []gogpt.ToolCall
	for _, p := range m.Content {
		switch p.Type {
		case ai.ContentTypeText:
			textParts = append(textParts, p.Text)
		case ai.ContentTypeToolCall:
			if p.ToolCall != nil {
				argsJSON, _ := json.Marshal(p.ToolCall.Args)
				calls = append(calls, gogpt.ToolCall{
					ID:   p.ToolCall.ID,
					Type: gogpt.ToolTypeFunction,
					Function: gogpt.FunctionCall{
						Name:      p.ToolCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
	}
	msg.Content = strings.Join(textParts, "")
	if len(calls) > 0 {
		msg.ToolCalls = calls
	}
	return msg
}

// ToolsFromSchemas converts tool schemas to OpenAI tool definitions.
func ToolsFromSchemas(schemas []ai.ToolSchema) []gogpt.Tool {
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
