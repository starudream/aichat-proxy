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
	registerChatHandler(&chatQwenHandler{})
}

type chatQwenHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator
}

func (h *chatQwenHandler) Name() string {
	return "qwen"
}

func (h *chatQwenHandler) URL() string {
	return "https://chat.qwen.ai"
}

func (h *chatQwenHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatQwenHandler) Input(prompt string) (err error) {
	closed := atomic.Bool{}

	go func() {
		h.log.Debug().Msg("wait for close sidebar button")
		locSide := h.page.Locator("button.slide-switch")
		if err = locSide.WaitFor(); err == nil {
			h.log.Debug().Msg("click close sidebar button")
			if err = locSide.Click(); err == nil {
				closed.Store(true)
			}
		}
	}()

	go func() {
		h.log.Debug().Msg("wait for closed sidebar button")
		locSide := h.page.Locator("button#sidebar-toggle-button")
		if err = locSide.WaitFor(); err == nil {
			closed.Store(true)
		}
	}()

	timeout := time.After(10 * time.Second)
loop:
	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait for close sidebar button timeout")
		default:
			time.Sleep(100 * time.Millisecond)
			if closed.Load() {
				if err != nil {
					return err
				}
				break loop
			}
		}
	}

	h.log.Debug().Msg("wait for create chat button")
	locCreate := h.page.Locator("button#new-chat-button")
	if err = locCreate.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for create chat button error")
		return err
	}
	h.log.Debug().Msg("click create chat button")
	if err = locCreate.Click(); err != nil {
		h.log.Error().Err(err).Msg("click create chat button error")
		return err
	}

	h.log.Debug().Msg("wait for chat main")
	h.locChat = h.page.Locator("div#chat-message-input")
	if err = h.locChat.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	if h.options.Thinking != "disabled" {
		h.log.Debug().Msg("wait for deep think button")
		locThink := h.locChat.Locator("button.common-btn-padding")
		if err = locThink.WaitFor(); err != nil {
			h.log.Error().Err(err).Msg("wait for deep think button error")
		} else {
			h.log.Debug().Msg("click deep think button")
			if err = locThink.Click(); err != nil {
				h.log.Error().Err(err).Msg("click deep think button error")
			}
		}
	}

	if h.options.WebSearch != "disabled" {
		h.log.Debug().Msg("wait for web search button")
		locSearch := h.locChat.Locator("button.websearch_button")
		if err = locSearch.WaitFor(); err != nil {
			h.log.Error().Err(err).Msg("wait for web search button error")
		} else {
			h.log.Debug().Msg("click web search button")
			if err = locSearch.Click(); err != nil {
				h.log.Error().Err(err).Msg("click web search button error")
			}
		}
	}

	h.log.Debug().Msg("wait for chat textarea")
	locText := h.locChat.Locator("textarea#chat-input")
	if err = locText.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat textarea error")
		return err
	}

	h.log.Debug().Msg("fill prompt to chat textarea")
	if err = locText.Fill(prompt); err != nil {
		h.log.Error().Err(err).Msg("fill prompt to chat textarea error")
		return err
	}

	return nil
}

func (h *chatQwenHandler) Send() (err error) {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.locChat.Locator("button#send-message-button")
	if err = locSend.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat send button error")
		return err
	}

	h.log.Debug().Msg("click chat send button")
	if err = locSend.Click(); err != nil {
		h.log.Error().Err(err).Msg("click chat send button error")
		return err
	}

	return nil
}

type qwenEvent struct {
	Choices []qwenEventChoice `json:"choices"`
}

type qwenEventChoice struct {
	Delta struct {
		// assistant/function
		Role string `json:"role"`
		// think/answer
		Phase   string `json:"phase"`
		Content string `json:"content"`
		// typing/finished
		Status string `json:"status"`
	} `json:"delta"`
}

func (h *chatQwenHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*qwenEvent](s)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal qwen event error")
		return nil
	}
	if len(event.Choices) == 0 {
		return nil
	}
	delta := event.Choices[0].Delta
	switch delta.Phase {
	case "think":
		return &ChatMessage{ReasoningContent: delta.Content}
	case "answer":
		return &ChatMessage{Content: delta.Content}
	}
	return nil
}
