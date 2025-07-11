package browser

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/logger"
)

type chatHandler interface {
	Name() string
	URL() string
	Setup(page playwright.Page, log logger.ZLogger)
	Input(prompt string) error
	Send() error
	Unmarshal(s string) *ChatMessage
}

var chatHandlers = map[string]chatHandler{}

func registerChatHandler(h chatHandler) {
	chatHandlers[h.Name()] = h
}

func ExistModel(model string) bool {
	_, ok := chatHandlers[model]
	return ok
}

func Models() (ss []string) {
	for k := range chatHandlers {
		ss = append(ss, k)
	}
	slices.Sort(ss)
	return
}

type ChatHandler struct {
	Id string
	Ch chan *ChatMessage
}

func (h *ChatHandler) WaitFinish(ctx context.Context) (string, string) {
	content, reason := &bytes.Buffer{}, &bytes.Buffer{}
	tag := ""
	for {
		next := false
		select {
		case <-ctx.Done():
			break
		case msg, ok := <-h.Ch:
			if !ok {
				break
			}
			next = true
			if msg.ReasoningTag != "" {
				tag = msg.ReasoningTag
				continue
			}
			if tag == "1" {
				reason.WriteString(msg.Content)
			} else {
				content.WriteString(msg.Content)
			}
		}
		if !next {
			break
		}
	}
	return content.String(), reason.String()
}

type ChatMessage struct {
	Index        string `json:"index,omitempty"`
	Content      string `json:"content,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`

	// 1开始 2结束
	ReasoningTag string `json:"-"`
}

func (s *Browser) HandleChat(model, prompt string) (hdr *ChatHandler, err error) {
	ch, ok := chatHandlers[model]
	if !ok {
		return hdr, fmt.Errorf("model not found: %s", model)
	}

	defer func() {
		if err != nil {
			if s.ec.Add(1) >= 3 {
				s.ec.Store(0)
				s.resetPages(ch.URL())
			}
		}
	}()

	page, err := s.openPage(ch.URL())
	if err != nil {
		return hdr, err
	}

	page.SetDefaultTimeout(5 * 1000)

	hdr = &ChatHandler{
		Id: uuid.Must(uuid.NewV7()).String(),
		Ch: make(chan *ChatMessage, 1024),
	}

	log := logger.With().Str("model", model).Str("handlerId", hdr.Id).Logger()

	done := atomic.Bool{}
	finish := func() {
		if done.CompareAndSwap(false, true) {
			log.Debug().Msg("handle finish")
			time.Sleep(200 * time.Millisecond)
			close(hdr.Ch)
			s.mu.Unlock()
			log.Debug().Msg("release lock")
		}
	}

	log.Debug().Msg("acquire lock")
	s.mu.Lock()
	defer func() {
		if err != nil {
			finish()
		}
	}()

	ch.Setup(page, log)

	if err = ch.Input(prompt); err != nil {
		return nil, err
	}

	unix := atomic.Int64{}
	unix.Store(time.Now().Unix())

	go func() {
		flag := false
		for {
			if done.Load() {
				return
			}
			v := <-proxyCh
			unix.Store(time.Now().Unix())
			switch x := v.(type) {
			case bool:
				if x {
					log.Debug().Msg("listen sse start")
					flag = true
				} else if flag {
					hdr.Ch <- &ChatMessage{FinishReason: "stop"}
					log.Debug().Msg("listen sse finish")
					finish()
				}
			case string:
				if flag {
					msg := ch.Unmarshal(x)
					if msg != nil {
						hdr.Ch <- msg
					}
				}
			}
		}
	}()

	if err = ch.Send(); err != nil {
		return nil, err
	}

	go func() {
		for {
			time.Sleep(time.Second)
			if done.Load() {
				return
			}
			if t := unix.Load(); time.Now().Unix()-t >= 30 {
				finish()
				return
			}
		}
	}()

	return hdr, nil
}
