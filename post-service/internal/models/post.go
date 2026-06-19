// Package models contient les structures de données du service post.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Rôles valides (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// Post — document de la collection `posts`. Champs en snake_case, alignés
// sur le validateur $jsonSchema (cf. internal/database/init.go).
//
// LikesCount / CommentsCount sont des compteurs dénormalisés, maintenus par
// `$inc` au fil des likes/commentaires (le validateur Mongo les déclare en
// `int` → on utilise `int32`, sinon un `int64` casserait la validation).
type Post struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthorID    string        `bson:"author_id" json:"author_id"`
	Content     string        `bson:"content" json:"content"`
	Hashtags    []string      `bson:"hashtags,omitempty" json:"hashtags,omitempty"`
	Media       []MediaRef    `bson:"media,omitempty" json:"media,omitempty"`
	Poll        *Poll         `bson:"poll,omitempty" json:"poll,omitempty"`
	QuotePostID string        `bson:"quote_post_id,omitempty" json:"quote_post_id,omitempty"`
	// ReplyAudience : qui peut commenter/répondre au post — `everyone` (défaut)
	// ou `followers` (auteur + abonnés ; mods/admins toujours autorisés). Choisi
	// à la création, non éditable ensuite (parité avec l'audience d'un sondage).
	// `omitempty` ⇒ jamais de `""` persisté ; absent sur un vieux doc = `everyone`
	// (cf. ReplyAudienceOf), donc aucune migration nécessaire.
	ReplyAudience string     `bson:"reply_audience,omitempty" json:"reply_audience"`
	LikesCount    int32      `bson:"likes_count" json:"likes_count"`
	CommentsCount int32      `bson:"comments_count" json:"comments_count"`
	RepostsCount  int32      `bson:"reposts_count" json:"reposts_count"`
	PinnedAt      *time.Time `bson:"pinned_at,omitempty" json:"pinned_at,omitempty"`
	// Nsfw : post marqué « contenu sensible ». Posé par l'AUTEUR à la création
	// (bouton NSFW du composer) ou par un modérateur/admin après coup (PATCH
	// /posts/:id/nsfw). Le post reste DÉLIVRÉ tel quel : c'est le front qui le
	// floute si le lecteur n'a pas le droit/la préférence de voir le NSFW
	// (nsfw_visible calculé serveur côté profil). `omitempty` ⇒ jamais de `false`
	// persisté ; absent sur un vieux doc = non-NSFW (aucune migration nécessaire).
	Nsfw bool `bson:"nsfw,omitempty" json:"nsfw,omitempty"`
	// NsfwBy / NsfwAt : qui a marqué le post NSFW et quand (métadonnée de
	// modération, non exposée au front).
	NsfwBy string     `bson:"nsfw_by,omitempty" json:"-"`
	NsfwAt *time.Time `bson:"nsfw_at,omitempty" json:"-"`
	// Suppression « douce » par la modération : un modérateur/admin qui retire le
	// post d'autrui le MASQUE (is_hidden) au lieu de l'effacer → il sort des fils
	// publics mais reste restaurable depuis la corbeille de modération. HiddenAt
	// sert aussi de point de départ à la purge RGPD (5 ans). L'auteur qui supprime
	// SON post déclenche, lui, une vraie suppression (hard), pas un masquage.
	IsHidden bool       `bson:"is_hidden,omitempty" json:"is_hidden,omitempty"`
	HiddenBy string     `bson:"hidden_by,omitempty" json:"hidden_by,omitempty"`
	HiddenAt *time.Time `bson:"hidden_at,omitempty" json:"hidden_at,omitempty"`
	// AutoHidden : masquage AUTOMATIQUE déclenché par le report-service quand le
	// post dépasse le seuil de signalements (en attente d'une décision de
	// modération). Distinct de is_hidden (retrait manuel → corbeille) : il sort
	// des fils publics mais N'apparaît PAS dans la corbeille de modération ; un
	// modérateur le rétablit (validation « conforme ») ou le retire (is_hidden).
	// `omitempty` ⇒ jamais de `false` persité ; absent sur un vieux doc = visible
	// (pas de migration de données nécessaire).
	AutoHidden bool `bson:"auto_hidden,omitempty" json:"auto_hidden,omitempty"`
	// PurgeWarnedAt : date d'envoi du préavis de purge RGPD (évite de re-notifier).
	PurgeWarnedAt *time.Time `bson:"purge_warned_at,omitempty" json:"-"`
	// PurgeAt : date prévue de purge définitive (transient = hidden_at + rétention),
	// calculée à la lecture de la corbeille pour l'affichage front. NON stockée.
	PurgeAt      *time.Time `bson:"-" json:"purge_at,omitempty"`
	RepostedByID string     `bson:"-" json:"reposted_by_id,omitempty"`
	RepostedAt   *time.Time `bson:"-" json:"reposted_at,omitempty"`
	// CanReply : transient, hydraté par viewer — `true` si le lecteur courant peut
	// commenter/répondre (toujours `true` si reply_audience=everyone ; sinon auteur,
	// abonné, ou mod/admin). Sert à (dés)activer le composer côté front. Non stocké.
	CanReply  bool      `bson:"-" json:"can_reply"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

const (
	PollAudienceEveryone  = "everyone"
	PollAudienceFollowers = "followers"
)

// Audience des réponses à un post (qui peut commenter/répondre).
const (
	ReplyAudienceEveryone  = "everyone"
	ReplyAudienceFollowers = "followers"
)

// ReplyAudienceOf normalise l'audience des réponses d'un post : une valeur vide
// ou absente (vieux documents antérieurs au champ) vaut `everyone`.
func ReplyAudienceOf(p *Post) string {
	if p == nil || p.ReplyAudience == "" {
		return ReplyAudienceEveryone
	}
	return p.ReplyAudience
}

// Poll — sondage embarqué dans un post. Les votes sont stockés à part dans
// `poll_votes` pour garantir "un vote par utilisateur", tandis que les
// compteurs dénormalisés vivent ici pour afficher le résultat rapidement.
type Poll struct {
	Choices         []PollChoice `bson:"choices" json:"choices"`
	EndsAt          time.Time    `bson:"ends_at" json:"ends_at"`
	ClosedAt        *time.Time   `bson:"closed_at,omitempty" json:"closed_at,omitempty"`
	Audience        string       `bson:"audience" json:"audience"`
	TotalVotes      int32        `bson:"total_votes" json:"total_votes"`
	VotedChoiceID   string       `bson:"-" json:"voted_choice_id,omitempty"`
	WinnerChoiceIDs []string     `bson:"-" json:"winner_choice_ids,omitempty"`
	CanViewResults  bool         `bson:"-" json:"can_view_results"`
	CanClose        bool         `bson:"-" json:"can_close"`
}

type PollChoice struct {
	ID         string `bson:"id" json:"id"`
	Label      string `bson:"label" json:"label"`
	VotesCount int32  `bson:"votes_count" json:"votes_count"`
	// ImageURL (optionnel) : illustration du choix, chemin relatif `/media/<id>`
	// servi par le media-service via la gateway (le post-service reste agnostique
	// du contenu). Vide = choix texte seul (les choix mixtes sont autorisés).
	ImageURL string `bson:"image_url,omitempty" json:"image_url,omitempty"`
}

type PollVote struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID    string        `bson:"post_id" json:"post_id"`
	UserID    string        `bson:"user_id" json:"user_id"`
	ChoiceID  string        `bson:"choice_id" json:"choice_id"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// Repost — document de la collection `reposts`. Index unique `post_id+user_id`
