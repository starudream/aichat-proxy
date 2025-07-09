package browser

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

// CamoufoxParams https://camoufox.com/python/usage/
type CamoufoxParams struct {
	// 调试模式，输出更多信息
	Debug bool `json:"debug,omitempty"`
	// 插件
	Addons []string `json:"addons,omitempty"`
	// 无头模式，
	Headless bool `json:"headless,omitempty"`
	// 虚拟显示器
	VirtualDisplay string `json:"virtual_display,omitempty"`
	// websocket 端口
	Port int `json:"port,omitempty"`
	// websocket 路径
	WsPath string `json:"ws_path,omitempty"`
	// 操作系统
	OS string `json:"os,omitempty"`
	// 使光标移动更人性化
	Humanize float64 `json:"humanize,omitempty"`
	// 是否缓存之前的页面、请求
	EnableCache bool `json:"enable_cache,omitempty"`
	// 区域设置
	Locale string `json:"locale,omitempty"`
	// 禁用 Cross-Origin-Opener-Policy
	DisableCoop bool `json:"disable_coop,omitempty"`
	// 地理信息
	GeoIP bool `json:"geoip,omitempty"`
	// 代理
	Proxy *Proxy `json:"proxy,omitempty"`
}

func (p *CamoufoxParams) init() *CamoufoxParams {
	if p == nil {
		//goland:noinspection GoAssignmentToReceiver
		p = &CamoufoxParams{}
	}
	if config.DEBUG("BROWSER") {
		p.Debug = true
	}
	if len(p.Addons) == 0 {
		p.Addons = []string{config.AddonTamperMonkeyPath}
	}
	if !p.Headless && p.VirtualDisplay == "" {
		p.VirtualDisplay = ":0.0"
	}
	if p.Port <= 0 {
		p.Port = config.CamoufoxPort
	}
	p.WsPath = "/ws"
	if p.OS == "" {
		p.OS = "macos"
	}
	p.Humanize = 0.1
	p.EnableCache = true
	if p.Locale == "" {
		p.Locale = "zh-CN"
	}
	// p.DisableCoop = true
	p.GeoIP = true
	p.Proxy = &Proxy{Server: strings.Join([]string{"http", config.ProxyAddress}, "://")}
	return p
}

type CamoufoxOptions struct {
	ExecutablePath   string            `json:"executable_path"`
	Headless         bool              `json:"headless,omitempty"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	Proxy            *Proxy            `json:"proxy,omitempty"`
	FirefoxUserPrefs map[string]any    `json:"firefox_user_prefs,omitempty"`
}

type Proxy struct {
	Server   string `json:"server,omitempty"`
	Bypass   string `json:"bypass,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (o *CamoufoxOptions) PWProxy() *playwright.Proxy {
	if o.Proxy == nil || o.Proxy.Server == "" {
		return nil
	}
	px := &playwright.Proxy{
		Server: o.Proxy.Server,
	}
	if o.Proxy.Bypass != "" {
		px.Bypass = playwright.String(o.Proxy.Bypass)
	}
	if o.Proxy.Username != "" {
		px.Username = playwright.String(o.Proxy.Username)
	}
	if o.Proxy.Password != "" {
		px.Password = playwright.String(o.Proxy.Password)
	}
	return px
}

const camoufoxPython = `
import json
import camoufox

args = json.loads("""__JSON__""")
args["exclude_addons"] = [camoufox.addons.DefaultAddons.UBO]
opts = json.dumps(camoufox.utils.launch_options(**args))
print("OPTIONS={}".format(opts))
`

func GetCamoufoxOptions(ctx context.Context, params *CamoufoxParams) (*CamoufoxOptions, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	paramsS := json.MustMarshalToString(params.init())
	logger.Info().Msgf("camoufox params: %s", paramsS)
	python := strings.Replace(camoufoxPython, "__JSON__", paramsS, 1)
	buf := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "python", "-c", python)
	cmd.Stdout, cmd.Stderr = buf, buf
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command exec error: %w, output: %s", err, buf.String())
	}
	outputS := strings.TrimSpace(buf.String())
	if idx := strings.Index(outputS, "OPTIONS="); idx == -1 {
		return nil, fmt.Errorf("camoufox output error: %s", outputS)
	} else {
		outputS = outputS[idx+8:]
	}
	if config.DEBUG("BROWSER") {
		logger.Debug().Msgf("camoufox options: %s", outputS)
	}
	options, err := json.UnmarshalTo[*CamoufoxOptions](outputS)
	if err != nil {
		return nil, err
	}
	if config.DEBUG("BROWSER") {
		keys := slices.Collect(maps.Keys(json.MustUnmarshalTo[map[string]any](outputS)))
		slices.Sort(keys)
		logger.Debug().Msgf("camoufox options keys: %s", strings.Join(keys, ","))
	}
	return options, nil
}
