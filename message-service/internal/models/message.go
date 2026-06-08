// Package models contient les structures de données du message-service.
//
// Modèle E2EE : le serveur ne voit jamais le clair. Il ne stocke que des
// ciphertexts (messages) et des « enveloppes » de clé (la clé de contenu d'une
// conversation, chiffrée pour chaque membre avec sa clé publique). Le serveur
// contrôle l'appartenance et les rôles, pas le contenu.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Rôles globaux (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// Types de conversation.
const (
	TypeDM        = "dm"
	TypeGroup     = "group"
	TypeCommunity = "community"
)

// Rôles d'un membre AU SEIN d'une conversation. owner/admin/talker peuvent
// écrire ; viewer est en lecture seule (communautés).
const (
	MemberOwner  = "owner"
	MemberAdmin  = "admin"
	MemberTalker = "talker"
	MemberViewer = "viewer"
)

// UserKey — clé publique d'identité d'un utilisateur (X25519, base64). La clé
// privée correspondante ne quitte JAMAIS le navigateur (jamais stockée ici).
type UserKey struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string        `bson:"user_id" json:"user_id"`
	PublicKey string        `bson:"public_key" json:"public_key"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

// Conversation — document de la collection `conversations`.
type Conversation struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Type      string        `bson:"type" json:"type"`
	MemberIDs []string      `bson:"member_ids" json:"member_ids"`
	DMKey     string        `bson:"dm_key,omitempty" json:"-"`
	Title     string        `bson:"title,omitempty" json:"title,omitempty"`
	CreatedBy string        `bson:"created_by" json:"created_by"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

// Member — appartenance d'un utilisateur à une conversation : son rôle + son
// enveloppe de clé (la clé de contenu emballée pour lui).
type Member struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID string        `bson:"conversation_id" json:"conversation_id"`
	UserID         string        `bson:"user_id" json:"user_id"`
	Role           string        `bson:"role" json:"role"`
	KeyEnvelope    string        `bson:"key_envelope,omitempty" json:"key_envelope,omitempty"`
	CreatedAt      time.Time     `bson:"created_at" json:"created_at"`
}

// Message — document de la collection `messages`. UNIQUEMENT du chiffré.
type Message struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID string        `bson:"conversation_id" json:"conversation_id"`
	SenderID       string        `bson:"sender_id" json:"sender_id"`
	Ciphertext     string        `bson:"ciphertext" json:"ciphertext"`
	Nonce          string        `bson:"nonce" json:"nonce"`
	CreatedAt      time.Time     `bson:"created_at" json:"created_at"`
}

// ConversationView — vue renvoyée au client : la conversation + l'enveloppe de
// clé DU DEMANDEUR (les enveloppes des autres ne le concernent pas) + son rôle.
type ConversationView struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	MemberIDs  []string  `json:"member_ids"`
	Title      string    `json:"title,omitempty"`
	MyRole     string    `json:"my_role"`
	MyEnvelope string    `json:"my_envelope"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// --- Corps de requêtes -------------------------------------------------------

// PublishKeyRequest : corps de PUT /messages/keys (publier sa clé publique).
type PublishKeyRequest struct {
	PublicKey string `json:"public_key" binding:"required"`
}

// CreateDMRequest : corps de POST /messages/conversations (DM). Le client génère
// la clé de contenu, l'emballe pour les DEUX membres et fournit les enveloppes
// (map user_id -> enveloppe base64). L'auteur est dérivé du JWT.
type CreateDMRequest struct {
	PeerID    string            `json:"peer_id" binding:"required"`
	Envelopes map[string]string `json:"envelopes" binding:"required"`
}

// SendMessageRequest : corps de POST .../messages. Déjà chiffré côté client.
type SendMessageRequest struct {
	Ciphertext string `json:"ciphertext" binding:"required"`
	Nonce      string `json:"nonce" binding:"required"`
}
