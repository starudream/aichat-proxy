package browser

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/writer"
	"github.com/starudream/aichat-proxy/server/logger"
)

var b *Browser

func B() *Browser {
	return b
}

type Browser struct {
	cp *CamoufoxParams
	co *CamoufoxOptions

	pw *playwright.Playwright
	bc playwright.BrowserContext

	ec atomic.Uint32
	mu sync.Mutex
}

func startBrowser(ctx context.Context, wg *sync.WaitGroup) {
	var err error

	b = &Browser{}

	b.cp = &CamoufoxParams{}
	b.co, err = GetCamoufoxOptions(ctx, b.cp)
	if err != nil {
		logger.Fatal().Err(err).Msg("camoufox get launch options error")
	}

	logger.Info().Msg("wait for playwright launch browser ready, may take a few seconds")

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

	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Warn().Msg("browser closing")
		_ = b.bc.Close()
		logger.Info().Msg("browser closed")
	}()

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
