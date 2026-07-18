package router

import (
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/api"
	tr069Middleware "github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type ArtifactRouter struct {
	api *api.ArtifactApi
}

func NewArtifactRouter(transfers *service.TransferStore, objects service.ArtifactStore) *ArtifactRouter {
	return &ArtifactRouter{api: api.NewArtifactApi(transfers, objects)}
}

func (r *ArtifactRouter) InitArtifactRouter(parent *gin.RouterGroup) {
	artifacts := parent.Group("artifact")
	artifacts.GET("list", r.api.List)
	artifacts.GET(":artifactId/download", tr069Middleware.DownloadAudit(), r.api.Download)
}
