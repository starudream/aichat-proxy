package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/docs"
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
func setupSwagger(app *fiber.App) {
	docs.SwaggerInfo.Version = config.GetVersion().GitVersion
	app.Get("/swagger/*", swagger.New(swagger.Config{
		TagsSorter:             "'alpha'",
		TryItOutEnabled:        true,
		RequestSnippetsEnabled: true,
		DisplayRequestDuration: true,
	}))
}

func setupRoutes(app *fiber.App) {
	app.Get("/", hdrIndex)

	v1 := app.Group("/v1", mdLogger(), mdAuth())
	{
		v1.Get("/models", hdrModels)
		v1.Post("/chat/completions", hdrChatCompletions)
	}
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
func hdrIndex(c *fiber.Ctx) error {
	ver := config.GetVersion()
	return c.JSON(&Index{
		AppName:    config.AppName,
		GitVersion: ver.GitVersion,
		BuildDate:  ver.BuildDate,
	})
}
