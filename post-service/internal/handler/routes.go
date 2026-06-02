package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, serviceName string) {
	r.GET("/health", Health(serviceName))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/comments/:commentId", CreateComment)
		posts := v1.Group("/posts")
		{
			posts.GET("", ListPosts) 
			posts.POST("", CreatePost)

			post := posts.Group("/:id")
			{
				post.GET("", GetPost)
				post.DELETE("", DeletePost)

				post.GET("/likes", ListPostLikes)
				post.POST("/like", LikePost) 
				post.DELETE("/like", UnlikePost)

				comment := post.Group("/comments")
				{
					comment.GET("", ListPostComments) 
					comment.POST("", CreatPostComment)
					comment.DELETE("/:commentId", DeletePostComment)
				}
			}
		}

		profile := v1.Group("/profile/:id")
		{
			profile.GET("/likes", ListProfileLikes)
			profile.GET("/posts", ListProfilePosts) 
			profile.GET("/comments", ListProfileComments) 
		}


	}
}

