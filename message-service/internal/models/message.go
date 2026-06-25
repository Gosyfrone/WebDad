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

// KeyBackup — sauvegarde CHIFFRÉE de la clé privée d'identité d'un utilisateur,
// permettant de retrouver sa messagerie sur un autre appareil.
//
// Zero-knowledge : la clé privée est emballée côté client par une clé dérivée
// (Argon2id) d'une PHRASE DE PASSE choisie par l'utilisateur. Le serveur ne
// stocke qu'un blob opaque (`salt`, `nonce`, `wrapped_private_key`, paramètres
// KDF) : il ne voit jamais la passphrase ni la clé privée. La propriété
// admin-proof des DM est donc préservée (cf. la doc d'architecture).
type KeyBackup struct {
	ID                bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID            string        `bson:"user_id" json:"user_id"`
	Salt              string        `bson:"salt" json:"salt"`                               // sel Argon2id (base64)
	Nonce             string        `bson:"nonce" json:"nonce"`                             // nonce XChaCha20 de l'emballage (base64)
	WrappedPrivateKey string        `bson:"wrapped_private_key" json:"wrapped_private_key"` // clé privée emballée (base64)
	KDFParams         string        `bson:"kdf_params" json:"kdf_params"`                   // paramètres Argon2id (JSON: m,t,p)
	PublicKey         string        `bson:"public_key" json:"public_key"`                   // clé publique associée (vérif. post-déballage)
	CreatedAt         time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time     `bson:"updated_at" json:"updated_at"`
}

