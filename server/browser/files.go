package browser

import (
	_ "embed"
)

var (
	//go:embed file_tampermonkey_sse.js
	FileTamperMonkeySSE []byte
)
