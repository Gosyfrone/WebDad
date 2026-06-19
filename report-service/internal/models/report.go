// Package models contient les structures de données du report-service.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Catégories de ticket (alignées sur l'enum du validateur Mongo).
const (
	CategoryModeration = "moderation" // signalement de contenu/compte → onglet Modération
	CategoryBug        = "bug"        // rapport technique → onglet Administration
)

// Types d'entité signalée.
const (
	EntityPost         = "post"
	EntityMessage      = "message"
	EntityGroupMessage = "group_message"
	EntityProfile      = "profile"
	EntityApp          = "app" // bug applicatif (pas d'entité précise)
)

// États du cycle de vie d'un ticket.
const (
	StatusOpen     = "open"
	StatusClosed   = "closed"
	StatusReopened = "reopened"
	// StatusApproved : état TERMINAL. Un modérateur a jugé l'entité conforme
	// (« ne doit pas être signalée »). Conséquences : l'entité est démasquée si
	// elle avait été auto-masquée, et tout NOUVEAU signalement est refusé
	// (verrou définitif). Distinct de `closed`, qui peut se rouvrir au seuil.
	StatusApproved = "approved"
)

// Types d'action posée par un modérateur sur un ticket.
const (
	ActionReply          = "reply"           // réponse interne
	ActionStatusChange   = "status_change"   // changement de statut
	ActionTransfer       = "transfer"        // transfert bug → modération (admin)
	ActionAutoReopen     = "auto_reopen"     // réouverture automatique (seuil de re-signalements)
	ActionContentRemoved = "content_removed" // contenu signalé retiré par la modération
	ActionAutoHidden     = "auto_hidden"     // entité auto-masquée (seuil de signalements atteint)
	ActionApproved       = "approved"        // entité jugée conforme par la modération (terminal)
	ActionWarned         = "warned"          // auteur de l'entité averti par la modération (depuis le ticket)
)

// Motifs de signalement (enum fermé : `reason_tags.<reason>` est une clé Mongo,
// donc bornée pour éviter l'injection de champs arbitraires). Les libellés
// affichés sont traduits côté front (i18n).
const (
	ReasonInappropriate = "inappropriate" // comportement inapproprié
	ReasonOffensive     = "offensive"     // langage offensant
	ReasonBug           = "bug"           // bug technique
	ReasonSpam          = "spam"          // spam / publicité
	ReasonOther         = "other"         // autre
)

// Bornes de longueur du texte libre, par catégorie (exigence fonctionnelle).
const (
	MaxModerationText = 255
	MaxBugText        = 500
	MaxWarningMessage = 500
)

// Report — entrée enfant : un signalement individuel empilé dans un ticket
// parent. Le fil de discussion affiche, par entrée, l'utilisateur, son motif et
// son texte.
type Report struct {
	ReporterID string `bson:"reporter_id" json:"reporter_id"`
	Reason     string `bson:"reason" json:"reason"`
	Text       string `bson:"text,omitempty" json:"text,omitempty"`
	// DisclosedContent : copie EN CLAIR d'un contenu chiffré (message privé/groupe)
	// que le signaleur — qui en est destinataire — choisit de divulguer pour son
	// signalement. Le serveur reste aveugle (E2EE) : seul un participant peut
	// révéler CE message précis, volontairement, au moment du signalement.
	DisclosedContent string    `bson:"disclosed_content,omitempty" json:"disclosed_content,omitempty"`
	AttachmentID     string    `bson:"attachment_id,omitempty" json:"attachment_id,omitempty"`
	CreatedAt        time.Time `bson:"created_at" json:"created_at"`
}

// Action — trace d'une action de modération sur un ticket. Enregistre TOUJOURS
// l'id du modérateur responsable (exigence fonctionnelle).
type Action struct {
	ModeratorID string    `bson:"moderator_id" json:"moderator_id"`
	Type        string    `bson:"type" json:"type"`
	Text        string    `bson:"text,omitempty" json:"text,omitempty"`
	Status      string    `bson:"status,omitempty" json:"status,omitempty"` // pour status_change
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}

