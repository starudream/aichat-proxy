package tiktoken

import (
	"testing"
)

func TestTokens(t *testing.T) {
	text := "tiktoken is a fast BPE tokeniser for use with OpenAI's models.\n\n"
	t.Log(Tokens(text))
	t.Log(NumTokens(text))
}
