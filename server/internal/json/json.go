package json

import (
	stdjson "encoding/json"
	"fmt"
	"io"

	"github.com/bytedance/sonic"

	"github.com/starudream/aichat-proxy/server/internal/conv"
)

var json = sonic.Config{
	EscapeHTML:       false,
	SortMapKeys:      false,
	CompactMarshaler: true,
	CopyString:       true,
	ValidateString:   true,
}.Froze()

var (
	Marshal         = json.Marshal
	MarshalToString = json.MarshalToString

	Unmarshal           = json.Unmarshal
	UnmarshalFromString = json.UnmarshalFromString

	NewEncoder = json.NewEncoder
	NewDecoder = json.NewDecoder

	Compact = stdjson.Compact
)

func MustMarshal(v any) []byte {
	bs, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return bs
}

func MustMarshalToString(v any) string {
	bs, err := MarshalToString(v)
	if err != nil {
		panic(err)
	}
	return bs
}

func UnmarshalTo[T any](v any) (t T, err error) {
	switch x := v.(type) {
	case string:
		err = Unmarshal(conv.StringToBytes(x), &t)
	case []byte:
		err = Unmarshal(x, &t)
	case io.Reader:
		err = json.NewDecoder(x).Decode(&t)
	default:
		panic(fmt.Errorf("json.UnmarshalTo: invalid type: %T", v))
	}
	return
}

func MustUnmarshalTo[T any](v any) T {
	t, err := UnmarshalTo[T](v)
	if err != nil {
		panic(err)
	}
	return t
}
