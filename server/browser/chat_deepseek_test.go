package browser

import (
	"testing"
)

func TestChatDeepseekHandler_Unmarshal(t *testing.T) {
	h := &chatDeepseekHandler{}
	for i, s := range []string{
		`{"v": [{"id": 1, "type": "THINK", "content": "唔", "elapsed_secs": null}], "p": "response/fragments", "o": "APPEND"}`,
		`{"v": "，", "p": "response/fragments/0/content"}`,
		`{"v": "用户"}`,
		`{"v": 3.6751182079315186, "p": "response/fragments/0/elapsed_secs", "o": "SET"}`,
		`{"v": [{"id": 2, "type": "RESPONSE", "content": "你好"}], "p": "response/fragments", "o": "APPEND"}`,
		`{"v": "！", "p": "response/fragments/1/content"}`,
		`{"v": [{"v": "FINISHED", "p": "status"}, {"v": 57, "p": "accumulated_token_usage"}], "p": "response", "o": "BATCH"}`,
	} {
		msg := h.Unmarshal(s)
		t.Logf("%d: %#v", i, msg)
	}
}
