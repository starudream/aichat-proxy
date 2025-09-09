package browser

import (
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatZhiPuHandler{})
}

type chatZhiPuHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	blocks []string
}

func (h *chatZhiPuHandler) Name() string {
	return "zhipu"
}

func (h *chatZhiPuHandler) URL() string {
	return "https://chat.z.ai"
}

func (h *chatZhiPuHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatZhiPuHandler) Input(prompt string) (err error) {
	h.log.Debug().Msg("wait for new chat button")
	locNew := h.page.Locator("button#new-chat-button")
	if err = locNew.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for new chat button error")
		return err
	}
	h.log.Debug().Msg("click new chat button")
	if err = locNew.Click(); err != nil {
		h.log.Error().Err(err).Msg("click new chat button error")
		return err
	}

	if h.options.WebSearch != "disabled" {
		h.log.Debug().Msg("wait for tool button")
		locTool := h.page.Locator("button", playwright.PageLocatorOptions{HasText: "工具"}).First()
		if err = locTool.WaitFor(); err != nil {
			h.log.Error().Err(err).Msg("wait for tool button error")
		} else {
			h.log.Debug().Msg("click tool button")
			if err = locTool.Click(); err != nil {
				h.log.Error().Err(err).Msg("click tool button error")
			}
			h.log.Debug().Msgf("wait for web search button")
			locSearch := h.page.Locator("button", playwright.PageLocatorOptions{HasText: "全网搜索"})
			if err = locSearch.WaitFor(); err != nil {
				h.log.Error().Err(err).Msg("wait for web search button error")
			} else {
				h.log.Debug().Msg("click web search button")
				if err = locSearch.Click(); err != nil {
					h.log.Error().Err(err).Msg("click web search button error")
				}
			}
			if err = locTool.Click(); err != nil {
				h.log.Error().Err(err).Msg("click tool button error")
			}
		}
	}

	h.log.Debug().Msg("wait for chat textarea")
	locText := h.page.Locator("textarea#chat-input")
	if err = locText.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat textarea error")
		return err
	}

	h.log.Debug().Msg("fill prompt to chat textarea")
	if err = locText.Focus(); err != nil {
		h.log.Error().Err(err).Msg("focus chat textarea error")
		return err
	}
	if err = locText.Fill(prompt); err != nil {
		logger.Error().Err(err).Msg("fill prompt to chat textarea error")
		return err
	}

	return nil
}

func (h *chatZhiPuHandler) Send() error {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.page.Locator("button#send-message-button")
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

type zhipuEvent struct {
	// chat:completion
	Type string `json:"type"`
	Data struct {
		// thinking/answer
		Phase        string `json:"phase"`
		DeltaContent string `json:"delta_content"`
		// EditIndex    int    `json:"edit_index"`
		EditContent string `json:"edit_content"`
	} `json:"data,omitempty"`
}

func (h *chatZhiPuHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*zhipuEvent](s)
	if err != nil {
		return nil
	}
	content := event.Data.DeltaContent
	if content != "" {
		h.blocks = append(h.blocks, content)
	} else if event.Data.EditContent != "" {
		t := strings.Join(h.blocks[len(h.blocks)-3:], "")
		i := strings.LastIndex(event.Data.EditContent, t)
		if i >= 0 {
			content = event.Data.EditContent[i+len(t):]
		}
	}
	if content == "" {
		return nil
	}
	switch event.Data.Phase {
	case "thinking":
		return &ChatMessage{ReasoningContent: content}
	case "answer":
		return &ChatMessage{Content: content}
	}
	return nil
}
