package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
	"github.com/webdad/post-service/internal/models"
)

type PostRepository struct {
	collection *mongo.Collection
}

func NewPostRepository(db *mongo.Database) *PostRepository {
	return &PostRepository{
		collection: db.Collection("posts"),
	}
}

func (r *PostRepository) Create(ctx context.Context, post *models.Post) error {
	_, err := r.collection.InsertOne(ctx, post)
	return err
}

func (r *PostRepository) GetAll(ctx context.Context) ([]models.Post, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

    var posts []models.Post
    if err := cursor.All(ctx, &posts); err != nil {
        return nil, err
    }

    return posts, nil
}

func (r *PostRepository) Get(ctx context.Context, id string) (*models.Post, error) {

    objectID, err := bson.ObjectIDFromHex(id)
    if err != nil {
        return nil, err
    }

    var post models.Post

    err = r.collection.FindOne(ctx, bson.M{
        "_id": objectID,
    }).Decode(&post)

    if err != nil {
        return nil, err
    }

    return &post, nil
}

func (r *PostRepository) Delete(ctx context.Context, id string) error {

    objectID, err := bson.ObjectIDFromHex(id)
    if err != nil {
        return err
    }

    result, err := r.collection.DeleteOne(ctx, bson.M{
        "_id": objectID,
    })

    if err != nil {
        return err
    }

    if result.DeletedCount == 0 {
        return mongo.ErrNoDocuments
    }

    return nil
}

func (r *PostRepository) Update(ctx context.Context, id string, content string) (*models.Post, error) {
    objectID, err := bson.ObjectIDFromHex(id)
    if err != nil {
        return nil,err
    }

    update := bson.M{
        "$set": bson.M{
            "content":   content,
            "updated_at": time.Now(),
        },
    }

    var post models.Post
    opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
    err = r.collection.FindOneAndUpdate(
        ctx,
        bson.M{"_id": objectID},
        update,
        opts,
    ).Decode(&post)

    if err != nil {
        return nil,err
    }

    return &post, nil
}

func (r *PostRepository) GetByProfile(ctx context.Context, id string) ([]models.Post, error) {
    cursor, err := r.collection.Find(ctx, bson.M{"author_id": id})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var posts []models.Post
    if err := cursor.All(ctx, &posts); err != nil {
        return nil, err
    }

    return posts, nil
}