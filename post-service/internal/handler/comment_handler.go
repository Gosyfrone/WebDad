package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/service"
)

type CommentHandler struct {
	service *service.PostService
	name    string
}

func NewCommentHandler(svc *service.PostService, serviceName string) *CommentHandler {
	return &CommentHandler{
		service: svc,
		name:    serviceName,
	}
}

func (h *CommentHandler) ListPostComments(c *gin.Context)    {}
func (h *CommentHandler) CreatPostComment(c *gin.Context)    {}
func (h *CommentHandler) DeletePostComment(c *gin.Context)   {}
func (h *CommentHandler) ListProfileComments(c *gin.Context) {}
func (h *CommentHandler) CreateComment(c *gin.Context)       {}
