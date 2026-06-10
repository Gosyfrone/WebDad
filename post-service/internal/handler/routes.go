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
	optionalAuth := middleware.OptionalJWTAuth(jwtSecret)

	r.GET("/health", Health(serviceName))
	PostHandler := NewPostHandler(postService, serviceName)
	LikeHandler := NewLikeHandler(postService, serviceName)
	CommentHandler := NewCommentHandler(postService, serviceName)
	BookmarkHandler := NewBookmarkHandler(postService, serviceName)

	posts := r.Group("/posts")
	{
		// Lecture publique. Le fil par auteur est un filtre de la liste :
		// GET /posts?author_id=<id> (cf. ListPosts) — pas de préfixe séparé,
		// pour rester sous `/posts` (le seul routé par la gateway).
		posts.GET("", optionalAuth, PostHandler.ListPosts)
		// Création authentifiée (auteur dérivé du JWT).
		posts.POST("", auth, PostHandler.CreatePost)

		// Posts likés par l'utilisateur courant — route STATIQUE placée avant
		// le groupe `/:id` (sinon « me » serait capturé comme un id).
		posts.GET("/me/liked-ids", auth, LikeHandler.LikedByMe)
		posts.GET("/me/reposted-ids", auth, PostHandler.RepostedByMe)
		posts.GET("/me/bookmarked-ids", auth, BookmarkHandler.BookmarkedByMe)

		// Modération — routes STATIQUES (`/posts/moderation/…`) placées avant le
		// groupe `/:id`. Corbeille partagée mod/admin (tweets retirés en
		// suppression douce). Garde ModeratorOnly (mod ou admin).
		moderation := posts.Group("/moderation", auth, middleware.ModeratorOnly())
		{
			moderation.GET("/deleted", PostHandler.ListHidden)
		}

		// Effacement RGPD : purge toutes les données d'un utilisateur (admin).
		// Route STATIQUE (`/posts/by-author/:id`) avant le groupe `/:id`.
		posts.DELETE("/by-author/:id", auth, middleware.AdminOnly(), PostHandler.PurgeUserData)

		// Signets — routes STATIQUES (`/posts/bookmarks/…`), placées avant le
		// groupe `/:id` ; toutes protégées (les signets sont strictement privés).
		bookmarks := posts.Group("/bookmarks", auth)
		{
			bookmarks.GET("", BookmarkHandler.ListAll) // vue « Tous mes signets »
			collections := bookmarks.Group("/collections")
			{
				collections.GET("", BookmarkHandler.ListCollections)
				collections.POST("", BookmarkHandler.CreateCollection)
				collection := collections.Group("/:cid")
				{
					collection.PATCH("", BookmarkHandler.RenameCollection)
					collection.DELETE("", BookmarkHandler.DeleteCollection)
					collection.GET("/posts", BookmarkHandler.ListCollectionPosts)
				}
			}
		}

		post := posts.Group("/:id")
		{
			post.GET("", optionalAuth, PostHandler.GetPost)
			// Édition / suppression : authentifiées (auteur ou modérateur/admin).
			post.PATCH("", auth, PostHandler.UpdatePost)
			post.DELETE("", auth, PostHandler.DeletePost)
			post.PATCH("/pin", auth, PostHandler.PinPost)
			post.DELETE("/pin", auth, PostHandler.UnpinPost)
			// Modération d'un post précis (mod/admin) : restaurer depuis la
			// corbeille ou effacer définitivement.
			post.POST("/restore", auth, middleware.ModeratorOnly(), PostHandler.RestorePost)
			post.DELETE("/purge", auth, middleware.ModeratorOnly(), PostHandler.PurgePost)

			post.GET("/likes", LikeHandler.ListPostLikes)
			post.POST("/like", auth, LikeHandler.LikePost)
			post.DELETE("/like", auth, LikeHandler.UnlikePost)
			post.POST("/repost", auth, PostHandler.RepostPost)
			post.DELETE("/repost", auth, PostHandler.UnrepostPost)

			post.POST("/bookmark", auth, BookmarkHandler.Bookmark)
			post.DELETE("/bookmark", auth, BookmarkHandler.Unbookmark)
			post.GET("/bookmark/collections", auth, BookmarkHandler.PostCollections)

			comment := post.Group("/comments")
			{
				comment.GET("", CommentHandler.ListPostComments)
				comment.POST("", auth, CommentHandler.CreatPostComment)
				comment.GET("/:commentId/replies", CommentHandler.ListCommentReplies)
				comment.DELETE("/:commentId", auth, CommentHandler.DeletePostComment)
			}
		}
	}
}
