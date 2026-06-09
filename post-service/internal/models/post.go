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
	Media         []MediaRef    `bson:"media,omitempty" json:"media,omitempty"`
	QuotePostID   string        `bson:"quote_post_id,omitempty" json:"quote_post_id,omitempty"`
	LikesCount    int32         `bson:"likes_count" json:"likes_count"`
	CommentsCount int32         `bson:"comments_count" json:"comments_count"`
	RepostsCount  int32         `bson:"reposts_count" json:"reposts_count"`
	PinnedAt      *time.Time    `bson:"pinned_at,omitempty" json:"pinned_at,omitempty"`
	RepostedByID  string        `bson:"-" json:"reposted_by_id,omitempty"`
	RepostedAt    *time.Time    `bson:"-" json:"reposted_at,omitempty"`
	CreatedAt     time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at" json:"updated_at"`
}

// Repost — document de la collection `reposts`. Index unique `post_id+user_id`
// pour que l'action soit idempotente : un utilisateur ne peut repost qu'une
// fois le même post.
type Repost struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID    string        `bson:"post_id" json:"post_id"`
	UserID    string        `bson:"user_id" json:"user_id"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
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

// MediaRef : pièce jointe d'un post (image ou vidéo). Le post-service est
// agnostique du contenu : il ne stocke que l'URL (chemin relatif `/media/<id>`
// servi par le media-service via la gateway) et la nature (image/vidéo, pour
// que le front choisisse `<img>` ou `<video>`). Sert à la fois de modèle DB et
// de payload de requête (tags `binding` ignorés hors ShouldBindJSON).
type MediaRef struct {
	URL  string `bson:"url" json:"url" binding:"required"`
	Type string `bson:"type" json:"type" binding:"oneof=image video"`
}

// CreatePostRequest : corps de POST /posts. L'auteur n'est PAS dans le corps —
// il est dérivé du JWT (un utilisateur ne poste que pour lui-même). `content`
// est optionnel SI au moins un média est joint (vérifié dans le handler) ;
// jusqu'à 4 médias.
type CreatePostRequest struct {
	Content     string     `json:"content" binding:"max=280"`
	Media       []MediaRef `json:"media" binding:"max=4,dive"`
	QuotePostID string     `json:"quote_post_id"`
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
