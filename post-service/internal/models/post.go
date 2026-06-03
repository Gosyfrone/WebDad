package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
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
