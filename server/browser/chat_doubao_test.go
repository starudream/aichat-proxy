package browser

import (
	"strings"
	"testing"
)

const doubaoRaw = `

`

func TestChatDoubaoHandler_Unmarshal(t *testing.T) {
	h := &chatDoubaoHandler{}
	for i, s := range strings.Split(doubaoRaw, "\n") {
		if s == "" {
			continue
		}
		msg := h.Unmarshal(s)
		if msg == nil {
			continue
		}
		t.Logf("%d: %#v", i, msg)
	}
}
