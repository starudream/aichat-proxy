package osx_test

import (
	"testing"

	"github.com/starudream/aichat-proxy/server/internal/osx"
)

func TestFuncName(t *testing.T) {
	t.Logf("%s", osx.FuncName(osx.FuncName))
}

func TestStack(t *testing.T) {
	t.Logf("\n%s", osx.Stack())
}
