package router

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
	app.Get("/health", hdrHealth)
	app.Get("/version", hdrVersion)

	app.Get("/models", hdrModels)
	app.Get("/v1/models", hdrModels)

	app.Post("/chat/completions", hdrChatCompletions)
	app.Post("/v1/chat/completions", hdrChatCompletions)
}

// Health Check
//
//	@router		/health [get]
//	@summary	Health Check
//	@tags		0_global
//	@produce	plain
//	@success	200	{string}	string	"OK
func hdrHealth(c *fiber.Ctx) error {
	return c.SendString("OK")
}

// Show Version
//
//	@router		/version [get]
//	@summary	show version
//	@tags		0_global
//	@success	200	{object}	config.Version	"OK
func hdrVersion(c *fiber.Ctx) error {
	return c.JSON(config.GetVersion())
}
