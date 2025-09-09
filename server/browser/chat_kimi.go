package browser

import (
	"strings"

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
	Op string `json:"op"`
	// block.think.content or block.text.content
	Mask        string `json:"mask"`
	EventOffset int    `json:"eventOffset"`
	Block       struct {
		Id    string `json:"id"`
		Think struct {
			Content string `json:"content"`
		} `json:"think,omitempty"`
		Text struct {
			Content string `json:"content"`
		} `json:"text,omitempty"`
	} `json:"block"`
}

func (h *chatKimiHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*kimiEvent](s)
	if err != nil {
		return nil
	}
	switch event.Mask {
	case "block.think.content":
		return &ChatMessage{ReasoningContent: event.Block.Think.Content}
	case "block.text.content":
		return &ChatMessage{Content: event.Block.Text.Content}
	}
	return nil
}
