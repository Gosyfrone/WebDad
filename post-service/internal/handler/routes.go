package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, serviceName string) {
	r.GET("/health", Health(serviceName))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/comments/:commentId", CreateComment(serviceName))
		posts := v1.Group("/posts")
		{
			posts.GET("", ListPosts(serviceName))
			posts.POST("", CreatePost(serviceName))

			post := posts.Group("/:id")
			{
				post.GET("", GetPost(serviceName))
				post.DELETE("", DeletePost(serviceName))

				post.GET("/likes", ListPostLikes(serviceName))
				post.POST("/like", LikePost(serviceName))
				post.DELETE("/like", UnlikePost(serviceName))

				comment := post.Group("/comments")
				{
					comment.GET("", ListPostComments(serviceName))
					comment.POST("", CreatPostComment(serviceName))
					comment.DELETE("/:commentId", DeletePostComment(serviceName))
				}
			}
		}

		profile := v1.Group("/profile/:id")
		{
			profile.GET("/likes", ListProfileLikes(serviceName))
			profile.GET("/posts", ListProfilePosts(serviceName))
			profile.GET("/comments", ListProfileComments(serviceName))
		}
	}
}

