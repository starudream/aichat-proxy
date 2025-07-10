package router

import (
	"bufio"
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/browser"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

type ChatCompletionReq struct {
	// 模型 Id
	Model string `json:"model"`
	// 消息列表
	Messages []*ChatCompletionMessage `json:"messages"`
	// 是否流式
	Stream bool `json:"stream,omitempty"`
}

type ChatCompletionMessage struct {
	// 角色
	Role string `json:"role"`
	// 内容
	Content string `json:"content"`
	// 推理内容（仅响应）
	ReasoningContent string `json:"reasoning_content"`
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
func hdrChatCompletions(c *Ctx) error {
	req := &ChatCompletionReq{}
	if err := c.BodyParser(req); err != nil {
		return err
	}

	if !browser.ExistModel(req.Model) {
		return NewError(404, "model not found: %q", req.Model)
	}

	buf := &bytes.Buffer{}
	if err := chatPrompt.Execute(buf, req); err != nil {
		return err
	}

	created := time.Now().Unix()

	hdr, err := browser.B().HandleChat(req.Model, buf.String())
	if err != nil {
		return err
	}

	if !req.Stream {
		content, reason := hdr.WaitFinish(c.Context())
		return c.Status(200).JSON(&ChatCompletionResp{
			Id:      hdr.Id,
			Object:  "chat.completion",
			Created: created,
			Model:   req.Model,
			Choices: []*ChatCompletionChoice{{
				Message: &ChatCompletionMessage{
					Role:             "assistant",
					Content:          content,
					ReasoningContent: reason,
				},
				FinishReason: "stop",
			}},
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Status(200).Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		tag := ""
		for {
			msg, ok := <-hdr.Ch
			if !ok {
				return
			}
			if msg.FinishReason == "" {
				if msg.ReasoningTag != "" {
					tag = msg.ReasoningTag
					continue
				}
				delta := &ChatCompletionMessage{Role: "assistant"}
				if tag == "1" {
					delta.ReasoningContent = msg.Content
				} else {
					delta.Content = msg.Content
				}
				event := json.MustMarshalToString(&ChatCompletionResp{
					Id:      hdr.Id,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   req.Model,
					Choices: []*ChatCompletionChoice{{
						Index: cast.To[int64](msg.Index),
						Delta: delta,
					}},
				})
				_, err = fmt.Fprintf(w, "data: %s\n\n", event)
			} else {
				_, err = fmt.Fprintf(w, "[DONE]\n")
			}
			if err != nil {
				logger.Ctx(c.UserContext()).Error().Err(err).Msg("write sse data error")
				return
			}
			err = w.Flush()
			if err != nil {
				logger.Ctx(c.UserContext()).Error().Err(err).Msg("flush sse data error")
				return
			}
		}
	})

	return nil
}
