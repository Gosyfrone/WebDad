// Package repository : accès aux données Mongo du post-service. Les méthodes
// renvoient les erreurs brutes du driver (notamment mongo.ErrNoDocuments) ; la
// traduction en erreurs métier est faite par la couche service.
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

// Create insère le post et renseigne post.ID avec l'identifiant généré.
func (r *PostRepository) Create(ctx context.Context, post *models.Post) error {
	res, err := r.collection.InsertOne(ctx, post)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		post.ID = oid
	}
	return nil
}

// GetAll renvoie le fil global trié du plus récent au plus ancien, paginé.
func (r *PostRepository) GetAll(ctx context.Context, limit, skip int64) ([]models.Post, error) {
	return r.find(ctx, bson.M{}, limit, skip)
}

// GetByProfile renvoie les posts d'un auteur, triés du plus récent au plus ancien.
func (r *PostRepository) GetByProfile(ctx context.Context, authorID string, limit, skip int64) ([]models.Post, error) {
	return r.find(ctx, bson.M{"author_id": authorID}, limit, skip)
}

// find factorise la lecture paginée + triée des posts.
func (r *PostRepository) find(ctx context.Context, filter bson.M, limit, skip int64) ([]models.Post, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	posts := []models.Post{}
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// Get renvoie un post par son ObjectID (mongo.ErrNoDocuments si absent).
func (r *PostRepository) Get(ctx context.Context, id bson.ObjectID) (*models.Post, error) {
	var post models.Post
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// Delete supprime un post (mongo.ErrNoDocuments si rien n'a été supprimé).
func (r *PostRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Update modifie le contenu et renvoie le document à jour (ReturnDocument
// After). mongo.ErrNoDocuments si le post n'existe pas.
func (r *PostRepository) Update(ctx context.Context, id bson.ObjectID, content string) (*models.Post, error) {
	update := bson.M{
		"$set": bson.M{
			"content":    content,
			"updated_at": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}
