package config

import (
	"strings"

	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/internal/conv"
)

type Array[T cast.Basic] []T

func (vs *Array[T]) UnmarshalText(bs []byte) error {
	ss := strings.FieldsFunc(conv.BytesToString(bs), func(r rune) bool { return r == ',' })
	*vs = make([]T, len(ss))
	for i, s := range ss {
		(*vs)[i] = cast.To[T](s)
	}
	return nil
}
