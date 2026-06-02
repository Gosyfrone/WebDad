package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/service"
)

type LikeHandler struct {
	service *service.PostService
	name    string
}

func NewLikeHandler(svc *service.PostService, serviceName string) *LikeHandler {
	return &LikeHandler{
		service: svc,
		name:    serviceName,
	}
}

func (h *LikeHandler) ListPostLikes(c *gin.Context)    {}
func (h *LikeHandler) LikePost(c *gin.Context)         {}
func (h *LikeHandler) UnlikePost(c *gin.Context)       {}
func (h *LikeHandler) ListProfileLikes(c *gin.Context) {}
