// Package models contient les structures de données du service post.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Rôles valides (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// Post — document de la collection `posts`. Champs en snake_case, alignés
// sur le validateur $jsonSchema (cf. internal/database/init.go).
//
// LikesCount / CommentsCount sont des compteurs dénormalisés, maintenus par
// `$inc` au fil des likes/commentaires (le validateur Mongo les déclare en
// `int` → on utilise `int32`, sinon un `int64` casserait la validation).
type Post struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthorID      string        `bson:"author_id" json:"author_id"`
	Content       string        `bson:"content" json:"content"`
	LikesCount    int32         `bson:"likes_count" json:"likes_count"`
	CommentsCount int32         `bson:"comments_count" json:"comments_count"`
	CreatedAt     time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at" json:"updated_at"`
}

// Comment — document de la collection `comments`. Un commentaire référence son
// post par l'identifiant hexadécimal (`post_id`, string) ; l'auteur est dérivé
// du JWT, jamais du corps.
//
// Threading à 2 niveaux : `parent_id` vide = commentaire racine, sinon = id du
// commentaire racine auquel la réponse est rattachée (réponses à plat sous leur
// racine, façon Instagram). `reply_count` (dénormalisé, `int32`) = nombre de
// réponses d'un commentaire racine.
type Comment struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID     string        `bson:"post_id" json:"post_id"`
	ParentID   string        `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
	AuthorID   string        `bson:"author_id" json:"author_id"`
	Content    string        `bson:"content" json:"content"`
	ReplyCount int32         `bson:"reply_count" json:"reply_count"`
	CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time     `bson:"updated_at" json:"updated_at"`
}

// CreatePostRequest : corps de POST /posts. L'auteur n'est PAS dans le corps —
// il est dérivé du JWT (un utilisateur ne poste que pour lui-même).
type CreatePostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

// UpdatePostRequest : corps de PATCH /posts/:id.
type UpdatePostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

// CreateCommentRequest : corps de POST /posts/:id/comments. L'auteur est dérivé
// du JWT, le post de l'URL. `parent_id` (optionnel) cible le commentaire auquel
// on répond (rattaché à plat à la racine côté service).
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,max=280"`
	ParentID string `json:"parent_id"`
}
