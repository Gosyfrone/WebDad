package models

import (
        "time"
        "go.mongodb.org/mongo-driver/v2/bson"

)

type Post struct {
    ID      bson.ObjectID `bson:"_id,omitempty" json:"id"`
    PublishedBy   string        `bson:"published_by" json:"published_by"`
    Content string        `bson:"content" json:"content"`
    PublishedAt time.Time  `bson:"published_at" json:"published_at"`
}