// Package models contient les structures de données du notification-service.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Types de notification (alignés sur l'enum du validateur Mongo).
const (
	TypeLike                       = "like"
	TypeComment                    = "comment"
	TypeReply                      = "reply"
	TypeMention                    = "mention"
	TypeRepost                     = "repost"
	TypeQuote                      = "quote"
	TypeFollow                     = "follow"
	TypeFollowRequest              = "follow_request"
	TypeFollowRequestAccepted      = "follow_request_accepted"
	TypeFollowRequestAcceptConfirm = "follow_request_accept_confirm"
	// TypeMessageMention : mention (@handle) DANS UN MESSAGE (DM / groupe /
	// communauté). Émise par message-service avec les `recipient_id` déjà
	// résolus (le serveur de messagerie connaît ses membres) ; agrégée par
	// conversation (`message_mention:<conversation_id>`). Navigue vers la
	// conversation, pas vers un post.
	TypeMessageMention = "message_mention"
)

// Types d'événement reçus des services émetteurs (au-delà des types de
// notification, `post_deleted` déclenche une purge en cascade — il ne crée pas
// de notification).
const (
	EventPostDeleted           = "post_deleted"
	EventFollowRequestRejected = "follow_request_rejected"
)

// Notification — document de la collection `notifications`. Une notification est
// un GROUPE agrégé : `group_key` est unique par destinataire (cf. init.go), et
// `count` est le nombre d'événements agrégés. L'affichage « X et N autres » se
// reconstruit à partir de `last_actor_id` + `count` ; les noms/avatars sont
// résolus côté front (cache auteur), comme pour les posts.
//
// `count` est un `int32` (le validateur Mongo le déclare en `int`) maintenu par
// `$inc`.
type Notification struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RecipientID string        `bson:"recipient_id" json:"recipient_id"`
	GroupKey    string        `bson:"group_key" json:"group_key"`
	Type        string        `bson:"type" json:"type"`
	PostID      string        `bson:"post_id,omitempty" json:"post_id,omitempty"`
	CommentID   string        `bson:"comment_id,omitempty" json:"comment_id,omitempty"`
	// ConversationID : cible d'une mention en message (navigation vers la conv).
	ConversationID string    `bson:"conversation_id,omitempty" json:"conversation_id,omitempty"`
	LastActorID    string    `bson:"last_actor_id" json:"last_actor_id"`
	Count          int32     `bson:"count" json:"count"`
	IsRead         bool      `bson:"is_read" json:"is_read"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at" json:"updated_at"`
}

// Event — charge utile de POST /internal/events, émise par post-service ou
// user-service après une action (like, commentaire, follow...) ou une suppression.
//
// Selon le type :
//   - like/comment/reply : `recipient_id` est fourni directement (post-service
//     connaît l'auteur du post / du commentaire visé) ;
//   - mention            : `mention_handles` est résolu par CE service via
//     user-service (seul détenteur des handles) → un destinataire par handle ;
//   - follow             : `recipient_id` est l'utilisateur suivi ;
//   - follow_request_accepted : `recipient_id` est le demandeur accepté ;
//   - follow_request_accept_confirm : `recipient_id` est le propriétaire qui
//     vient d'accepter la demande ;
//   - message_mention    : `recipient_id` + `conversation_id` sont fournis
//     directement (message-service connaît ses membres, et le contenu reste
//     chiffré → la résolution du handle se fait côté client/messagerie) ;
//   - post_deleted       : purge toutes les notifications du `post_id`.
//
// `retract = true` défait une action (unlike, suppression de commentaire) :
// décrémente le compteur du groupe, supprime la notification si le compteur
// tombe à zéro.
type Event struct {
	Type           string   `json:"type" binding:"required"`
	ActorID        string   `json:"actor_id" binding:"required"`
	RecipientID    string   `json:"recipient_id"`
	PostID         string   `json:"post_id"`
	CommentID      string   `json:"comment_id"`
	ConversationID string   `json:"conversation_id"`
	MentionHandles []string `json:"mention_handles"`
	Retract        bool     `json:"retract"`
}
