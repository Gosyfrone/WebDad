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
type Post struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthorID  string        `bson:"author_id" json:"author_id"`
	Content   string        `bson:"content" json:"content"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
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
