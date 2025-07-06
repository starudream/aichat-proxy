package browser

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

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

func (s *Browser) openPage(url string) (playwright.Page, error) {
	pages := s.bc.Pages()
	for i := range pages {
		if strings.HasPrefix(pages[i].URL(), url) {
			return pages[i], nil
		}
	}

	page, err := s.bc.NewPage()
	if err != nil {
		logger.Error().Err(err).Msg("browser new page error")
		return nil, err
	}

	_, err = page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30 * 1000),
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		logger.Error().Err(err).Msg("page goto error")
		return nil, err
	}

	logger.Info().Msgf("page goto %q ready", url)

	return page, nil
}

type DoubaoMessage struct {
	Id     string `json:"id"`
	Text   string `json:"text"`
	Finish string `json:"finish"`
}

type DoubaoHandler struct {
	Id string
	Ch chan *DoubaoMessage
}

func (s *Browser) HandleDoubao(prompt string) (*DoubaoHandler, error) {
	page, err := s.openPage("https://www.doubao.com/chat/")
	if err != nil {
		return nil, err
	}

	page.SetDefaultTimeout(10 * 1000)

	log := logger.With().Str("chat", "doubao").Logger()

	log.Debug().Msg("page wait for chat")
	locChat := page.GetByTestId("chat_input")
	err = locChat.WaitFor()
	if err != nil {
		log.Error().Err(err).Msg("page wait for chat input error")
		return nil, err
	}
	log.Debug().Msg("page find chat")

	// log.Info().Msg("page wait for left tools")
	// locLeft := locChat.Locator(`div[class^="left-tools-wrapper-"]`)
	// err = locLeft.WaitFor(locatorWaitForOptions)
	// if err != nil {
	// 	log.Error().Err(err).Msg("page wait for left tools error")
	// 	return nil, err
	// }
	// log.Info().Msg("page find left tools")
	//
	// locButtons, err := locLeft.Locator("button").All()
	// if err != nil {
	// 	log.Error().Err(err).Msg("page get left tools buttons error")
	// 	return nil, err
	// }
	// log.Info().Msg("page click deep think select button")
	// locThink := locButtons[len(locButtons)-1]
	// err = locThink.Click(playwright.LocatorClickOptions{
	// 	Button:  playwright.MouseButtonLeft,
	// 	Delay:   playwright.Float(10),
	// 	Timeout: playwright.Float(3 * 1000),
	// })
	// if err != nil {
	// 	log.Error().Err(err).Msg("page click deep think select button error")
	// 	return nil, err
	// }

	log.Debug().Msg("page wait for chat textarea")
	locText := locChat.Locator("textarea")
	err = locText.WaitFor()
	if err != nil {
		log.Error().Err(err).Msg("page wait for chat textarea error")
		return nil, err
	}
	log.Debug().Msg("page find chat textarea")

	log.Debug().Msg("page fill chat textarea")
	err = locText.Fill(prompt)
	if err != nil {
		log.Error().Err(err).Msg("page fill chat textarea error")
		return nil, err
	}
	log.Debug().Msg("page fill chat textarea done")

	log.Debug().Msg("page wait for chat send button")
	locSend := locChat.GetByTestId("chat_input_send_button")
	err = locSend.WaitFor()
	if err != nil {
		log.Error().Err(err).Msg("page wait for chat send button error")
		return nil, err
	}
	log.Debug().Msg("page find chat send button")

	ch := make(chan *DoubaoMessage)
	log.Info().Msg("page on console")
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if msg.Type() != "info" {
			return
		}
		ss := reDoubaoEvent.FindStringSubmatch(msg.Text())
		if len(ss) != 3 {
			return
		}
		switch ss[1] {
		case "aichat-proxy-sse-new":
			log.Info().Msg("receive new sse")
		case "aichat-proxy-sse-closed":
			log.Warn().Msg("sse closed")
			ch <- &DoubaoMessage{Finish: "stop"}
			close(ch)
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
				ch <- &DoubaoMessage{Id: data.Message.Id, Text: content.Text}
			}
		}
	})

	log.Debug().Msg("page click chat send button")
	err = locSend.Click(playwright.LocatorClickOptions{
		Button: playwright.MouseButtonLeft,
		Delay:  playwright.Float(10),
	})
	if err != nil {
		log.Error().Err(err).Msg("page click chat send button error")
		return nil, err
	}
	log.Info().Msg("page click chat send button done")

	return &DoubaoHandler{Id: uuid.Must(uuid.NewV7()).String(), Ch: ch}, nil
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
