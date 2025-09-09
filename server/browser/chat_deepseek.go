package browser

import (
	"strings"
	"sync/atomic"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatDeepseekHandler{})
}

type chatDeepseekHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	reasoning atomic.Bool
}

func (h *chatDeepseekHandler) Name() string {
	return "deepseek"
}

func (h *chatDeepseekHandler) URL() string {
	return "https://chat.deepseek.com"
}

func (h *chatDeepseekHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatDeepseekHandler) Input(prompt string) (err error) {
	h.log.Debug().Msg("wait for chat textarea")
	locText := h.page.Locator("textarea#chat-input")
	if err = locText.WaitFor(); err != nil {
		logger.Error().Err(err).Msg("wait for chat textarea error")
		return err
	}

	h.log.Debug().Msg("fill prompt to chat textarea")
	if err = locText.Fill(prompt); err != nil {
		logger.Error().Err(err).Msg("fill prompt to chat textarea error")
		return err
	}

	return nil
}

func (h *chatDeepseekHandler) Send() error {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.page.Locator(`input[type="file"] + div`)
	if err := locSend.WaitFor(); err != nil {
		logger.Error().Err(err).Msg("wait for chat send button error")
		return err
	}

	h.log.Debug().Msg("click chat send button")
	if err := locSend.Click(); err != nil {
		logger.Error().Err(err).Msg("click chat send button error")
		return err
	}

	return nil
}

type deepseekEvent struct {
	V any    `json:"v"`
	P string `json:"p,omitempty"`
	O string `json:"o,omitempty"`
}

type deepseekV struct {
	Id int `json:"id,omitempty"`
	// THINK or RESPONSE
	Type    string `json:"type,omitempty"`
	Content string `json:"content,omitempty"`
}

func (h *chatDeepseekHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimPrefix(s, "data:")
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*deepseekEvent](s)
	if err != nil {
		return nil
	}
	content := ""
	switch x := event.V.(type) {
	case string:
		content = x
	case []any:
		v, e := json.UnmarshalTo[[]*deepseekV](json.MustMarshal(x))
		if e != nil || len(v) == 0 {
			return nil
		}
		switch v[0].Type {
		case "THINK":
			h.reasoning.Store(true)
		case "RESPONSE":
			h.reasoning.Store(false)
		default:
			return nil
		}
		content = v[0].Content
	}
	if content == "" {
		return nil
	}
	if h.reasoning.Load() {
		return &ChatMessage{ReasoningContent: content}
	}
	return &ChatMessage{Content: content}
}
