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
	keyBackups    *mongo.Collection
	conversations *mongo.Collection
	members       *mongo.Collection
	messages      *mongo.Collection
}

func NewMessageRepository(db *mongo.Database) *MessageRepository {
	return &MessageRepository{
		keys:          db.Collection("user_keys"),
		keyBackups:    db.Collection("key_backups"),
		conversations: db.Collection("conversations"),
		members:       db.Collection("members"),
		messages:      db.Collection("messages"),
	}
}

// PurgeUser efface DÉFINITIVEMENT la participation d'un utilisateur à la
// messagerie (effacement RGPD) : sa clé publique, ses appartenances aux
// conversations et les messages qu'il a envoyés. Les conversations partagées
// (DM/groupes) subsistent pour les autres membres ; un groupe/communauté dont
// il était propriétaire devient orphelin (compromis assumé, cf. CLAUDE.md §6).
func (r *MessageRepository) PurgeUser(ctx context.Context, userID string) error {
	if _, err := r.keys.DeleteMany(ctx, bson.M{"user_id": userID}); err != nil {
		return err
	}
	if _, err := r.keyBackups.DeleteMany(ctx, bson.M{"user_id": userID}); err != nil {
		return err
	}
	if _, err := r.members.DeleteMany(ctx, bson.M{"user_id": userID}); err != nil {
		return err
	}
	_, err := r.messages.DeleteMany(ctx, bson.M{"sender_id": userID})
	return err
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

// --- Sauvegarde chiffrée de la clé privée -----------------------------------

// UpsertBackup enregistre/remplace la sauvegarde chiffrée d'un utilisateur
// (idempotent). Les champs sont des blobs opaques (cf. models.KeyBackup).
func (r *MessageRepository) UpsertBackup(ctx context.Context, userID string, b models.PutBackupRequest) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"salt":                b.Salt,
			"nonce":               b.Nonce,
			"wrapped_private_key": b.WrappedPrivateKey,
			"kdf_params":          b.KDFParams,
			"public_key":          b.PublicKey,
			"updated_at":          now,
		},
		"$setOnInsert": bson.M{"user_id": userID, "created_at": now},
	}
	_, err := r.keyBackups.UpdateOne(ctx, bson.M{"user_id": userID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

// GetBackup renvoie la sauvegarde chiffrée d'un utilisateur (mongo.ErrNoDocuments si absente).
func (r *MessageRepository) GetBackup(ctx context.Context, userID string) (*models.KeyBackup, error) {
	var b models.KeyBackup
	if err := r.keyBackups.FindOne(ctx, bson.M{"user_id": userID}).Decode(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

// BackupExists indique si une sauvegarde chiffrée existe pour cet utilisateur.
func (r *MessageRepository) BackupExists(ctx context.Context, userID string) (bool, error) {
	n, err := r.keyBackups.CountDocuments(ctx, bson.M{"user_id": userID}, options.Count().SetLimit(1))
	return n > 0, err
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

// SetMemberPinned (dés)épingle une conversation pour un membre. `at == nil`
// retire l'épinglage (`$unset`), sinon le pose. mongo.ErrNoDocuments si absent.
func (r *MessageRepository) SetMemberPinned(ctx context.Context, conversationID, userID string, at *time.Time) error {
	var update bson.M
	if at == nil {
		update = bson.M{"$unset": bson.M{"pinned_at": ""}}
	} else {
		update = bson.M{"$set": bson.M{"pinned_at": *at}}
	}
	res, err := r.members.UpdateOne(ctx,
		bson.M{"conversation_id": conversationID, "user_id": userID}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// SetMemberMuted met en sourdine (ou réactive) une conversation pour un membre.
// `at == nil` réactive (`$unset`), sinon pose la sourdine. mongo.ErrNoDocuments
// si absent.
func (r *MessageRepository) SetMemberMuted(ctx context.Context, conversationID, userID string, at *time.Time) error {
	var update bson.M
	if at == nil {
		update = bson.M{"$unset": bson.M{"muted_at": ""}}
	} else {
		update = bson.M{"$set": bson.M{"muted_at": *at}}
	}
	res, err := r.members.UpdateOne(ctx,
		bson.M{"conversation_id": conversationID, "user_id": userID}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// SetMemberCleared pose la date de suppression côté user (masque + coupe
// l'historique). mongo.ErrNoDocuments si le membre n'existe pas.
func (r *MessageRepository) SetMemberCleared(ctx context.Context, conversationID, userID string, at time.Time) error {
	res, err := r.members.UpdateOne(ctx,
		bson.M{"conversation_id": conversationID, "user_id": userID},
		bson.M{"$set": bson.M{"cleared_at": at}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// HasMessagesAfter indique s'il existe au moins un message postérieur à `after`
// (sert à décider si une conversation « supprimée côté user » doit réapparaître).
func (r *MessageRepository) HasMessagesAfter(ctx context.Context, conversationID string, after time.Time) (bool, error) {
	filter := bson.M{"conversation_id": conversationID, "created_at": bson.M{"$gt": after}}
	n, err := r.messages.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	return n > 0, err
}

// SetMemberRead avance le curseur de lecture d'un membre (`last_read_at`) ET sa
// livraison (`last_delivered_at`) : lire implique avoir reçu, donc remis >= ouvert.
// mongo.ErrNoDocuments si le membre n'existe pas.
func (r *MessageRepository) SetMemberRead(ctx context.Context, conversationID, userID string, at time.Time) error {
	res, err := r.members.UpdateOne(ctx,
		bson.M{"conversation_id": conversationID, "user_id": userID},
		bson.M{"$set": bson.M{"last_read_at": at, "last_delivered_at": at}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// SetMemberDelivered avance le curseur de LIVRAISON d'un membre
// (`last_delivered_at`) — « remis ». mongo.ErrNoDocuments si le membre n'existe pas.
func (r *MessageRepository) SetMemberDelivered(ctx context.Context, conversationID, userID string, at time.Time) error {
	res, err := r.members.UpdateOne(ctx,
		bson.M{"conversation_id": conversationID, "user_id": userID},
		bson.M{"$set": bson.M{"last_delivered_at": at}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// CountUnreadConversations renvoie le NOMBRE de conversations de `userID` ayant au
// moins un message non lu, SANS jamais lire le contenu chiffré (compare seulement
// `created_at`/`sender_id`, des métadonnées en clair) → l'E2EE reste intact.
//
// Un message compte comme non lu s'il est postérieur au curseur de lecture du
// membre (`last_read_at`) ET à sa suppression côté user (`cleared_at`) ET qu'il
// n'a pas été envoyé par l'utilisateur lui-même. Le curseur effectif est le max
// des deux dates (epoch si aucune). Une seule requête (`$lookup` borné à 1
// message par conversation grâce à l'index `conversation_id`).
func (r *MessageRepository) CountUnreadConversations(ctx context.Context, userID string) (int, error) {
	epoch := time.Unix(0, 0)
	pipeline := mongo.Pipeline{
		// On exclut d'emblée les conversations en sourdine (muted_at posé) : elles
		// n'alimentent pas le badge. `muted_at: null` matche aussi le champ absent.
		bson.D{{Key: "$match", Value: bson.M{"user_id": userID, "muted_at": nil}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"cut": bson.M{"$max": bson.A{
				bson.M{"$ifNull": bson.A{"$last_read_at", epoch}},
				bson.M{"$ifNull": bson.A{"$cleared_at", epoch}},
			}},
		}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from": "messages",
			"let":  bson.M{"conv": "$conversation_id", "cut": "$cut"},
			"pipeline": mongo.Pipeline{
				bson.D{{Key: "$match", Value: bson.M{"$expr": bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$conversation_id", "$$conv"}},
					bson.M{"$ne": bson.A{"$sender_id", userID}},
					bson.M{"$gt": bson.A{"$created_at", "$$cut"}},
				}}}}},
				bson.D{{Key: "$limit", Value: 1}},
				bson.D{{Key: "$project", Value: bson.M{"_id": 1}}},
			},
			"as": "unread",
		}}},
		bson.D{{Key: "$match", Value: bson.M{"unread": bson.M{"$ne": bson.A{}}}}},
		bson.D{{Key: "$count", Value: "count"}},
	}

	cursor, err := r.members.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []struct {
		Count int `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Count, nil
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

// GetMessage renvoie un message par id dans une conversation donnée.
func (r *MessageRepository) GetMessage(ctx context.Context, conversationID string, id bson.ObjectID) (*models.Message, error) {
	var msg models.Message
	if err := r.messages.FindOne(ctx, bson.M{"_id": id, "conversation_id": conversationID}).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// UpdateMessageCiphertext remplace la version courante par une nouvelle version
// chiffrée, en conservant la version originale chiffrée au premier edit.
func (r *MessageRepository) UpdateMessageCiphertext(ctx context.Context, msg *models.Message, ciphertext, nonce string, editedAt time.Time) (*models.Message, error) {
	originalCiphertext := msg.OriginalCiphertext
	originalNonce := msg.OriginalNonce
	if originalCiphertext == "" || originalNonce == "" {
		originalCiphertext = msg.Ciphertext
		originalNonce = msg.Nonce
	}

	update := bson.M{"$set": bson.M{
		"ciphertext":          ciphertext,
		"nonce":               nonce,
		"original_ciphertext": originalCiphertext,
		"original_nonce":      originalNonce,
		"edited_at":           editedAt,
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated models.Message
	if err := r.messages.FindOneAndUpdate(ctx, bson.M{"_id": msg.ID, "conversation_id": msg.ConversationID}, update, opts).Decode(&updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// SoftDeleteMessage supprime « pour tout le monde » (tombstone) : pose
// `deleted_at`, VIDE le contenu chiffré (ciphertext/nonce) et retire toute
// version originale conservée. Renvoie le message tombstoné (pour la diffusion).
func (r *MessageRepository) SoftDeleteMessage(ctx context.Context, conversationID string, id bson.ObjectID, at time.Time) (*models.Message, error) {
	update := bson.M{
		"$set":   bson.M{"ciphertext": "", "nonce": "", "deleted_at": at},
		"$unset": bson.M{"original_ciphertext": "", "original_nonce": "", "edited_at": ""},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated models.Message
	if err := r.messages.FindOneAndUpdate(ctx, bson.M{"_id": id, "conversation_id": conversationID}, update, opts).Decode(&updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// ListMessages renvoie une page de messages d'une conversation, du plus ancien
// au plus récent (ordre d'affichage). `before` (ObjectID, optionnel) pagine vers
// l'arrière : on renvoie les messages ANTÉRIEURS à ce curseur (scroll vers le
// haut). Sans curseur : la page la plus récente. `after` (optionnel) coupe
// l'historique : on ignore les messages antérieurs ou égaux (suppression côté
// user → `cleared_at`).
func (r *MessageRepository) ListMessages(ctx context.Context, conversationID string, limit int64, before *bson.ObjectID, after *time.Time) ([]models.Message, error) {
	filter := bson.M{"conversation_id": conversationID}
	if before != nil {
		filter["_id"] = bson.M{"$lt": *before}
	}
	if after != nil {
		filter["created_at"] = bson.M{"$gt": *after}
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
