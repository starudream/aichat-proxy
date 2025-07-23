package api

import (
	"github.com/labstack/echo/v4"
	swagger "github.com/swaggo/echo-swagger"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/docs"
	"github.com/starudream/aichat-proxy/server/internal/echox"
)

// General Swagger API Info
//
//	@title						AIChat Proxy API
//	@version					1.0
//	@contact.name				github repo
//	@contact.url				https://github.com/starudream/aichat-proxy
//	@license.name				Apache-2.0
//	@license.url				https://www.apache.org/licenses/LICENSE-2.0
//	@tag.name					common
//	@tag.name					model
//	@tag.name					chat
//	@accept						json
//	@produce					json
//	@schemes					http
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
func setupRoutes(app *echo.Echo) {
	app.GET("/", hdrIndex)

	v1 := app.Group("/v1", echox.MiddlewareLogger(), mdAuth())
	{
		v1.GET("/models", hdrModels)
		v1.POST("/chat/completions", hdrChatCompletions)
	}
}

func setupSwagger(app *echo.Echo) {
	docs.SwaggerInfo.Version = config.GetVersion().GitVersion
	app.GET("/swagger/*", swagger.EchoWrapHandler(
		swagger.PersistAuthorization(true),
	))
}

type Index struct {
	AppName    string `json:"app_name"`
	GitVersion string `json:"git_version"`
	BuildDate  string `json:"build_date"`
}

// Index
//
//	@router		/ [get]
//	@summary	Index
//	@tags		common
//	@success	200	{object}	Index
func hdrIndex(c echo.Context) error {
	ver := config.GetVersion()
	return c.JSON(200, &Index{
		AppName:    config.AppName,
		GitVersion: ver.GitVersion,
		BuildDate:  ver.BuildDate,
	})
}
