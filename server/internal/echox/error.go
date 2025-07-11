package echox

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/internal/errx"
)

func ErrorHandler(app *echo.Echo) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		var he *echo.HTTPError
		if errors.As(err, &he) {
			if he.Internal != nil {
				var _he *echo.HTTPError
				if errors.As(he.Internal, &_he) {
					he = _he
				}
			}
		} else {
			he = &echo.HTTPError{
				Code:    http.StatusInternalServerError,
				Message: http.StatusText(http.StatusInternalServerError),
			}
		}

		var ee *errx.Error
		if !errors.As(err, &ee) {
			ee = errx.Newf(he.Code, cast.To[string](he.Message))
		}

		if c.Request().Method == http.MethodHead {
			err = c.NoContent(ee.Status)
		} else {
			err = c.JSON(ee.Status, ee)
		}
		if err != nil {
			app.Logger.Error(err)
		}
	}
}
