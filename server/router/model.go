package router

import (
	"github.com/starudream/aichat-proxy/server/config"
)

type ListModelResp struct {
	Object string   `json:"object"`
	Data   []*Model `json:"data"`
}

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

const defaultCreated = 1751731200

// Chat Completions
//
//	@router			/models [get]
//	@router			/v1/models [get]
//	@summary		Model List
//	@description	Follows the exact same API spec as `https://platform.openai.com/docs/api-reference/models/list`
//	@tags			1_model
//	@security		ApiKeyAuth
//	@produce		json
//	@success		200	{object}	ListModelResp
func hdrModels(c *Ctx) error {
	return c.JSON(&ListModelResp{
		Object: "list",
		Data: []*Model{
			{
				Id:      "doubao",
				Object:  "model",
				Created: defaultCreated,
				OwnedBy: config.AppName,
			},
		},
	})
}
