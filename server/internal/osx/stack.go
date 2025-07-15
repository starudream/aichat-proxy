package osx

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/starudream/aichat-proxy/server/internal/conv"
)

func FuncName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func Stack(skips ...int) string {
	skip := 2
	if len(skips) > 0 {
		skip += skips[0]
	}
	stack := conv.BytesToString(debug.Stack())
	lines := strings.Split(stack, "\n")
	if len(lines) <= 1+2*skip {
		return stack
	}
	return strings.Join(append(lines[:1], lines[1+2*skip:]...), "\n")
}
