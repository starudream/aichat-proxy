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
	registerChatHandler(&chatDoubaoHandler{})
}

type chatDoubaoHandler struct {
	options HandleChatOptions

	log  logger.ZLogger
	page playwright.Page

	locChat playwright.Locator

	reasoning atomic.Bool
}

func (h *chatDoubaoHandler) Name() string {
	return "doubao"
}

func (h *chatDoubaoHandler) URL() string {
	return "https://www.doubao.com/chat/"
}

func (h *chatDoubaoHandler) Setup(options HandleChatOptions) {
	h.log = options.log
	h.page = options.page
	h.options = options
}

func (h *chatDoubaoHandler) Input(prompt string) (err error) {
	closed := atomic.Bool{}

	go func() {
		h.log.Debug().Msg("wait for close sidebar button")
		locSide := h.page.GetByTestId("siderbar_close_btn")
		if err = locSide.WaitFor(); err == nil {
			h.log.Debug().Msg("click close sidebar button")
			if err = locSide.Click(); err == nil {
				closed.Store(true)
			}
		}
	}()

	go func() {
		h.log.Debug().Msg("wait for closed sidebar button")
		locSide := h.page.GetByTestId("siderbar_closed_status_btn")
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

	h.log.Debug().Msg("wait for create conversation button")
	locCreate := h.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "新对话"})
	if err = locCreate.WaitFor(); err != nil {
		h.log.Error().Err(err).Msg("wait for create conversation button error")
		return err
	}
	if disabled, _ := locCreate.IsDisabled(); !disabled {
		h.log.Debug().Msg("click create conversation button")
		if err = locCreate.Click(); err != nil {
			h.log.Error().Err(err).Msg("click create conversation button error")
			return err
		}
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

type doubaoEvent struct {
	EventId   string `json:"event_id"`
	EventType int    `json:"event_type"`
	EventData string `json:"event_data"`
}

// 正文
// {
//  "message": {
//    "content_type": 2001,
//    "content": "{\"text\":\"Hello\"}",
//    "id": ""
//  }
// }
// 建议
// {
//  "message": {
//    "content_type": 2002,
//    "content": "{\"suggest\":\"Can you tell me some interesting facts?\",\"suggestions\":[\"Can you tell me some interesting facts?\"]}",
//    "id": ""
//  }
// }
// 联网搜索
// {
//  "message": {
//    "content_type": 2003,
//    "content": "{\"type\":1,\"text\":\"正在搜索\"}",
//    "id": ""
//  }
// }
// 深度思考开始
// {
//  "message": {
//    "content_type": 10040,
//    "content": "{\"finish_title\":\"深度思考中\"}",
//    "id": ""
//  }
// }
// 深度思考内容
// {
//  "message": {
//    "content_type": 10000,
//    "content": "{\"text\":\"用户说\\\"hello\"}",
//    "id": "",
//    "pid": ""
//  }
// }
// 深度思考结束
// {
//  "message": {
//    "content_type": 10040,
//    "content": "{\"finish_title\":\"已完成思考\"}",
//    "reset": true,
//    "id": "",
//    "is_finish": true
//  }
// }
// 正文内容
// {
//  "message": {
//    "content_type": 10000,
//    "content": "{\"text\":\"Hello! 很高兴能和\"}",
//    "id": ""
//  }
// }
// 正文结束
// {
//  "message": {
//    "content_type": 10000,
//    "content": "{}",
//    "id": ""
//  },
//  "is_finish": true
// }

type doubaoEventData struct {
	Message struct {
		Id string `json:"id"`
		//  2001 正文
		//  2002 建议
		//  2003 搜索提示
		// 10000 正文
		// 10025 参考资料
		// 10040 深度思考开始/结束
		ContentType int    `json:"content_type"`
		Content     string `json:"content"`
		// 思考结束
		IsFinish bool `json:"is_finish"`
	} `json:"message"`
}

type doubaoEventContent struct {
	Type  int    `json:"type,omitempty"`
	Text  string `json:"text,omitempty"`
	Think string `json:"think,omitempty"`
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
	if data.Message.ContentType == 2003 {
		if content.Type == 5 {
			h.reasoning.Store(true)
		} else if content.Type == 6 {
			h.reasoning.Store(false)
		}
		return nil
	} else if data.Message.ContentType == 10040 {
		if data.Message.IsFinish {
			h.reasoning.Store(false)
		} else {
			h.reasoning.Store(true)
		}
		return nil
	}
	text := content.Text
	if content.Think != "" {
		text = content.Think
	}
	if h.reasoning.Load() {
		return &ChatMessage{Index: event.EventId, ReasoningContent: text}
	}
	return &ChatMessage{Index: event.EventId, Content: text}
}
