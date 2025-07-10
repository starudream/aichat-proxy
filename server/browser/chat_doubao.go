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
	Type int    `json:"type,omitempty"`
	Text string `json:"text"`
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
	switch data.Message.ContentType {
	case 2001, 10000:
		if content.Type == 0 && content.Text != "" {
			return &ChatMessage{Index: event.EventId, Content: content.Text}
		}
	case 10040:
		tag := "1"
		if data.Message.IsFinish {
			tag = "2"
		}
		return &ChatMessage{Index: event.EventId, ReasoningTag: tag}
	}
	return nil
}
