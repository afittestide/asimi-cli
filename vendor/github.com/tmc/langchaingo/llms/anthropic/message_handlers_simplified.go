package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"

	anthropicclient "github.com/tmc/langchaingo/llms/anthropic/internal/anthropicclient"
	"github.com/tmc/langchaingo/llms"
)

// handleAIMessageSimplified - Cleaner version using type switch
func handleAIMessageSimplified(msg llms.MessageContent) (anthropicclient.ChatMessage, error) {
	contents := make([]anthropicclient.Content, 0, len(msg.Parts))

	for _, part := range msg.Parts {
		switch p := part.(type) {
		case llms.TextContent:
			// Only add non-empty text
			if text := strings.TrimSpace(p.Text); text != "" {
				contents = append(contents, &anthropicclient.TextContent{
					Type: "text",
					Text: p.Text, // Keep original spacing
				})
			}

		case llms.ToolCall:
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(p.FunctionCall.Arguments), &input); err != nil {
				return anthropicclient.ChatMessage{},
					fmt.Errorf("anthropic: failed to unmarshal tool call arguments: %w", err)
			}
			contents = append(contents, anthropicclient.ToolUseContent{
				Type:  "tool_use",
				ID:    p.ID,
				Name:  p.FunctionCall.Name,
				Input: input,
			})

		default:
			return anthropicclient.ChatMessage{},
				fmt.Errorf("anthropic: invalid AI message part type: %T", part)
		}
	}

	if len(contents) == 0 {
		return anthropicclient.ChatMessage{},
			fmt.Errorf("anthropic: AI message has no valid content")
	}

	return anthropicclient.ChatMessage{
		Role:    RoleAssistant,
		Content: contents,
	}, nil
}

// handleToolMessageSimplified - More direct approach
func handleToolMessageSimplified(msg llms.MessageContent) (anthropicclient.ChatMessage, error) {
	contents := make([]anthropicclient.Content, 0, len(msg.Parts))

	for _, part := range msg.Parts {
		resp, ok := part.(llms.ToolCallResponse)
		if !ok {
			return anthropicclient.ChatMessage{},
				fmt.Errorf("anthropic: expected ToolCallResponse, got %T", part)
		}

		contents = append(contents, anthropicclient.ToolResultContent{
			Type:      "tool_result",
			ToolUseID: resp.ToolCallID,
			Content:   resp.Content,
		})
	}

	if len(contents) == 0 {
		return anthropicclient.ChatMessage{},
			fmt.Errorf("anthropic: tool message has no responses")
	}

	return anthropicclient.ChatMessage{
		Role:    RoleUser,
		Content: contents,
	}, nil
}