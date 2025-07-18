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
	Setup(options HandleChatOptions)
	Input(prompt string) error
	Send() error
	Unmarshal(s string) *ChatMessage
}

var chatHandlers = map[string]chatHandler{}

func registerChatHandler(h chatHandler) {
	if _, ok := chatHandlers[h.Name()]; ok {
		logger.Fatal().Msgf("chat handler %s already exists", h.Name())
	}
	logger.Info().Msgf("register chat handler %s", h.Name())
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
	for {
		next := false
		select {
		case <-ctx.Done():
		case msg, ok := <-h.Ch:
			if !ok {
				break
			}
			next = true
			if msg.Content != "" {
				content.WriteString(msg.Content)
			} else if msg.ReasoningContent != "" {
				reason.WriteString(msg.ReasoningContent)
			}
		}
		if !next {
			break
		}
	}
	return content.String(), reason.String()
}

type ChatMessage struct {
	Index            string `json:"index,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
	FinishReason     string `json:"finish_reason,omitempty"`
}

type HandleChatOptions struct {
	log  logger.ZLogger
	page playwright.Page

	Thinking  string
	WebSearch string
}

func (s *Browser) HandleChat(ctx context.Context, model, prompt string, options HandleChatOptions) (hdr *ChatHandler, err error) {
	ch, ok := chatHandlers[model]
	if !ok {
		return hdr, fmt.Errorf("model not found: %s", model)
	}

	defer func() {
		if err != nil {
			s.resetPages(ch.URL())
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
			_, _ = page.Evaluate(`window.__aichat_proxy_active_time=Date.now();window.__aichat_proxy_idle_timer=setInterval(()=>{const t=window.__aichat_proxy_active_time;if(t&&Date.now()-t>3e5){window.location.href="about:blank"}},1e4);`)
		}
	}

	_, _ = page.Evaluate(`if(window.__aichat_proxy_idle_timer){clearInterval(window.__aichat_proxy_idle_timer)}`)

	go func() {
		<-ctx.Done()
		finish()
	}()

	log.Debug().Msg("acquire lock")
	s.mu.Lock()
	defer func() {
		if err != nil {
			finish()
		}
	}()

	options.log = log
	options.page = page
	ch.Setup(options)

	if err = ch.Input(prompt); err != nil {
		return hdr, err
	}

	unix := atomic.Int64{}
	unix.Store(time.Now().Unix())

	go func() {
		flag := false
		for {
			time.Sleep(10 * time.Millisecond)
			if done.Load() {
				return
			}
			v := <-proxyChs[ch.Name()]
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
		return hdr, err
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
