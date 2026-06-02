package database

import (
    "context"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
)

func InitMongo(db *mongo.Database) {
    posts := db.Collection("posts")

    index := mongo.IndexModel{
        Keys: bson.D{{Key: "authorId", Value: 1}},
    }

    posts.Indexes().CreateOne(context.TODO(), index)
}