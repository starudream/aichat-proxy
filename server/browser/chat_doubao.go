package browser

import (
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatDoubaoHandler{})
}

type chatDoubaoHandler struct {
	page playwright.Page
	log  logger.ZLogger

	locChat playwright.Locator
}

func (h *chatDoubaoHandler) Name() string {
	return "doubao"
}

func (h *chatDoubaoHandler) URL() string {
	return "https://www.doubao.com/chat/"
}

func (h *chatDoubaoHandler) Setup(page playwright.Page, log logger.ZLogger) {
	h.page = page
	h.log = log
}

func (h *chatDoubaoHandler) Input(prompt string) (err error) {
	h.log.Debug().Msg("wait for create conversation button")
	locCreate := h.page.GetByTestId("create_conversation_button")
	if err = locCreate.WaitFor(); err != nil {
		locCreate = h.page.Locator(`button[class*="create-chat-"]`)
		if err = locCreate.WaitFor(); err != nil {
			h.log.Error().Err(err).Msg("wait for create conversation button error")
			return err
		}
	}

	h.log.Debug().Msg("click create conversation button")
	if err = locCreate.Click(); err != nil {
		logger.Error().Err(err).Msg("click create conversation button error")
		return err
	}

	h.log.Debug().Msg("wait for chat main")
	h.locChat = h.page.GetByTestId("chat_input")
	if err = h.locChat.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	h.log.Debug().Msg("wait for chat textarea")
	locText := h.locChat.Locator("textarea")
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

func (h *chatDoubaoHandler) Send() (err error) {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.locChat.GetByTestId("chat_input_send_button")
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

func (h *chatDoubaoHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*doubaoEvent](s)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal doubao event error")
		return nil
	}
	data, err := json.UnmarshalTo[*doubaoEventData](event.EventData)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal doubao event data error")
		return nil
	}
	if data.Message.Content == "" {
		return nil
	}
	content, err := json.UnmarshalTo[*doubaoEventContent](data.Message.Content)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal doubao event content error")
		return nil
	}
	if content.Text == "" {
		return nil
	}
	return &ChatMessage{Id: event.EventId, Text: content.Text}
}

type doubaoEvent struct {
	EventId   string `json:"event_id"`
	EventType int    `json:"event_type"`
	EventData string `json:"event_data"`
}

type doubaoEventData struct {
	Message struct {
		Id          string `json:"id"`
		ContentType int    `json:"content_type"`
		Content     string `json:"content"`
	} `json:"message"`
}

type doubaoEventContent struct {
	Text string `json:"text"`
}
