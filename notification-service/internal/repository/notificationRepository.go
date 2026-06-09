// Package repository : accès aux données Mongo du notification-service. Les
// méthodes renvoient les erreurs brutes du driver (notamment mongo.ErrNoDocuments) ;
// la traduction en erreurs métier est faite par la couche service.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/notification-service/internal/models"
)

// NotificationRepository encapsule la collection `notifications`.
type NotificationRepository struct {
	notifications *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) *NotificationRepository {
	return &NotificationRepository{notifications: db.Collection("notifications")}
}

// Upsert agrège un événement dans le groupe (recipient_id, group_key) : crée la
// notification si elle n'existe pas, sinon incrémente son compteur et la fait
// remonter (updated_at) en la repassant non-lue. Renvoie le document à jour
// (pour la diffusion temps réel). C'est le cœur du « stacking » : 300 likes sur
// un post = 300 appels ici → un seul document, count = 300.
func (r *NotificationRepository) Upsert(ctx context.Context, n *models.Notification) (*models.Notification, error) {
	now := time.Now()
	filter := bson.M{"recipient_id": n.RecipientID, "group_key": n.GroupKey}
	update := bson.M{
		"$set": bson.M{
			"type":          n.Type,
			"post_id":       n.PostID,
			"comment_id":    n.CommentID,
			"last_actor_id": n.LastActorID,
			"is_read":       false,
			"updated_at":    now,
		},
		"$inc":         bson.M{"count": int32(1)},
		"$setOnInsert": bson.M{"recipient_id": n.RecipientID, "group_key": n.GroupKey, "created_at": now},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var out models.Notification
	if err := r.notifications.FindOneAndUpdate(ctx, filter, update, opts).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Decrement défait un événement : décrémente le compteur du groupe. Si le
// compteur tombe à zéro (ou moins), supprime la notification et renvoie
// `deleted = true` avec le document supprimé (pour pousser une suppression temps
// réel). mongo.ErrNoDocuments si le groupe n'existe pas (rien à défaire).
func (r *NotificationRepository) Decrement(ctx context.Context, recipientID, groupKey string) (n *models.Notification, deleted bool, err error) {
	filter := bson.M{"recipient_id": recipientID, "group_key": groupKey}
	update := bson.M{"$inc": bson.M{"count": int32(-1)}, "$set": bson.M{"updated_at": time.Now()}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var out models.Notification
	if err := r.notifications.FindOneAndUpdate(ctx, filter, update, opts).Decode(&out); err != nil {
		return nil, false, err
	}
	if out.Count <= 0 {
		if _, err := r.notifications.DeleteOne(ctx, bson.M{"_id": out.ID}); err != nil {
			return nil, false, err
		}
		return &out, true, nil
	}
	return &out, false, nil
}

// DeleteByPost purge toutes les notifications liées à un post (cascade à sa
// suppression) et renvoie les destinataires impactés (pour pousser la mise à
// jour temps réel à chacun).
func (r *NotificationRepository) DeleteByPost(ctx context.Context, postID string) ([]string, error) {
	recipients, err := r.distinctRecipients(ctx, bson.M{"post_id": postID})
	if err != nil {
		return nil, err
	}
	if _, err := r.notifications.DeleteMany(ctx, bson.M{"post_id": postID}); err != nil {
		return nil, err
	}
	return recipients, nil
}

func (r *NotificationRepository) distinctRecipients(ctx context.Context, filter bson.M) ([]string, error) {
	out := []string{}
	if err := r.notifications.Distinct(ctx, "recipient_id", filter).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// List renvoie les notifications d'un utilisateur, de la plus récente à la plus
// ancienne, paginées par curseur. `before` (ObjectID, optionnel) renvoie les
// notifications ANTÉRIEURES (scroll vers le bas / pages suivantes).
func (r *NotificationRepository) List(ctx context.Context, recipientID string, limit int64, before *bson.ObjectID) ([]models.Notification, error) {
	filter := bson.M{"recipient_id": recipientID}
	if before != nil {
		filter["_id"] = bson.M{"$lt": *before}
	}
	opts := options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(limit)

	cursor, err := r.notifications.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	out := []models.Notification{}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CountUnread renvoie le nombre de notifications non lues d'un utilisateur (badge).
func (r *NotificationRepository) CountUnread(ctx context.Context, recipientID string) (int64, error) {
	return r.notifications.CountDocuments(ctx, bson.M{"recipient_id": recipientID, "is_read": false})
}

// MarkAllRead passe toutes les notifications d'un utilisateur en lues.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, recipientID string) error {
	_, err := r.notifications.UpdateMany(ctx,
		bson.M{"recipient_id": recipientID, "is_read": false},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	return err
}

// MarkRead passe une notification en lue (réservée à son destinataire via le
// filtre). mongo.ErrNoDocuments si elle n'existe pas / n'appartient pas à l'user.
func (r *NotificationRepository) MarkRead(ctx context.Context, recipientID string, id bson.ObjectID) error {
	res, err := r.notifications.UpdateOne(ctx,
		bson.M{"_id": id, "recipient_id": recipientID},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