// Conversation — document de la collection `conversations`.
//
// Pour un groupe, le NOM est chiffré avec la clé de contenu du groupe : `Title`
// porte le ciphertext (base64) et `TitleNonce` son nonce. Le serveur ne voit
// donc jamais le nom en clair (le client le déchiffre à l'affichage). DM : pas
// de nom (champs vides).
type Conversation struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Type       string        `bson:"type" json:"type"`
	MemberIDs  []string      `bson:"member_ids" json:"member_ids"`
	DMKey      string        `bson:"dm_key,omitempty" json:"-"`
	Title      string        `bson:"title,omitempty" json:"title,omitempty"`
	TitleNonce string        `bson:"title_nonce,omitempty" json:"title_nonce,omitempty"`
	// ContentKey : clé de contenu (base64) d'une COMMUNAUTÉ, détenue par le
	// serveur (auto-join). `json:"-"` : jamais sérialisée directement — elle
	// n'est exposée qu'aux MEMBRES via ConversationView (cf. buildView).
	ContentKey string    `bson:"content_key,omitempty" json:"-"`
	CreatedBy  string    `bson:"created_by" json:"created_by"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

// Member — appartenance d'un utilisateur à une conversation : son rôle + son
// enveloppe de clé (la clé de contenu emballée pour lui).
//
// État PAR-UTILISATEUR (ne concerne que ce membre, jamais les autres) :
//   - PinnedAt : épinglage de la conversation en tête de SA liste (nil = non
//     épinglée) ;
//   - ClearedAt : suppression « côté user » — masque la conversation de SA liste
//     et coupe SON historique (seuls les messages postérieurs sont renvoyés).
//     La conversation réapparaît si un message plus récent arrive.
type Member struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID string        `bson:"conversation_id" json:"conversation_id"`
	UserID         string        `bson:"user_id" json:"user_id"`
	Role           string        `bson:"role" json:"role"`
	KeyEnvelope    string        `bson:"key_envelope,omitempty" json:"key_envelope,omitempty"`
	PinnedAt       *time.Time    `bson:"pinned_at,omitempty" json:"-"`
	ClearedAt      *time.Time    `bson:"cleared_at,omitempty" json:"-"`
	// LastReadAt : curseur de lecture de CE membre (nil = jamais lu ici). Sert au
	// calcul du non-lu côté serveur (compteur + pastille) — métadonnée, jamais le
	// contenu chiffré.
	LastReadAt *time.Time `bson:"last_read_at,omitempty" json:"-"`
	// LastDeliveredAt : curseur de LIVRAISON de CE membre (nil = jamais reçu ici).
	// « Remis » = le serveur a livré les messages jusqu'ici (poussés sur sa socket
	// ou récupérés via l'historique). Sert aux accusés « remis » des EXPÉDITEURS —
	// métadonnée d'horodatage, jamais le contenu chiffré. Toujours >= LastReadAt
	// (lire implique avoir reçu).
	LastDeliveredAt *time.Time `bson:"last_delivered_at,omitempty" json:"-"`
	// MutedAt : mise en sourdine de la conversation par CE membre (nil = active).
	// En sourdine, la conversation est EXCLUE du badge non-lu (mais reste « non
	// lue » dans la liste). État par-utilisateur.
	MutedAt   *time.Time `bson:"muted_at,omitempty" json:"-"`
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
}

// Message — document de la collection `messages`. UNIQUEMENT du chiffré.
type Message struct {
	ID                 bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID     string        `bson:"conversation_id" json:"conversation_id"`
	SenderID           string        `bson:"sender_id" json:"sender_id"`
	Ciphertext         string        `bson:"ciphertext" json:"ciphertext"`
	Nonce              string        `bson:"nonce" json:"nonce"`
	OriginalCiphertext string        `bson:"original_ciphertext,omitempty" json:"original_ciphertext,omitempty"`
	OriginalNonce      string        `bson:"original_nonce,omitempty" json:"original_nonce,omitempty"`
	CreatedAt          time.Time     `bson:"created_at" json:"created_at"`
	EditedAt           *time.Time    `bson:"edited_at,omitempty" json:"edited_at,omitempty"`
	// DeletedAt : suppression « pour tout le monde » (tombstone). Non nil = le
	// contenu chiffré a été effacé (ciphertext/nonce vidés) ; le client affiche
	// « Message supprimé ». Métadonnée d'horodatage.
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	// DeletedByModeration : true si la suppression a été faite par la modération
	// de plateforme (et non par l'auteur ou un owner/admin de groupe) → le client
	// affiche « supprimé par la modération » au lieu du libellé générique.
	DeletedByModeration bool `bson:"deleted_by_moderation,omitempty" json:"deleted_by_moderation,omitempty"`
}

// ConversationView — vue renvoyée au client : la conversation + l'enveloppe de
// clé DU DEMANDEUR (les enveloppes des autres ne le concernent pas) + son rôle.
type ConversationView struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	MemberIDs  []string `json:"member_ids"`
	Title      string   `json:"title,omitempty"`
	TitleNonce string   `json:"title_nonce,omitempty"`
	MyRole     string   `json:"my_role"`
	MyEnvelope string   `json:"my_envelope"`
	// ContentKey : clé de contenu d'une COMMUNAUTÉ (base64), remise au membre par
	// le serveur. Vide pour DM/groupes (qui utilisent les enveloppes scellées).
	ContentKey string `json:"content_key,omitempty"`
	// PinnedAt : épinglage de la conversation par CE membre (nil/omis = non
	// épinglée). Sert au tri « épinglé d'abord » côté client.
	PinnedAt *time.Time `json:"pinned_at,omitempty"`
	// LastReadAt : curseur de lecture de CE membre (nil/omis = jamais lu). Sert à
	// la pastille « non-lu » de la liste et à l'ancre « Nouveaux messages ».
	LastReadAt *time.Time `json:"last_read_at,omitempty"`
	// Muted : la conversation est-elle en sourdine pour CE membre ? (exclue du
	// badge non-lu app-wide, mais toujours « non lue » dans la liste).
	Muted bool `json:"muted"`
	// MemberReceipts : curseurs « remis » / « ouvert » des AUTRES membres (jamais
	// soi), pour les accusés de réception côté expéditeur. Renseigné pour DM et
	// groupes uniquement (pas les communautés). Métadonnée d'horodatage, jamais le
	// contenu chiffré.
	MemberReceipts []MemberReceipt `json:"member_receipts,omitempty"`
	CreatedBy      string          `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// MemberReceipt — curseurs de livraison/lecture d'un membre, exposés à
