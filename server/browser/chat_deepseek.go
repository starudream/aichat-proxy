package browser

import (
	"strings"
	"sync/atomic"
	"time"

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
	closed := atomic.Bool{}

	go func() {
		h.log.Debug().Msg("wait for close sidebar button")
		locSide := h.page.Locator(`div.ds-icon-button:has(rect[id^="折叠边栏"])`)
		if err = locSide.WaitFor(); err == nil {
			h.log.Debug().Msg("click close sidebar button")
			if err = locSide.Click(); err == nil {
				closed.Store(true)
			}
		}
	}()

	go func() {
		h.log.Debug().Msg("wait for closed sidebar button")
		locSide := h.page.Locator(`div.ds-icon-button:has(rect[id^="打开边栏"])`)
		if err = locSide.WaitFor(); err == nil {
			closed.Store(true)
		}
	}()

	for {
		time.Sleep(100 * time.Millisecond)
		if closed.Load() {
			if err != nil {
				return err
			}
			break
		}
	}

	h.log.Debug().Msg("wait for new chat button")
	locNew := h.page.Locator(`div.ds-icon-button:has(rect[id^="新建会话"])`)
	if err = locNew.WaitFor(); err != nil {
		logger.Error().Err(err).Msg("wait for new chat button error")
		return err
	}
	h.log.Debug().Msg("click new chat button")
	if err = locNew.Click(); err != nil {
		logger.Error().Err(err).Msg("click new chat button error")
		return err
	}

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
	V string `json:"v"`
	// response/thinking_content or response/content
	P string `json:"p,omitempty"`
	O string `json:"o,omitempty"`
}

// {"v": "嗯", "p": "response/thinking_content"}
// {"v": "……", "o": "APPEND"}
// {"v": 8, "p": "response/thinking_elapsed_secs", "o": "SET"}
// {"v": "你好", "p": "response/content", "o": "APPEND"}

func (h *chatDeepseekHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*deepseekEvent](s)
	if err != nil {
		return nil
	}
	if event.V == "" {
		return nil
	}
	switch event.P {
	case "":
		// pass
	case "response/thinking_content":
		h.reasoning.Store(true)
	case "response/content":
		h.reasoning.Store(false)
	default:
		return nil
	}
	if h.reasoning.Load() {
		return &ChatMessage{ReasoningContent: event.V}
	}
	return &ChatMessage{Content: event.V}
}
