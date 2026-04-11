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
	h.log.Debug().Msg("click start new conversation")
	if err = h.page.GetByText("开启新对话").Click(); err != nil {
		h.log.Error().Err(err).Msg("click start new conversation error")
		return err
	}

	h.log.Debug().Msg("wait for chat main")
	locChat := h.page.GetByRole(*playwright.AriaRoleTextbox, playwright.PageGetByRoleOptions{Name: "给 DeepSeek 发送消息"})
	if err = locChat.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	h.log.Debug().Msg("wait for chat textarea")
	if err = locChat.Fill(prompt); err != nil {
		h.log.Error().Err(err).Msg("fill prompt error")
		return err
	}

	return nil
}

func (h *chatDeepseekHandler) Send() (err error) {
	h.log.Debug().Msg("click send")
	if err = h.page.GetByRole(*playwright.AriaRoleButton, playwright.PageGetByRoleOptions{}).Nth(4).Click(); err != nil {
		h.log.Error().Err(err).Msg("click send error")
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

	Response struct {
		Fragments []struct {
			Id      int    `json:"id"`
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"fragments"`
	} `json:"response"`
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
	case map[string]any:
		v, e := json.UnmarshalTo[*deepseekV](json.MustMarshal(x))
		if e != nil || v == nil {
			return nil
		}
		if len(v.Response.Fragments) == 0 {
			return nil
		}
		fragment := v.Response.Fragments[0]
		switch fragment.Type {
		case "THINK":
			h.reasoning.Store(true)
		case "RESPONSE":
			h.reasoning.Store(false)
		default:
			return nil
		}
		content = fragment.Content
	}
	if content == "" {
		return nil
	}
	if h.reasoning.Load() {
		return &ChatMessage{ReasoningContent: content}
	}
	return &ChatMessage{Content: content}
}