// l'expéditeur pour afficher les accusés « remis » (1 coche) / « ouvert » (2
// coches). `nil` = jamais livré / jamais lu.
type MemberReceipt struct {
	UserID      string     `json:"user_id"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}

// MemberView — un membre exposé dans la liste des membres (sans son enveloppe :
// la clé emballée d'un membre ne regarde que lui).
type MemberView struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// CommunityListItem — entrée de l'annuaire public des communautés. Le nom est
// en CLAIR (les communautés sont semi-publiques → découvrables). La clé de
// contenu n'y figure JAMAIS (réservée aux membres).
type CommunityListItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	MemberCount int64     `json:"member_count"`
	IsMember    bool      `json:"is_member"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// --- Corps de requêtes -------------------------------------------------------

// PublishKeyRequest : corps de PUT /messages/keys (publier sa clé publique).
type PublishKeyRequest struct {
	PublicKey string `json:"public_key" binding:"required"`
}

// PutBackupRequest : corps de PUT /messages/keys/backup — enregistre/remplace la
// sauvegarde chiffrée de la clé privée. Tous les champs sont des blobs opaques
// produits côté client ; le serveur ne les interprète pas.
type PutBackupRequest struct {
	Salt              string `json:"salt" binding:"required"`
	Nonce             string `json:"nonce" binding:"required"`
	WrappedPrivateKey string `json:"wrapped_private_key" binding:"required"`
	KDFParams         string `json:"kdf_params" binding:"required"`
	PublicKey         string `json:"public_key" binding:"required"`
}

// BackupStatusResponse : corps de GET /messages/keys/backup/status — indique si
// une sauvegarde existe (pilote l'UI « définir » vs « débloquer »).
type BackupStatusResponse struct {
	Exists bool `json:"exists"`
}

// CreateConversationRequest : corps de POST /messages/conversations.
// `type` discrimine :
//   - "dm" (défaut) : `peer_id` + `envelopes` (les 2 membres) ;
//   - "group"       : `title`+`title_nonce` (nom chiffré) + `envelopes` (TOUS
//     les membres initiaux, créateur inclus). Les clés de `envelopes` = le set
//     de membres. L'auteur (owner) est dérivé du JWT.
//
// Le client génère la clé de contenu, l'emballe par membre et fournit les
// enveloppes (map user_id -> enveloppe base64) ; le serveur ne voit pas la clé.
//   - "community"    : `title` (nom en CLAIR) + `content_key` (la clé de contenu
//     en base64, confiée au serveur pour l'auto-join). Pas d'`envelopes`.
type CreateConversationRequest struct {
	Type       string            `json:"type"`
	PeerID     string            `json:"peer_id"`
	Title      string            `json:"title"`
	TitleNonce string            `json:"title_nonce"`
	ContentKey string            `json:"content_key"`
	Envelopes  map[string]string `json:"envelopes"`
}

// AddMemberRequest : corps de POST .../:id/members (inviter). L'invitant emballe
// la clé de contenu (qu'il détient) pour la clé publique de l'invité.
type AddMemberRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Envelope string `json:"envelope" binding:"required"`
}

// InviteMemberRequest : corps de POST .../:id/invite (inviter dans une
// communauté). Pas d'envelope : le serveur détient la clé de contenu et la
// remet à l'invité (cf. buildView). Seul l'id de la cible est requis.
type InviteMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// UpdateGroupRequest : corps de PATCH .../:id (renommer). Groupe : nom
// re-chiffré (title+title_nonce). Communauté : nom en clair (title seul,
// title_nonce vide).
type UpdateGroupRequest struct {
	Title      string `json:"title" binding:"required"`
	TitleNonce string `json:"title_nonce"`
}

// SetMemberRoleRequest : corps de PATCH .../:id/members/:userId — promouvoir /
// rétrograder un membre d'une communauté (owner uniquement). Rôle attendu :
// "talker" (peut écrire) ou "viewer" (lecture seule).
type SetMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// SendMessageRequest : corps de POST .../messages. Déjà chiffré côté client.
//
// MentionedMemberIDs : ids des membres mentionnés (@handle) DANS ce message,
// résolus côté client (le serveur ne lit pas le ciphertext). Métadonnée
// d'appartenance uniquement (jamais de texte) → sert à notifier les membres
// mentionnés sans casser l'E2EE. Les ids non-membres sont ignorés côté serveur.
type SendMessageRequest struct {
	Ciphertext         string   `json:"ciphertext" binding:"required"`
	Nonce              string   `json:"nonce" binding:"required"`
	MentionedMemberIDs []string `json:"mentioned_member_ids"`
}

// EditMessageRequest : corps de PATCH .../messages/:messageId. Le client
// chiffre déjà la nouvelle version ; le serveur conserve l'ancienne version
// chiffrée sans jamais lire le clair.
type EditMessageRequest struct {
	Ciphertext         string   `json:"ciphertext" binding:"required"`
	Nonce              string   `json:"nonce" binding:"required"`
	MentionedMemberIDs []string `json:"mentioned_member_ids"`
}
