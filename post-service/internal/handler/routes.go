package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/webdad/post-service/internal/service"
)

func RegisterRoutes(r *gin.Engine, serviceName string, postService *service.PostService) {
	r.GET("/health", Health(serviceName))
	PostHandler := NewPostHandler(postService, serviceName)
	LikeHandler := NewLikeHandler(postService, serviceName)
	CommentHandler := NewCommentHandler(postService, serviceName)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/comments/:commentId", CommentHandler.CreateComment)

		posts := v1.Group("/posts")
		{
			posts.GET("", PostHandler.ListPosts)
			posts.POST("", PostHandler.CreatePost)

			post := posts.Group("/:id")
			{
				post.GET("", PostHandler.GetPost)
				post.PATCH("", PostHandler.UpdatePost)
				post.DELETE("", PostHandler.DeletePost)
			

				post.GET("/likes", LikeHandler.ListPostLikes)
				post.POST("/like", LikeHandler.LikePost)
				post.DELETE("/like", LikeHandler.UnlikePost)

				comment := post.Group("/comments")
				{
					comment.GET("", CommentHandler.ListPostComments)
					comment.POST("", CommentHandler.CreatPostComment)
					comment.DELETE("/:commentId", CommentHandler.DeletePostComment)
				}
			}
		}

		profile := v1.Group("/profile/:id")
		{
			profile.GET("/likes", LikeHandler.ListProfileLikes)
			profile.GET("/posts", PostHandler.ListProfilePosts)
			profile.GET("/comments", CommentHandler.ListProfileComments)
		}
	}
}
