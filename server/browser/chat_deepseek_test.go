package browser

import (
	"testing"
)

func TestChatDeepseekHandler_Unmarshal(t *testing.T) {
	h := &chatDeepseekHandler{}
	for i, s := range []string{
		`{"v":{"response":{"message_id":2,"parent_id":1,"model":"","role":"ASSISTANT","thinking_enabled":true,"ban_edit":false,"ban_regenerate":false,"status":"WIP","incomplete_message":null,"accumulated_token_usage":0,"files":[],"feedback":null,"inserted_at":1775897050.974429,"search_enabled":true,"fragments":[{"id":2,"type":"THINK","content":"我们","elapsed_secs":null,"references":[],"stage_id":1}],"has_pending_fragment":false,"auto_continue":false}}}`,
		`{"p":"response/fragments/-1/content","o":"APPEND","v":"收到"}`,
		`{"v": "用户"}`,
		`{"v":"。"}`,
		`{"p":"response/fragments/-1/elapsed_secs","o":"SET","v":1.38384612}`,
		`{"p":"response/fragments","o":"APPEND","v":[{"id":3,"type":"RESPONSE","content":"你好","references":[],"stage_id":1}]}`,
		`{"p":"response/fragments/-1/content","v":"！"}`,
		`{"v":"？"}`,
		`{"p":"response","o":"BATCH","v":[{"p":"accumulated_token_usage","v":54},{"p":"quasi_status","v":"FINISHED"}]}`,
	} {
		msg := h.Unmarshal(s)
		t.Logf("%d: %#v", i, msg)
	}
}