// Ticket — document de la collection `tickets`. Un ticket parent agrège tous
// les signalements d'une même entité (modération) ; un bug est un ticket
// autonome. `report_count` est dénormalisé (int32, maintenu par $inc) et
// `reason_tags` compte les motifs pour exposer les étiquettes récurrentes.
type Ticket struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Category   string        `bson:"category" json:"category"`
	EntityType string        `bson:"entity_type" json:"entity_type"`
	EntityID   string        `bson:"entity_id,omitempty" json:"entity_id,omitempty"`
	// EntityOwnerID : id de l'utilisateur PROPRIÉTAIRE de l'entité signalée
	// (auteur du post, du message, ou l'utilisateur du profil), capturé au
	// signalement. Permet d'avertir l'auteur sans dépendre d'une relecture de
	// l'entité (post masqué/supprimé → l'id reste disponible).
	EntityOwnerID string `bson:"entity_owner_id,omitempty" json:"entity_owner_id,omitempty"`
	Status        string `bson:"status" json:"status"`
	ReportCount   int32  `bson:"report_count" json:"report_count"`
	// ReportsSinceClosed : nombre de signalements reçus depuis le dernier
	// changement de statut. Sert au seuil de réouverture automatique : un ticket
	// CLÔTURÉ se rouvre seul une fois ce compteur ≥ seuil (cf. ReopenThreshold).
	// Remis à zéro à chaque changement de statut.
	ReportsSinceClosed int32            `bson:"reports_since_closed,omitempty" json:"reports_since_closed,omitempty"`
	ReasonTags         map[string]int32 `bson:"reason_tags,omitempty" json:"reason_tags,omitempty"`
	Reports            []Report         `bson:"reports,omitempty" json:"reports,omitempty"`
	Actions            []Action         `bson:"actions,omitempty" json:"actions,omitempty"`
	LastReportedAt     time.Time        `bson:"last_reported_at" json:"last_reported_at"`
	CreatedAt          time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time        `bson:"updated_at" json:"updated_at"`
}

// Identifiant du document singleton de configuration (collection `settings`).
const SettingsSingletonID = "global"

// DefaultAutoHideThreshold : seuil par défaut d'auto-masquage d'un post. Un post
// de modération atteignant ce nombre de signalements est masqué en attendant la
// décision d'un modérateur. Réglable par l'administrateur (cf. Settings).
const DefaultAutoHideThreshold int32 = 5

// Settings — document SINGLETON (`_id="global"`) de configuration runtime de la
// modération, éditable par l'administrateur. AutoHideThreshold = nombre de
// signalements à partir duquel un post est auto-masqué (0 = auto-masquage
// désactivé).
type Settings struct {
	ID                string `bson:"_id" json:"-"`
	AutoHideThreshold int32  `bson:"auto_hide_threshold" json:"auto_hide_threshold"`
}

// Warning — avertissement asynchrone émis par un modérateur vers un utilisateur.
// Intercepté à la prochaine requête/connexion de la cible (modale bloquante),
// puis acquitté (`acknowledged=true`).
type Warning struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	TargetUserID   string        `bson:"target_user_id" json:"target_user_id"`
	TicketID       string        `bson:"ticket_id,omitempty" json:"ticket_id,omitempty"`
	Message        string        `bson:"message" json:"message"`
	IssuedBy       string        `bson:"issued_by" json:"issued_by"`
	Acknowledged   bool          `bson:"acknowledged" json:"acknowledged"`
	CreatedAt      time.Time     `bson:"created_at" json:"created_at"`
	AcknowledgedAt *time.Time    `bson:"acknowledged_at,omitempty" json:"acknowledged_at,omitempty"`
}
