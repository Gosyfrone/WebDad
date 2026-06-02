package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/service"
)

type PostHandler struct {
	service *service.PostService
	name    string
}

func NewPostHandler(svc *service.PostService, serviceName string) *PostHandler {
	return &PostHandler{
		service: svc,
		name:    serviceName,
	}
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	var req struct {
		AuthorID string `json:"author_id" binding:"required"`
		Content  string `json:"content" binding:"required,max=280"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "payload invalide : " + err.Error()})
		return
	}

	if err := h.service.CreatePost(c.Request.Context(), req.AuthorID, req.Content); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"status": "created"})
}

func (h *PostHandler) ListPosts(c *gin.Context) {
	posts, err := h.service.GetPosts(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status": "ok",
		"posts":  posts,
	})
}

func (h *PostHandler) GetPost(c *gin.Context) {}

func (h *PostHandler) DeletePost(c *gin.Context) {}

func (h *PostHandler) ListProfilePosts(c *gin.Context) {}
