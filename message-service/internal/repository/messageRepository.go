// Package repository : accès aux données Mongo du message-service. Les méthodes
// renvoient les erreurs brutes du driver (notamment mongo.ErrNoDocuments) ; la
// traduction en erreurs métier est faite par la couche service.
package repository

import (
	"context"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/webdad/message-service/internal/models"
)

// MessageRepository agrège les collections du domaine messagerie.
type MessageRepository struct {
	keys          *mongo.Collection
	conversations *mongo.Collection
	members       *mongo.Collection
	messages      *mongo.Collection
}

func NewMessageRepository(db *mongo.Database) *MessageRepository {
	return &MessageRepository{
		keys:          db.Collection("user_keys"),
		conversations: db.Collection("conversations"),
		members:       db.Collection("members"),
		messages:      db.Collection("messages"),
	}
}

// --- Clés publiques ----------------------------------------------------------

// UpsertKey publie/met à jour la clé publique d'un utilisateur (idempotent).
func (r *MessageRepository) UpsertKey(ctx context.Context, userID, publicKey string) error {
	now := time.Now()
	update := bson.M{
		"$set":         bson.M{"public_key": publicKey, "updated_at": now},
		"$setOnInsert": bson.M{"user_id": userID, "created_at": now},
	}
	_, err := r.keys.UpdateOne(ctx, bson.M{"user_id": userID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

// GetKey renvoie la clé publique d'un utilisateur (mongo.ErrNoDocuments si absente).
func (r *MessageRepository) GetKey(ctx context.Context, userID string) (*models.UserKey, error) {
	var k models.UserKey
	if err := r.keys.FindOne(ctx, bson.M{"user_id": userID}).Decode(&k); err != nil {
		return nil, err
	}
	return &k, nil
}

// --- Conversations -----------------------------------------------------------

// FindDMByKey renvoie le DM identifié par sa paire d'ids triée (mongo.ErrNoDocuments si absent).
func (r *MessageRepository) FindDMByKey(ctx context.Context, dmKey string) (*models.Conversation, error) {
	var conv models.Conversation
	if err := r.conversations.FindOne(ctx, bson.M{"type": models.TypeDM, "dm_key": dmKey}).Decode(&conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// CreateConversation insère la conversation et renseigne conv.ID.
func (r *MessageRepository) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	res, err := r.conversations.InsertOne(ctx, conv)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		conv.ID = oid
	}
	return nil
}

// GetConversation renvoie une conversation par son ObjectID.
func (r *MessageRepository) GetConversation(ctx context.Context, id bson.ObjectID) (*models.Conversation, error) {
	var conv models.Conversation
	if err := r.conversations.FindOne(ctx, bson.M{"_id": id}).Decode(&conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// TouchConversation remonte updated_at (la conversation passe en tête de liste).
func (r *MessageRepository) TouchConversation(ctx context.Context, id bson.ObjectID, at time.Time) error {
	_, err := r.conversations.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"updated_at": at}})
	return err
}

// UpdateTitle remplace le nom chiffré (ciphertext + nonce) d'un groupe et
// renvoie la conversation à jour.
func (r *MessageRepository) UpdateTitle(ctx context.Context, id bson.ObjectID, title, titleNonce string) (*models.Conversation, error) {
	update := bson.M{"$set": bson.M{"title": title, "title_nonce": titleNonce, "updated_at": time.Now()}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var conv models.Conversation
	if err := r.conversations.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// PushMemberID ajoute un id à la liste dénormalisée member_ids (idempotent).
func (r *MessageRepository) PushMemberID(ctx context.Context, id bson.ObjectID, userID string) error {
	_, err := r.conversations.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$addToSet": bson.M{"member_ids": userID}})
	return err
}

// PullMemberID retire un id de la liste dénormalisée member_ids.
func (r *MessageRepository) PullMemberID(ctx context.Context, id bson.ObjectID, userID string) error {
	_, err := r.conversations.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$pull": bson.M{"member_ids": userID}})
	return err
}

// DeleteConversation supprime la conversation (mongo.ErrNoDocuments si absente).
func (r *MessageRepository) DeleteConversation(ctx context.Context, id bson.ObjectID) error {
	res, err := r.conversations.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// --- Membres -----------------------------------------------------------------

// AddMember insère un membre (l'enveloppe de clé est fournie par l'appelant).
func (r *MessageRepository) AddMember(ctx context.Context, m *models.Member) error {
	_, err := r.members.InsertOne(ctx, m)
	return err
}

// GetMember renvoie l'appartenance d'un utilisateur à une conversation
// (mongo.ErrNoDocuments si l'utilisateur n'est pas membre → sert au contrôle d'accès).
func (r *MessageRepository) GetMember(ctx context.Context, conversationID, userID string) (*models.Member, error) {
	var m models.Member
	filter := bson.M{"conversation_id": conversationID, "user_id": userID}
	if err := r.members.FindOne(ctx, filter).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMembersByUser renvoie les appartenances d'un utilisateur (pour lister ses
// conversations), avec son enveloppe et son rôle pour chacune.
func (r *MessageRepository) ListMembersByUser(ctx context.Context, userID string) ([]models.Member, error) {
	cursor, err := r.members.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	out := []models.Member{}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListMembers renvoie tous les membres d'une conversation (avec rôle).
func (r *MessageRepository) ListMembers(ctx context.Context, conversationID string) ([]models.Member, error) {
	cursor, err := r.members.Find(ctx, bson.M{"conversation_id": conversationID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	out := []models.Member{}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CountMembers renvoie le nombre total de membres d'une conversation.
func (r *MessageRepository) CountMembers(ctx context.Context, conversationID string) (int64, error) {
	return r.members.CountDocuments(ctx, bson.M{"conversation_id": conversationID})
}

// CountWritableMembers compte les membres pouvant écrire (owner/admin/talker) —
// c'est sur eux que porte le cap de 32 (les viewers d'une communauté ne comptent pas).
func (r *MessageRepository) CountWritableMembers(ctx context.Context, conversationID string) (int64, error) {
	filter := bson.M{
		"conversation_id": conversationID,
		"role":            bson.M{"$in": bson.A{models.MemberOwner, models.MemberAdmin, models.MemberTalker}},
	}
	return r.members.CountDocuments(ctx, filter)
}

// SetMemberRole change le rôle d'un membre. mongo.ErrNoDocuments si absent.
func (r *MessageRepository) SetMemberRole(ctx context.Context, conversationID, userID, role string) error {
	res, err := r.members.UpdateOne(ctx,
		bson.M{"conversation_id": conversationID, "user_id": userID},
		bson.M{"$set": bson.M{"role": role}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// ListCommunities renvoie l'annuaire public des communautés, du plus actif au
// plus ancien, paginé. `q` (optionnel) filtre par nom (regex insensible à la
// casse, métacaractères échappés — le nom d'une communauté est en clair).
func (r *MessageRepository) ListCommunities(ctx context.Context, limit, skip int64, q string) ([]models.Conversation, error) {
	filter := bson.M{"type": models.TypeCommunity}
	if q != "" {
		filter["title"] = bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.conversations.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	out := []models.Conversation{}
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveMember retire un membre. Renvoie true si une ligne a été supprimée.
func (r *MessageRepository) RemoveMember(ctx context.Context, conversationID, userID string) (bool, error) {
	res, err := r.members.DeleteOne(ctx, bson.M{"conversation_id": conversationID, "user_id": userID})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// DeleteMembersByConversation purge l'appartenance d'une conversation (cascade).
func (r *MessageRepository) DeleteMembersByConversation(ctx context.Context, conversationID string) error {
	_, err := r.members.DeleteMany(ctx, bson.M{"conversation_id": conversationID})
	return err
}

// DeleteMessagesByConversation purge les messages d'une conversation (cascade).
func (r *MessageRepository) DeleteMessagesByConversation(ctx context.Context, conversationID string) error {
	_, err := r.messages.DeleteMany(ctx, bson.M{"conversation_id": conversationID})
	return err
}

// MemberIDs renvoie les ids des membres d'une conversation (pour la diffusion
// temps réel : on pousse le message à chaque membre connecté).
func (r *MessageRepository) MemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	cursor, err := r.members.Find(ctx, bson.M{"conversation_id": conversationID},
		options.Find().SetProjection(bson.M{"user_id": 1}))
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
		if v, ok := d["user_id"].(string); ok {
			ids = append(ids, v)
		}
	}
	return ids, nil
}

// --- Messages ----------------------------------------------------------------

// InsertMessage insère un message chiffré et renseigne msg.ID.
func (r *MessageRepository) InsertMessage(ctx context.Context, msg *models.Message) error {
	res, err := r.messages.InsertOne(ctx, msg)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		msg.ID = oid
	}
	return nil
}

// ListMessages renvoie une page de messages d'une conversation, du plus ancien
// au plus récent (ordre d'affichage). `before` (ObjectID, optionnel) pagine vers
// l'arrière : on renvoie les messages ANTÉRIEURS à ce curseur (scroll vers le
// haut). Sans curseur : la page la plus récente.
func (r *MessageRepository) ListMessages(ctx context.Context, conversationID string, limit int64, before *bson.ObjectID) ([]models.Message, error) {
	filter := bson.M{"conversation_id": conversationID}
	if before != nil {
		filter["_id"] = bson.M{"$lt": *before}
	}
	// On prend les N plus récents (tri desc), puis on renverse pour l'affichage.
	opts := options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(limit)

	cursor, err := r.messages.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	desc := []models.Message{}
	if err := cursor.All(ctx, &desc); err != nil {
		return nil, err
	}

	// Renverse en place (asc) : du plus ancien au plus récent.
	for i, j := 0, len(desc)-1; i < j; i, j = i+1, j-1 {
		desc[i], desc[j] = desc[j], desc[i]
	}
	return desc, nil
}
