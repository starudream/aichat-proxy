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
	Model    string                   `json:"model"`
	Messages []*ChatCompletionMessage `json:"messages"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResp struct {
	Id      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []*ChatCompletionChoice `json:"choices"`
}

type ChatCompletionChoice struct {
	Index        int64                  `json:"index"`
	Message      *ChatCompletionMessage `json:"message"`
	FinishReason string                 `json:"finish_reason,omitempty"`
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
//	@router			/chat/completions [post]
//	@router			/v1/chat/completions [post]
//	@summary		Chat Completions
//	@description	Follows the exact same API spec as `https://platform.openai.com/docs/api-reference/chat`
//	@tags			2_chat
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

	buf := &bytes.Buffer{}
	if err := chatPrompt.Execute(buf, req); err != nil {
		return err
	}

	hdr, err := browser.B().HandleDoubao(buf.String())
	if err != nil {
		return err
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Status(200).Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for {
			msg, ok := <-hdr.Ch
			if !ok {
				return
			}
			event := json.MustMarshalToString(&ChatCompletionResp{
				Id:      hdr.Id,
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []*ChatCompletionChoice{{
					Index: cast.To[int64](msg.Id),
					Message: &ChatCompletionMessage{
						Role:    "assistant",
						Content: msg.Text,
					},
					FinishReason: msg.Finish,
				}},
			})
			_, err = fmt.Fprintf(w, "data: %s\n\n", event)
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
