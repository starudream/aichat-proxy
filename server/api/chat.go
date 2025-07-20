package api

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/browser"
	"github.com/starudream/aichat-proxy/server/internal/errx"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
	"github.com/starudream/aichat-proxy/server/tiktoken"
)

type ChatCompletionReq struct {
	// 模型 Id
	Model string `json:"model" validate:"required"`
	// 消息列表
	Messages []*ChatCompletionMessage `json:"messages"`
	// 是否流式
	Stream bool `json:"stream,omitempty"`
	// 是否启用推理
	EnableThinking bool `json:"enable_thinking,omitempty"`
	// 推理配置
	Thinking *ChatCompletionThinking `json:"thinking,omitempty"`
}

type ChatCompletionMessage struct {
	// 角色
	Role string `json:"role" validate:"required"`
	// 内容
	Content string `json:"content"`
	// 推理内容（仅响应）
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type ChatCompletionThinking struct {
	// auto：自动思考模式
	// enabled：开启思考模式
	// disabled：关闭思考模式
	Type string `json:"type"`
}

type ChatCompletionResp struct {
	// 请求的唯一标识
	Id string `json:"id"`
	// 响应类型
	Object string `json:"object"`
	// 请求创建的时间戳（秒级）
	Created int64 `json:"created"`
	// 模型 Id
	Model string `json:"model"`
	// 模型输出内容
	Choices []*ChatCompletionChoice `json:"choices"`
	// 用量
	Usage *ChatCompletionUsage `json:"usage,omitempty"`
}

type ChatCompletionChoice struct {
	// 消息索引
	Index int64 `json:"index"`
	// 模型输出消息列表（非流式）
	Message *ChatCompletionMessage `json:"message,omitempty"`
	// 模型输出的增量内容（流式）
	Delta *ChatCompletionMessage `json:"delta,omitempty"`
	// 模型停止输出原因
	FinishReason string `json:"finish_reason,omitempty"`
}

type ChatCompletionUsage struct {
	// 总消耗 tokens
	TotalTokens int `json:"total_tokens"`
	// 输入 tokens
	PromptTokens int `json:"prompt_tokens"`
	// 输出 tokens
	CompletionTokens int `json:"completion_tokens"`
	// 输出 tokens
	CompletionTokensDetails *ChatCompletionTokens `json:"completion_tokens_details,omitempty"`
}

type ChatCompletionTokens struct {
	// 思维链 tokens
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

var chatPrompt = template.Must(template.New("").Parse(`
{{- range $index, $message := .Messages }}
{{- if gt $index 0 }}{{ print "---\n" }}{{ end }}
{{- if eq $message.Role "system" }}
{{- print "【系统】" }}
{{- else if eq $message.Role "user" }}
{{- print "【用户】" }}
{{- else if eq $message.Role "assistant" }}
{{- print "【助手】" }}
{{- else if eq $message.Role "tool" }}
{{- print "【工具】" }}
{{- end }}
{{ $message.Content }}
{{ end -}}
`))

// Chat Completions
//
//	@router			/v1/chat/completions [post]
//	@summary		Chat Completions
//	@description	Follows the exact same API spec as `https://platform.openai.com/docs/api-reference/chat`
//	@tags			chat
//	@security		ApiKeyAuth
//	@produce		json
//	@produce		text/event-stream
//	@param			*	body		ChatCompletionReq	true	"Request"
//	@success		200	{object}	ChatCompletionResp
func hdrChatCompletions(c Ctx) error {
	req := &ChatCompletionReq{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	if !browser.ExistModel(req.Model) {
		return errx.NotFound().WithMsgf("model not found: %s", req.Model)
	}

	buf := &bytes.Buffer{}
	if err := chatPrompt.Execute(buf, req); err != nil {
		return err
	}
	prompt := strings.TrimSpace(buf.String())
	promptN := tiktoken.NumTokens(prompt)

	ctx := c.Request().Context()
	unix := time.Now().Unix()

	options := browser.HandleChatOptions{}
	if req.EnableThinking {
		options.Thinking = "enabled"
	} else if req.Thinking != nil {
		options.Thinking = req.Thinking.Type
	}

	hdr, err := browser.B().HandleChat(ctx, req.Model, prompt, options)
	if err != nil {
		return err
	}

	if !req.Stream {
		content, reason := hdr.WaitFinish(ctx)
		contentN, reasonN := tiktoken.NumTokens(content), tiktoken.NumTokens(reason)
		return c.JSON(200, &ChatCompletionResp{
			Id:      hdr.Id,
			Object:  "chat.completion",
			Created: unix,
			Model:   req.Model,
			Choices: []*ChatCompletionChoice{{
				Message: &ChatCompletionMessage{
					Role:             "assistant",
					Content:          content,
					ReasoningContent: reason,
				},
				FinishReason: "stop",
			}},
			Usage: &ChatCompletionUsage{
				TotalTokens:      promptN + contentN + reasonN,
				PromptTokens:     promptN,
				CompletionTokens: contentN + reasonN,
				CompletionTokensDetails: &ChatCompletionTokens{
					ReasoningTokens: reasonN,
				},
			},
		})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	contentB, reasonB := &bytes.Buffer{}, &bytes.Buffer{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-hdr.Ch:
			if !ok {
				return nil
			}
			data := "[DONE]"
			if msg.FinishReason == "" {
				delta := &ChatCompletionMessage{Role: "assistant"}
				if msg.Content != "" {
					delta.Content = msg.Content
					contentB.WriteString(msg.Content)
				} else if msg.ReasoningContent != "" {
					delta.ReasoningContent = msg.ReasoningContent
					reasonB.WriteString(msg.ReasoningContent)
				} else {
					continue
				}
				data = json.MustMarshalToString(&ChatCompletionResp{
					Id:      hdr.Id,
					Object:  "chat.completion.chunk",
					Created: unix,
					Model:   req.Model,
					Choices: []*ChatCompletionChoice{{
						Index: cast.To[int64](msg.Index),
						Delta: delta,
					}},
				})
			} else {
				contentN, reasonN := tiktoken.NumTokens(contentB.String()), tiktoken.NumTokens(reasonB.String())
				data = json.MustMarshalToString(&ChatCompletionResp{
					Object:  "chat.completion.chunk",
					Created: unix,
					Model:   req.Model,
					Usage: &ChatCompletionUsage{
						TotalTokens:      promptN + contentN + reasonN,
						PromptTokens:     promptN,
						CompletionTokens: contentN + reasonN,
						CompletionTokensDetails: &ChatCompletionTokens{
							ReasoningTokens: reasonN,
						},
					},
				})
			}
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				logger.Ctx(ctx).Error().Err(err).Msg("write sse data error")
				return err
			}
			w.Flush()
		}
	}
}
