package api

import (
	"github.com/starudream/aichat-proxy/server/browser"
	"github.com/starudream/aichat-proxy/server/config"
)

type ListModelResp struct {
	// 固定为 list
	Object string `json:"object"`
	// 模型列表
	Data []*Model `json:"data"`
}

type Model struct {
	// 模型 Id
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

const defaultCreated = 1751731200

// Chat Completions
//
//	@router			/v1/models [get]
//	@summary		Model List
//	@description	Follows the exact same API spec as `https://platform.openai.com/docs/api-reference/models/list`
//	@tags			model
//	@security		ApiKeyAuth
//	@success		200	{object}	ListModelResp
func hdrModels(c Ctx) error {
	models := make([]*Model, 0)
	for _, m := range browser.Models() {
		models = append(models, &Model{
			Id:      m,
			Object:  "model",
			Created: defaultCreated,
			OwnedBy: config.AppName,
		})
	}
	return c.JSON(200, &ListModelResp{Object: "list", Data: models})
}
