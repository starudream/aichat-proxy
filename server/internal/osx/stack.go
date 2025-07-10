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

func Stack() string {
	stack := conv.BytesToString(debug.Stack())
	lines := strings.Split(stack, "\n")
	if len(lines) <= 5 {
		return stack
	}
	return strings.Join(append(lines[:1], lines[5:]...), "\n")
}
