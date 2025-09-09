package browser

import (
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatYuanbaoHandler{})
}

type chatYuanbaoHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator
}

func (h *chatYuanbaoHandler) Name() string {
	return "yuanbao"
}

func (h *chatYuanbaoHandler) URL() string {
	return "https://yuanbao.tencent.com"
}

func (h *chatYuanbaoHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatYuanbaoHandler) Input(prompt string) (err error) {
	h.log.Debug().Msg("wait for chat main")
	h.locChat = h.page.Locator("div.yb-input-box-textarea")
	if err = h.locChat.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	h.log.Debug().Msg("wait for chat editor")
	locEditor := h.locChat.Locator("div.ql-editor")
	if err = locEditor.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat editor error")
		return err
	}

	h.log.Debug().Msg("fill prompt to chat editor")
	if err = locEditor.Fill(prompt); err != nil {
		logger.Error().Err(err).Msg("fill prompt to chat editor error")
		return err
	}

	return nil
}

func (h *chatYuanbaoHandler) Send() error {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.locChat.Locator("a#yuanbao-send-btn")
	if err := locSend.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat send button error")
		return err
	}

	h.log.Debug().Msg("click chat send button")
	if err := locSend.Click(); err != nil {
		h.log.Error().Err(err).Msg("click chat send button error")
		return err
	}

	return nil
}

type yuanbaoEvent struct {
	// think/text
	Type    string `json:"type"`
	Msg     string `json:"msg,omitempty"`
	Content string `json:"content,omitempty"`
}

// {"type":"think","title":"思考中...","iconType":9,"content":"雨","status":1}
// {"type":"text","msg":"日"}

func (h *chatYuanbaoHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimPrefix(s, "data:")
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*yuanbaoEvent](s)
	if err != nil {
		return nil
	}
	switch event.Type {
	case "think":
		return &ChatMessage{ReasoningContent: event.Content}
	case "text":
		return &ChatMessage{Content: event.Msg}
	}
	return nil
}
