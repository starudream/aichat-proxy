package errx

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

type Error struct {
	Status   int            `json:"status"`
	Code     int            `json:"code,omitempty"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func New(status int) *Error {
	e := &Error{Status: status, Message: http.StatusText(status)}
	return e
}

func Newf(status int, f string, a ...any) *Error {
	return New(status).WithMsgf(f, a...)
}

func (e *Error) WithCode(code int) *Error {
	e.Code = code
	return e
}

func (e *Error) WithMsgf(f string, a ...any) *Error {
	if len(a) == 0 {
		e.Message = f
	} else {
		e.Message = fmt.Sprintf(f, a...)
	}
	return e
}

func (e *Error) WithMetadata(mds map[string]any) *Error {
	e.Metadata = mds
	return e
}

func (e *Error) AppendMetadata(mds map[string]any) *Error {
	if e.Metadata == nil {
		e.Metadata = mds
	} else {
		for k, v := range mds {
			e.Metadata[k] = v
		}
	}
	return e
}

func (e *Error) Error() string {
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
	ss := []string{"status=" + strconv.Itoa(e.Status)}
	if e.Code > 0 {
		ss = append(ss, "code="+strconv.Itoa(e.Code))
	}
	ss = append(ss, "message="+e.Message)
	for k, v := range e.Metadata {
		ss = append(ss, fmt.Sprintf("%s=%s", k, cast.To[string](v)))
	}
	return strings.Join(ss, ", ")
}

func BadRequest() *Error   { return New(http.StatusBadRequest) }
func Unauthorized() *Error { return New(http.StatusUnauthorized) }
func Forbidden() *Error    { return New(http.StatusForbidden) }
func NotFound() *Error     { return New(http.StatusNotFound) }
func Conflict() *Error     { return New(http.StatusConflict) }
func Default() *Error      { return New(http.StatusInternalServerError) }
