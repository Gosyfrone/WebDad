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

// PostRepository agrège les collections du domaine post (posts + likes +
// comments). Les likes/comments référencent leur post par l'identifiant
// hexadécimal (`post_id`, string) ; les compteurs dénormalisés vivent sur le
// document `posts` et sont maintenus par `$inc`.
type PostRepository struct {
	posts    *mongo.Collection
	likes    *mongo.Collection
	comments *mongo.Collection
}

func NewPostRepository(db *mongo.Database) *PostRepository {
	return &PostRepository{
		posts:    db.Collection("posts"),
		likes:    db.Collection("likes"),
		comments: db.Collection("comments"),
	}
}

// --- Posts -------------------------------------------------------------------

// Create insère le post et renseigne post.ID avec l'identifiant généré.
func (r *PostRepository) Create(ctx context.Context, post *models.Post) error {
	res, err := r.posts.InsertOne(ctx, post)
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

// GetByAuthors renvoie les posts d'un ensemble d'auteurs (fil « Abonnements »),
// triés du plus récent au plus ancien. Une seule requête indexée (`$in` sur
// author_id) : la sélection est faite côté DB, pas côté client.
func (r *PostRepository) GetByAuthors(ctx context.Context, authorIDs []string, limit, skip int64) ([]models.Post, error) {
	return r.find(ctx, bson.M{"author_id": bson.M{"$in": authorIDs}}, limit, skip)
}

// find factorise la lecture paginée + triée des posts.
func (r *PostRepository) find(ctx context.Context, filter bson.M, limit, skip int64) ([]models.Post, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.posts.Find(ctx, filter, opts)
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
	if err := r.posts.FindOne(ctx, bson.M{"_id": id}).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// Delete supprime un post (mongo.ErrNoDocuments si rien n'a été supprimé).
func (r *PostRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.posts.DeleteOne(ctx, bson.M{"_id": id})
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
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// IncCounter applique `$inc` sur un compteur dénormalisé (likes_count /
// comments_count) et renvoie le post à jour. Le compteur reste `int32` pour
// rester conforme au validateur $jsonSchema (bsonType "int").
func (r *PostRepository) IncCounter(ctx context.Context, id bson.ObjectID, field string, delta int32) (*models.Post, error) {
	update := bson.M{"$inc": bson.M{field: delta}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var post models.Post
	if err := r.posts.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

// --- Likes -------------------------------------------------------------------

// AddLike enregistre un like (idempotent grâce à l'index unique
// post_id+user_id). Renvoie true si le like a été créé, false s'il existait
// déjà (clé dupliquée) → le service n'incrémente le compteur que dans le 1er cas.
func (r *PostRepository) AddLike(ctx context.Context, postID, userID string) (bool, error) {
	_, err := r.likes.InsertOne(ctx, bson.M{
		"post_id":    postID,
		"user_id":    userID,
		"created_at": time.Now(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoveLike supprime un like. Renvoie true si un like a effectivement été
// supprimé (sinon le service n'a pas à décrémenter).
func (r *PostRepository) RemoveLike(ctx context.Context, postID, userID string) (bool, error) {
	res, err := r.likes.DeleteOne(ctx, bson.M{"post_id": postID, "user_id": userID})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// LikedPostIDs renvoie les ids des posts likés par un utilisateur (sert à
// initialiser l'état des cœurs côté front, façon getFollowingIds).
func (r *PostRepository) LikedPostIDs(ctx context.Context, userID string) ([]string, error) {
	return r.distinctStrings(ctx, r.likes, bson.M{"user_id": userID}, "post_id")
}

// LikersByPost renvoie les ids des utilisateurs ayant liké un post.
func (r *PostRepository) LikersByPost(ctx context.Context, postID string) ([]string, error) {
	return r.distinctStrings(ctx, r.likes, bson.M{"post_id": postID}, "user_id")
}

// DeleteLikesByPost purge les likes d'un post (nettoyage à la suppression).
func (r *PostRepository) DeleteLikesByPost(ctx context.Context, postID string) error {
	_, err := r.likes.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

// --- Comments ----------------------------------------------------------------

// AddComment insère le commentaire et renseigne comment.ID.
func (r *PostRepository) AddComment(ctx context.Context, comment *models.Comment) error {
	res, err := r.comments.InsertOne(ctx, comment)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		comment.ID = oid
	}
	return nil
}

// ListComments renvoie les commentaires d'un post, du plus ancien au plus
// récent (lecture chronologique d'un fil), paginés.
func (r *PostRepository) ListComments(ctx context.Context, postID string, limit, skip int64) ([]models.Comment, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.comments.Find(ctx, bson.M{"post_id": postID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	comments := []models.Comment{}
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// GetComment renvoie un commentaire par son ObjectID (mongo.ErrNoDocuments si absent).
func (r *PostRepository) GetComment(ctx context.Context, id bson.ObjectID) (*models.Comment, error) {
	var comment models.Comment
	if err := r.comments.FindOne(ctx, bson.M{"_id": id}).Decode(&comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeleteComment supprime un commentaire (mongo.ErrNoDocuments si rien supprimé).
func (r *PostRepository) DeleteComment(ctx context.Context, id bson.ObjectID) error {
	res, err := r.comments.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteCommentsByPost purge les commentaires d'un post (nettoyage à la suppression).
func (r *PostRepository) DeleteCommentsByPost(ctx context.Context, postID string) error {
	_, err := r.comments.DeleteMany(ctx, bson.M{"post_id": postID})
	return err
}

// distinctStrings factorise la projection d'un champ string sur un filtre.
func (r *PostRepository) distinctStrings(ctx context.Context, coll *mongo.Collection, filter bson.M, field string) ([]string, error) {
	cursor, err := coll.Find(ctx, filter, options.Find().SetProjection(bson.M{field: 1}))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(docs))
	for _, d := range docs {
		if v, ok := d[field].(string); ok {
			ids = append(ids, v)
		}
	}
	return ids, nil
}
