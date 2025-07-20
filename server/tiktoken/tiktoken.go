package tiktoken

func Tokens(text string) []int {
	return encoding.EncodeOrdinary(text)
}

func NumTokens(text string) int {
	return len(encoding.EncodeOrdinary(text))
}
