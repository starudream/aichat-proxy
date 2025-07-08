package browser

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/internal/writer"
	"github.com/starudream/aichat-proxy/server/logger"
)

var b *Browser

func B() *Browser {
	return b
}

type Browser struct {
	ctx    context.Context
	cancel context.CancelFunc

	cp *CamoufoxParams
	co *CamoufoxOptions

	pw *playwright.Playwright
	bc playwright.BrowserContext
	ps playwright.PlaywrightAssertions

	ec atomic.Uint32
	mu sync.Mutex

	doubaoMu sync.Mutex
}

func Run(ctx context.Context) {
	var err error

	b = &Browser{}
	b.ctx, b.cancel = context.WithCancel(ctx)

	b.cp = &CamoufoxParams{}
	b.co, err = GetCamoufoxOptions(ctx, b.cp)
	if err != nil {
		logger.Fatal().Err(err).Msg("camoufox get launch options error")
	}

	b.pw, err = playwright.Run(&playwright.RunOptions{
		SkipInstallBrowsers: true,
		Verbose:             true,
		Stdout:              writer.NewPrefixWriter("playwright"),
		Stderr:              writer.NewPrefixWriter("playwright"),
		Logger:              slog.Default(),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("playwright run error")
	}

	b.bc, err = b.pw.Firefox.LaunchPersistentContext(config.Userdata0Path, playwright.BrowserTypeLaunchPersistentContextOptions{
		ExecutablePath:    playwright.String(b.co.ExecutablePath),
		Headless:          playwright.Bool(b.co.Headless),
		Args:              b.co.Args,
		Env:               b.co.Env,
		Proxy:             b.co.PWProxy(),
		FirefoxUserPrefs:  b.co.FirefoxUserPrefs,
		BypassCSP:         playwright.Bool(true),
		IgnoreHttpsErrors: playwright.Bool(true),
		AcceptDownloads:   playwright.Bool(true),
		DownloadsPath:     playwright.String(config.DownloadsPath),
		Timeout:           playwright.Float(60 * 1000),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("playwright launch persistent context error")
	}

	b.ps = playwright.NewPlaywrightAssertions(20 * 1000)

	b.bc.OnClose(func(bc playwright.BrowserContext) {
		b.cancel()
		logger.Warn().Msg("browser closed")
	})

	logger.Info().Msg("browser ready")
}

func (s *Browser) openPage(url string) (page playwright.Page, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pages := s.bc.Pages()
	for i := range pages {
		if pages[i].URL() == "about:blank" {
			page = pages[i]
			break
		}
		if strings.HasPrefix(pages[i].URL(), url) {
			return pages[i], nil
		}
	}

	if page == nil {
		page, err = s.bc.NewPage()
		if err != nil {
			logger.Error().Err(err).Msg("open new page error")
			return nil, err
		}
	}

	_, err = page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30 * 1000),
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		logger.Error().Err(err).Msg("page goto error")
		return nil, err
	}

	time.Sleep(time.Second)

	logger.Info().Msgf("page goto %q ready", url)

	return page, nil
}

func (s *Browser) resetPages(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pages := s.bc.Pages()
	for i := range pages {
		if strings.HasPrefix(pages[i].URL(), url) {
			_, _ = pages[i].Goto("about:blank")
		}
	}
}

type DoubaoMessage struct {
	Id     string `json:"id"`
	Text   string `json:"text"`
	Finish string `json:"finish"`
}

type DoubaoHandler struct {
	Id string
	Ch chan *DoubaoMessage

	unix atomic.Int64
	done atomic.Bool
}

const doubaoURL = "https://www.doubao.com/chat/"

