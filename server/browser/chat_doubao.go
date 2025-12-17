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
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator
}

func (h *chatDoubaoHandler) Name() string {
	return "doubao"
}

func (h *chatDoubaoHandler) URL() string {
	return "https://www.doubao.com/chat"
}

func (h *chatDoubaoHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatDoubaoHandler) Input(prompt string) (err error) {
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

type doubaoEvent struct {
	EventId   string `json:"event_id"`
	EventType int    `json:"event_type"`
	EventData string `json:"event_data"`
}

type doubaoEventData struct {
	Blocks []doubaoBlock `json:"blocks"`
}

type doubaoBlock struct {
	Id          string `json:"id"`
	Pid         string `json:"pid"`
	ContentType int    `json:"content_type"`
	Content     string `json:"content"`
	Reset       bool   `json:"reset"`
}

type doubaoContent struct {
	Text string `json:"text"`
}

func (h *chatDoubaoHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimPrefix(s, "data:")
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*doubaoEvent](s)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal doubao event error")
		return nil
	}
	if event.EventType != 2022 {
		return nil
	}
	data, err := json.UnmarshalTo[*doubaoEventData](event.EventData)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal doubao event data error")
		return nil
	}
	if len(data.Blocks) == 0 {
		return nil
	}
	block := data.Blocks[0]
	if block.ContentType != 10000 {
		return nil
	}
	content, err := json.UnmarshalTo[*doubaoContent](block.Content)
	if err != nil {
		h.log.Error().Err(err).Msg("unmarshal doubao event content error")
		return nil
	}
	if block.Pid != "" {
		return &ChatMessage{Index: event.EventId, ReasoningContent: content.Text + "\n\n"}
	}
	return &ChatMessage{Index: event.EventId, Content: content.Text + "\n\n"}
}
