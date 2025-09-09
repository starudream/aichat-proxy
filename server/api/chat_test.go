package api

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/tiktoken"
)

func TestChatPrompt(t *testing.T) {
	req := &ChatCompletionReq{
		Model: "test",
		Messages: []*ChatCompletionMessage{
			{
				Role: "system",
				Content: &ChatCompletionMessageContent{
					StringValue: "hello",
				},
			},
			{
				Role: "user",
				Content: &ChatCompletionMessageContent{
					ListValue: []*ChatCompletionMessageContentPart{
						{
							Type: "text",
							Text: "world1",
						},
						{
							Type: "text",
							Text: "world2",
						},
					},
				},
			},
			{
				Role: "assistant",
				Content: &ChatCompletionMessageContent{
					ListValue: []*ChatCompletionMessageContentPart{
						{
							Type: "text",
							Text: "world3",
						},
						{
							Type: "text",
							Text: "world4",
						},
					},
				},
			},
		},
		Stream: true,
		Tools: []*ChatCompletionTool{
			{
				Type: "function",
				Function: &ChatCompletionToolFunction{
					Name:        "Task",
					Description: "Launch a new agent that has access to the following tools",
					Parameters:  json.MustUnmarshalTo[any](`{"type": "object","properties": {"description": {"type": "string","description": "A short (3-5 word) description of the task"},"prompt": {"type": "string","description": "The task for the agent to perform"}},"required": ["description", "prompt"],"additionalProperties": false,"$schema": "http://json-schema.org/draft-07/schema#"}`),
				},
			},
			{
				Type: "function",
				Function: &ChatCompletionToolFunction{
					Name:        "Bash",
					Description: "Executes a given bash command",
					Parameters:  json.MustUnmarshalTo[any](`{"type": "object","properties": {"command": {"type": "string","description": "The command to execute"},"timeout": {"type": "number","description": "Optional timeout in milliseconds (max 600000)"},"description": {"type": "string","description": " Clear, concise description of what this command does in 5-10 words. Examples:\nInput: ls\nOutput: Lists files in current directory\n\nInput: git status\nOutput: Shows working tree status\n\nInput: npm install\nOutput: Installs package dependencies\n\nInput: mkdir foo\nOutput: Creates directory 'foo'"}},"required": ["command"],"additionalProperties": false,"$schema": "http://json-schema.org/draft-07/schema#"}`),
				},
			},
		},
	}
	t.Log(json.MustMarshalToString(req))

	buf := &bytes.Buffer{}
	if err := chatPrompt.Execute(buf, req); err != nil {
		t.Fatal(err)
	}
	prompt := buf.String()
	fmt.Println("===")
	fmt.Println(prompt)
	fmt.Println("===")
	fmt.Println(tiktoken.NumTokens(prompt))
}
