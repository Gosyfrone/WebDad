package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/webdad/post-service/internal/middleware"
	"github.com/webdad/post-service/internal/service"
)

// RegisterRoutes enregistre les routes du post-service. Toutes vivent sous le
// préfixe `/posts` : c'est le seul préfixe que l'API Gateway mappe vers ce
// service (la gateway conserve le chemin tel quel), donc tout doit en dériver
// pour être atteignable par le front. jwtSecret protège les routes mutables
// (validation locale du token émis par auth-service, secret partagé) ; la
// lecture reste publique.
func RegisterRoutes(r *gin.Engine, serviceName string, postService *service.PostService, jwtSecret string) {
	auth := middleware.JWTAuth(jwtSecret)

	r.GET("/health", Health(serviceName))
	PostHandler := NewPostHandler(postService, serviceName)
	LikeHandler := NewLikeHandler(postService, serviceName)
	CommentHandler := NewCommentHandler(postService, serviceName)

	posts := r.Group("/posts")
	{
		// Lecture publique. Le fil par auteur est un filtre de la liste :
		// GET /posts?author_id=<id> (cf. ListPosts) — pas de préfixe séparé,
		// pour rester sous `/posts` (le seul routé par la gateway).
		posts.GET("", PostHandler.ListPosts)
		// Création authentifiée (auteur dérivé du JWT).
		posts.POST("", auth, PostHandler.CreatePost)

		// Posts likés par l'utilisateur courant — route STATIQUE placée avant
		// le groupe `/:id` (sinon « me » serait capturé comme un id).
		posts.GET("/me/liked-ids", auth, LikeHandler.LikedByMe)

		post := posts.Group("/:id")
		{
			post.GET("", PostHandler.GetPost)
			// Édition / suppression : authentifiées (auteur ou modérateur/admin).
			post.PATCH("", auth, PostHandler.UpdatePost)
			post.DELETE("", auth, PostHandler.DeletePost)

			post.GET("/likes", LikeHandler.ListPostLikes)
			post.POST("/like", auth, LikeHandler.LikePost)
			post.DELETE("/like", auth, LikeHandler.UnlikePost)

			comment := post.Group("/comments")
			{
				comment.GET("", CommentHandler.ListPostComments)
				comment.POST("", auth, CommentHandler.CreatPostComment)
				comment.DELETE("/:commentId", auth, CommentHandler.DeletePostComment)
			}
		}
	}
}