// pour que l'action soit idempotente : un utilisateur ne peut repost qu'une
// fois le même post.
type Repost struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID    string        `bson:"post_id" json:"post_id"`
	UserID    string        `bson:"user_id" json:"user_id"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// HashtagTrend — compteur agrégé d'un hashtag dans les posts visibles.
type HashtagTrend struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// PostStat — compteurs dénormalisés d'un post (likes/commentaires/reposts) sans
// son contenu. Charge utile légère de GET /posts/stats : le front rafraîchit
// périodiquement les compteurs des posts affichés (façon X) sans recharger les
// posts ni toucher l'état « moi » (liked/reposted local).
type PostStat struct {
	ID            string `json:"id"`
	LikesCount    int32  `json:"likes_count"`
	CommentsCount int32  `json:"comments_count"`
	RepostsCount  int32  `json:"reposts_count"`
}

// Comment — document de la collection `comments`. Un commentaire référence son
// post par l'identifiant hexadécimal (`post_id`, string) ; l'auteur est dérivé
// du JWT, jamais du corps.
//
// Threading à 2 niveaux : `parent_id` vide = commentaire racine, sinon = id du
// commentaire racine auquel la réponse est rattachée (réponses à plat sous leur
// racine, façon Instagram). `reply_count` (dénormalisé, `int32`) = nombre de
// réponses d'un commentaire racine.
type Comment struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PostID     string        `bson:"post_id" json:"post_id"`
	ParentID   string        `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
	AuthorID   string        `bson:"author_id" json:"author_id"`
	Content    string        `bson:"content" json:"content"`
	Media      []MediaRef    `bson:"media,omitempty" json:"media,omitempty"`
	LikesCount int32         `bson:"likes_count" json:"likes_count"`
	ReplyCount int32         `bson:"reply_count" json:"reply_count"`
	Liked      bool          `bson:"-" json:"liked"`
	CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time     `bson:"updated_at" json:"updated_at"`
}

