package echox

import (
	stdjson "encoding/json"
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/starudream/aichat-proxy/server/internal/json"
)

type JSONSerializer struct{}

func (JSONSerializer) Serialize(c echo.Context, v any, indent string) error {
	enc := json.NewEncoder(c.Response())
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(v)
}

func (JSONSerializer) Deserialize(c echo.Context, v any) error {
	err := json.NewDecoder(c.Request().Body).Decode(v)
	var ute *stdjson.UnmarshalTypeError
	if errors.As(err, &ute) {
		return echo.NewHTTPError(400, fmt.Sprintf("unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
	}
	var se *stdjson.SyntaxError
	if errors.As(err, &se) {
		return echo.NewHTTPError(400, fmt.Sprintf("syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
	}
	return err
}
