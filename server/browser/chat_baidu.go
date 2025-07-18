package browser

import (
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func init() {
	registerChatHandler(&chatBaiduHandler{})
}

type chatBaiduHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator
}

func (h *chatBaiduHandler) Name() string {
	return "baidu"
}

func (h *chatBaiduHandler) URL() string {
	return "https://yiyan.baidu.com"
}

func (h *chatBaiduHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatBaiduHandler) Input(prompt string) (err error) {
	h.log.Debug().Msg("wait for create chat button")
	locCreate := h.page.Locator(`div[class^="newSession"]`, playwright.PageLocatorOptions{HasText: "新对话"})
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
	h.locChat = h.page.Locator(`div[class^="DialogueInputRow"]`)
	if err = h.locChat.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for chat main error")
		return err
	}

	h.log.Debug().Msg("wait for deep think button")
	locThink := h.locChat.Locator(`div[class^="item"]`, playwright.LocatorLocatorOptions{HasText: "思考"})
	if err = locThink.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for deep think button error")
	} else {
		active := false
		attrs, _ := locThink.GetAttribute("class")
		for _, attr := range strings.Split(attrs, " ") {
			if strings.HasPrefix(attr, "active") {
				active = true
			}
		}
		if !active {
			h.log.Debug().Msg("click deep think button")
			if err = locThink.Click(); err != nil {
				h.log.Error().Err(err).Msg("click deep think button error")
			}
		}
	}

	h.log.Debug().Msg("wait for chat editor")
	locEditor := h.locChat.Locator("div.yc-editor")
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

func (h *chatBaiduHandler) Send() error {
	h.log.Debug().Msg("wait for chat send button")
	locSend := h.locChat.Locator("span#sendBtn")
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

type baiduEvent struct {
	// thought
	ThoughtIndex *int   `json:"thought_index,omitempty"`
	StepId       string `json:"step_id,omitempty"`
	Thoughts     string `json:"thoughts,omitempty"`
	// IsEnd        int    `json:"is_end"`
	// content
	Data struct {
		Content string `json:"content,omitempty"`
		// IsEnd   int    `json:"is_end"`
	} `json:"data,omitempty"`
}

func (h *chatBaiduHandler) Unmarshal(s string) *ChatMessage {
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" {
		return nil
	}
	event, err := json.UnmarshalTo[*baiduEvent](s)
	if err != nil {
		return nil
	}
	if event.ThoughtIndex != nil {
		if *event.ThoughtIndex == 0 && cast.To[int](strings.TrimPrefix(event.StepId, "step-")) > 1 {
			event.Thoughts = "\n\n" + event.Thoughts
		}
		return &ChatMessage{ReasoningContent: event.Thoughts}
	} else if event.Data.Content != "" {
		return &ChatMessage{Content: event.Data.Content}
	}
	return nil
}