// CommentStat — compteur dénormalisé d'un commentaire (likes uniquement) sans
// le contenu. Sert au polling batch léger côté front pour rafraîchir les cœurs
// des commentaires visibles sans recharger le thread.
type CommentStat struct {
	ID         string `json:"id"`
	LikesCount int32  `json:"likes_count"`
}

// CommentWithPost enrichit un Comment du post parent visible (hydraté par la
// couche service) pour l'onglet « Réponses » du profil. Jamais persisté.
// `ParentComment` n'est renseigné que si le commentaire est lui-même une réponse
// (ParentID non vide) : le front affiche alors post → commentaire parent →
// réponse.
type CommentWithPost struct {
	Comment
	ParentPost    *Post    `bson:"-" json:"parent_post,omitempty"`
	ParentComment *Comment `bson:"-" json:"parent_comment,omitempty"`
}

// MediaRef : pièce jointe d'un post (image ou vidéo). Le post-service est
// agnostique du contenu : il ne stocke que l'URL (chemin relatif `/media/<id>`
// servi par le media-service via la gateway) et la nature (image/vidéo, pour
// que le front choisisse `<img>` ou `<video>`). Sert à la fois de modèle DB et
// de payload de requête (tags `binding` ignorés hors ShouldBindJSON).
type MediaRef struct {
	URL  string `bson:"url" json:"url" binding:"required"`
	Type string `bson:"type" json:"type" binding:"oneof=image video"`
}

// CreatePostRequest : corps de POST /posts. L'auteur n'est PAS dans le corps —
// il est dérivé du JWT (un utilisateur ne poste que pour lui-même). `content`
// est optionnel SI au moins un média est joint (vérifié dans le handler) ;
// jusqu'à 4 médias.
type CreatePostRequest struct {
	Content string             `json:"content" binding:"max=280"`
	Media   []MediaRef         `json:"media" binding:"max=4,dive"`
	Poll    *CreatePollRequest `json:"poll"`
	// ReplyAudience (optionnel) : qui peut répondre — `everyone` (défaut) ou
	// `followers`. Vide = `everyone`.
	ReplyAudience string `json:"reply_audience" binding:"omitempty,oneof=everyone followers"`
	QuotePostID   string `json:"quote_post_id"`
	// Nsfw : l'auteur marque son propre post comme sensible dès la publication
	// (bouton NSFW du composer). Modifiable ensuite uniquement par modo/admin.
	Nsfw bool `json:"nsfw"`
}

// SetNsfwRequest : corps de PATCH /posts/:id/nsfw (modo/admin). Marque ou
// dé-marque un post comme NSFW.
type SetNsfwRequest struct {
	Nsfw bool `json:"nsfw"`
}

type CreatePollRequest struct {
	Choices         []CreatePollChoiceRequest `json:"choices" binding:"min=2,max=4,dive"`
	DurationMinutes int64                     `json:"duration_minutes" binding:"required,min=1,max=10080"`
	Audience        string                    `json:"audience" binding:"omitempty,oneof=everyone followers"`
}

// CreatePollChoiceRequest : un choix de sondage à la création. `image_url`
// (optionnel) référence un média déjà uploadé (chemin relatif `/media/<id>`) ;
// les choix mixtes (certains avec image, d'autres non) sont autorisés.
type CreatePollChoiceRequest struct {
	Label    string `json:"label" binding:"required,min=1,max=80"`
	ImageURL string `json:"image_url" binding:"omitempty,max=512"`
}

type VotePollRequest struct {
	ChoiceID string `json:"choice_id" binding:"required"`
}

// UpdatePostRequest : corps de PATCH /posts/:id.
type UpdatePostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

// CreateCommentRequest : corps de POST /posts/:id/comments. L'auteur est dérivé
// du JWT, le post de l'URL. `parent_id` (optionnel) cible le commentaire auquel
// on répond (rattaché à plat à la racine côté service).
type CreateCommentRequest struct {
	Content  string     `json:"content" binding:"max=280"`
	Media    []MediaRef `json:"media" binding:"max=4,dive"`
	ParentID string     `json:"parent_id"`
}
