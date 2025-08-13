package browser

import (
	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatGoogleHandler{})
}

type chatGoogleHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator
}

func (h *chatGoogleHandler) Name() string {
	return "google"
}

func (h *chatGoogleHandler) URL() string {
	return "https://aistudio.google.com/prompts/new_chat"
}

func (h *chatGoogleHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatGoogleHandler) Input(prompt string) (err error) {
	h.log.Debug().Msg("wait for new chat link")
	locNew := h.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: "Chat"})
	if err = locNew.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for new chat link error")
		return err
	}
	h.log.Debug().Msg("click new chat link")
	if err = locNew.Click(); err != nil {
		h.log.Error().Err(err).Msg("click new chat link error")
		return err
	}

	h.log.Debug().Msg("wait for chat main")
	h.locChat = h.page.Locator("ms-prompt-input-wrapper")
	if err = h.locChat.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	h.log.Debug().Msg("wait for chat textarea")
	locText := h.locChat.Locator("ms-autosize-textarea textarea")
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

func (h *chatGoogleHandler) Send() error {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.locChat.Locator("run-button button")
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

func (h *chatGoogleHandler) Unmarshal(s string) *ChatMessage {
	node, err := json.Get(s, 0, 0, 0, 0, 0)
	if err != nil {
		return nil
	}
	if node.TypeSafe() != json.TypeArray {
		h.log.Error().Msgf("unmarshal google node expected array, got %d", node.TypeSafe())
		return nil
	}
	arr, err := node.Array()
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal google node array error")
		return nil
	}
	if len(arr) < 2 {
		h.log.Error().Msg("unmarshal google node array length error")
		return nil
	}
	content, ok := arr[1].(string)
	if !ok {
		h.log.Error().Msg("unmarshal google node array[1] error")
		return nil
	}
	if len(arr) >= 13 {
		return &ChatMessage{ReasoningContent: content}
	}
	return &ChatMessage{Content: content}
}