func (s *Browser) HandleDoubao(prompt string) (hdr *DoubaoHandler, err error) {
	defer func() {
		if err != nil {
			if s.ec.Add(1) >= 3 {
				s.ec.Store(0)
				s.resetPages(doubaoURL)
			}
		}
	}()

	page, err := s.openPage(doubaoURL)
	if err != nil {
		return hdr, err
	}

	page.SetDefaultTimeout(5 * 1000)

	hdr = &DoubaoHandler{
		Id: uuid.Must(uuid.NewV7()).String(),
		Ch: make(chan *DoubaoMessage),
	}

	log := logger.With().Str("chat", "doubao").Str("id", hdr.Id).Logger()

	finish := func() {
		if hdr.done.CompareAndSwap(false, true) {
			log.Debug().Msg("finish")
			time.Sleep(200 * time.Millisecond)
			page.RemoveListeners("console")
			// err = page.Close()
			// if err != nil {
			// 	log.Error().Err(err).Msg("page close error")
			// } else {
			// 	log.Debug().Msg("page closed")
			// }
			close(hdr.Ch)
			s.doubaoMu.Unlock()
			log.Debug().Msg("release lock")
		}
	}

	log.Debug().Msg("acquire lock")
	s.doubaoMu.Lock()
	defer func() {
		if err != nil {
			finish()
		}
	}()

	log.Debug().Msg("wait for create conversation button")
	locCreate := page.GetByTestId("create_conversation_button")
	if err = locCreate.WaitFor(); err != nil {
		locCreate = page.Locator(`button[class*="create-chat-"]`)
		if err = locCreate.WaitFor(); err != nil {
			log.Error().Err(err).Msg("wait for create conversation button error")
			return hdr, err
		}
	}

	log.Debug().Msg("click create conversation button")
	if err = locCreate.Click(); err != nil {
		logger.Error().Err(err).Msg("click create conversation button error")
		return hdr, err
	}

	log.Debug().Msg("wait for chat main")
	locChat := page.GetByTestId("chat_input")
	if err = locChat.WaitFor(); err != nil {
		log.Error().Err(err).Msg("wait for chat main error")
		return hdr, err
	}

	log.Debug().Msg("wait for chat textarea")
	locText := locChat.Locator("textarea")
	if err = locText.WaitFor(); err != nil {
		log.Error().Err(err).Msg("wait for chat textarea error")
		return hdr, err
	}

	log.Debug().Msg("fill prompt to chat textarea")
	if err = locText.Fill(prompt); err != nil {
		log.Error().Err(err).Msg("fill prompt to chat textarea error")
		return hdr, err
	}

	log.Debug().Msg("wait for chat send button")
	locSend := locChat.GetByTestId("chat_input_send_button")
	if err = locSend.WaitFor(); err != nil {
		log.Error().Err(err).Msg("wait for chat send button error")
		return hdr, err
	}

	log.Info().Msg("listen on console sse")
	hdr.unix.Store(time.Now().Unix())
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if hdr.done.Load() {
			return
		}
		if msg.Type() != "info" {
			return
		}
		ss := reDoubaoEvent.FindStringSubmatch(msg.Text())
		if len(ss) != 3 {
			return
		}
		hdr.unix.Store(time.Now().Unix())
		switch ss[1] {
		case "aichat-proxy-sse-new":
			log.Info().Msg("receive new sse")
		case "aichat-proxy-sse-closed":
			log.Warn().Msg("sse closed")
			hdr.Ch <- &DoubaoMessage{Finish: "stop"}
			finish()
		case "aichat-proxy-sse-error":
			log.Error().Msgf("sse error: %s", ss[2])
		case "aichat-proxy-sse-data":
			event, _err := json.UnmarshalTo[*doubaoEvent](ss[2])
			if _err != nil {
				log.Error().Err(_err).Msgf("sse unmarshal error: %s", ss[2])
				return
			}
			// log.Debug().Msgf("see data: id=%s, type=%d, data=%s", event.EventId, event.EventType, event.EventData)
			data, _err := json.UnmarshalTo[*doubaoEventData](event.EventData)
			if _err != nil {
				log.Error().Err(_err).Msgf("sse data unmarshal error: %s", event.EventData)
				return
			}
			if data.Message == nil {
				return
			}
			content, _err := json.UnmarshalTo[*doubaoEventMessageContent](data.Message.Content)
			if _err != nil {
				log.Error().Err(_err).Msgf("sse data content unmarshal error: %s", data.Message.Content)
				return
			}
			if content.Text != "" {
				hdr.Ch <- &DoubaoMessage{Id: data.Message.Id, Text: content.Text}
			}
		}
	})

	log.Debug().Msg("click chat send button")
	if err = locSend.Click(); err != nil {
		log.Error().Err(err).Msg("click chat send button error")
		return hdr, err
	}

	go func() {
		for {
			<-time.After(time.Second)
			if hdr.done.Load() {
				return
			}
			if t := hdr.unix.Load(); time.Now().Unix()-t >= 30 {
				finish()
				return
			}
		}
	}()

	return hdr, nil
}

type doubaoEvent struct {
	EventId   string `json:"event_id"`
	EventType int    `json:"event_type"`
	EventData string `json:"event_data"`
}

type doubaoEventData struct {
	Message *doubaoEventMessage `json:"message"`
}

type doubaoEventMessage struct {
	Id          string `json:"id"`
	ContentType int    `json:"content_type"`
	Content     string `json:"content"`
}

type doubaoEventMessageContent struct {
	Text string `json:"text"`
}

var reDoubaoEvent = regexp.MustCompile(`^\[(aichat-proxy-[a-z0-9\-]+)]\s?(.*)$`)
