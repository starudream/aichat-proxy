package browser

import (
	"testing"
)

func TestChatKimiHandler_Unmarshal(t *testing.T) {
	h := &chatKimiHandler{}
	for i, s := range []string{
		`{"op":"append","mask":"block.think.content","eventOffset":5,"block":{"id":"0_0","think":{"content":"Alright"}}}`,
		`{"op":"append","mask":"block.text.content","eventOffset":372,"block":{"id":"0_1","text":{"content":"How"}}}`,
	} {
		msg := h.Unmarshal(s)
		t.Logf("%d: %#v", i, msg)
	}
}
