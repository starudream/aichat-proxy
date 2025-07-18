package browser

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatKimiHandler{})
}

type chatKimiHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator
}

func (h *chatKimiHandler) Name() string {
	return "kimi"
}

func (h *chatKimiHandler) URL() string {
	return "https://www.kimi.com"
}

func (h *chatKimiHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatKimiHandler) Input(prompt string) (err error) {
	opened := atomic.Bool{}

	go func() {
		h.log.Debug().Msg("wait for open sidebar button")
		locSide := h.page.Locator("div.layout-header div.expand-btn")
		if err = locSide.WaitFor(); err == nil {
			h.log.Debug().Msg("click open sidebar button")
			if err = locSide.Click(); err == nil {
				opened.Store(true)
			}
		}
	}()

	go func() {
		h.log.Debug().Msg("wait for opened sidebar button")
		locSide := h.page.Locator("div.sidebar-header div.expand-btn")
		if err = locSide.WaitFor(); err == nil {
			opened.Store(true)
		}
	}()

	timeout := time.After(10 * time.Second)
loop:
	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait for open sidebar button timeout")
		default:
			time.Sleep(100 * time.Millisecond)
			if opened.Load() {
				if err != nil {
					return err
				}
				break loop
			}
		}
	}

	h.log.Debug().Msg("wait for new chat button")
	locNew := h.page.Locator("a.new-chat-btn")
	if err = locNew.WaitFor(); err != nil {
		logger.Error().Err(err).Msg("wait for new chat button error")
		return err
	}
	h.log.Debug().Msg("click new chat button")
	if err = locNew.Click(); err != nil {
		logger.Error().Err(err).Msg("click new chat button error")
		return err
	}

	h.log.Debug().Msg("wait for chat main")
	h.locChat = h.page.Locator("div.chat-editor")
	if err = h.locChat.WaitFor(); err != nil {
		logger.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	h.log.Debug().Msg("wait for chat editor")
	locText := h.locChat.GetByRole("textbox")
	if err = locText.WaitFor(); err != nil {
		logger.Error().Err(err).Msg("wait for chat editor error")
		return err
	}

	h.log.Debug().Msg("fill prompt to chat editor")
	if err = locText.Fill(prompt); err != nil {
		logger.Error().Err(err).Msg("fill prompt to chat editor error")
		return err
	}

	return nil
}

func (h *chatKimiHandler) Send() error {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.locChat.Locator("div.send-button")
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

type kimiEvent struct {
	Event string `json:"event"`
	Text  string `json:"text,omitempty"`
	View  string `json:"view,omitempty"`

	Error struct {
		ErrorType string `json:"error_type,omitempty"`
		Message   string `json:"message,omitempty"`
		Detail    string `json:"detail,omitempty"`
	} `json:"error,omitempty"`
}

func (h *chatKimiHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*kimiEvent](s)
	if err != nil {
		return nil
	}
	switch event.Event {
	case "error":
		return &ChatMessage{Content: event.Error.Message}
	case "cmpl":
		return &ChatMessage{Content: event.Text}
	}
	return nil
}
