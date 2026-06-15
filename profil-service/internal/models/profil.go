// Package models contient les structures de données du service profil.
package models

import "time"

// Rôles valides (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// Profil représente un document de la collection `profiles` : uniquement les
// données « décoratives » d'un utilisateur. L'identité (username), les rôles
// et le graphe social vivent dans user-service / le JWT — pas ici (une seule
// source de vérité par donnée). La vue agrégée d'un profil (identité +
// compteurs + décoratif) est composée à la lecture par l'appelant.
type Profil struct {
	UserID          string     `json:"user_id"               bson:"user_id"`
	DisplayName     string     `json:"display_name"          bson:"display_name"`
	Bio             string     `json:"bio"                   bson:"bio"`
	AvatarURL       string     `json:"avatar_url"            bson:"avatar_url"`
	BannerURL       string     `json:"banner_url"            bson:"banner_url"`
	Website         string     `json:"website"               bson:"website"`
	Location        string     `json:"location"              bson:"location"`
	BirthDate       *time.Time `json:"birth_date,omitempty"  bson:"birth_date,omitempty"`
	Gender          string     `json:"gender,omitempty"      bson:"gender,omitempty"`      // "male" | "female"
	Nationality     string     `json:"nationality,omitempty" bson:"nationality,omitempty"` // ISO 3166-1 alpha-2
	CreatedAt       time.Time  `json:"created_at"            bson:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"            bson:"updated_at"`
	Visibility      string     `json:"visibility"       bson:"visibility,omitempty"`
	LikesVisibility string     `json:"likes_visibility"  bson:"likes_visibility,omitempty"`
	// DisplayNameChangedAt : date du dernier changement EFFECTIF de display_name
	// (nil = jamais changé depuis le provisioning). Enregistrée dès aujourd'hui
	// pour servir de base à un cooldown « X jours entre deux changements de nom »
	// (cf. config DisplayNameCooldown ; enforcement désactivé par défaut). On la
	// capture maintenant pour ne pas avoir de profils sans baseline le jour où
	// le cooldown est activé. Symétrique de users.username_changed_at.
	DisplayNameChangedAt *time.Time `json:"display_name_changed_at,omitempty" bson:"display_name_changed_at,omitempty"`
}

// CreateProfilRequest : payload de POST /profils. L'id provient TOUJOURS du
// JWT, jamais du corps. display_name est OBLIGATOIRE : le BFF y met le username
// choisi à l'inscription. On ne dérive jamais de nom depuis l'email (le
// provisioning paresseux crée un profil à display_name vide, cf. service).
type CreateProfilRequest struct {
	DisplayName string     `json:"display_name" binding:"required,max=100"`
	BirthDate   *time.Time `json:"birth_date"   binding:"omitempty"`
	Gender      *string    `json:"gender"       binding:"omitempty,oneof=male female"`
}

// AdminCreateProfilRequest : payload de POST /profils/admin (admin). Crée le
// profil d'un utilisateur donné (= credentials.id) lors du provisioning d'un
// compte créé de force par un admin. display_name = username effectif ; pas de
// birth_date (l'utilisateur la renseignera lui-même ensuite).
type AdminCreateProfilRequest struct {
	ID          string `json:"id"           binding:"required"`
	DisplayName string `json:"display_name" binding:"required,max=100"`
}

// UpdateProfilRequest : payload de PATCH /profils/me. Champs optionnels
// (pointeurs) : seuls les champs fournis sont modifiés (nil = inchangé).
// birth_date est settable UNE SEULE FOIS : une fois posée, toute tentative de
// la changer est refusée (cf. service).
type UpdateProfilRequest struct {
	DisplayName     *string    `json:"display_name" binding:"omitempty,max=100"`
	Bio             *string    `json:"bio"          binding:"omitempty,max=160"`
	AvatarURL       *string    `json:"avatar_url"   binding:"omitempty"`
	BannerURL       *string    `json:"banner_url"   binding:"omitempty"`
	Website         *string    `json:"website"      binding:"omitempty"`
	Location        *string    `json:"location"     binding:"omitempty"`
	BirthDate       *time.Time `json:"birth_date"   binding:"omitempty"`
	Gender          *string    `json:"gender"       binding:"omitempty,oneof=male female"`
	Nationality     *string    `json:"nationality"  binding:"omitempty,iso3166_1_alpha2"`
	Visibility      *string    `json:"visibility"        binding:"omitempty,oneof=public private"`
	LikesVisibility *string    `json:"likes_visibility"   binding:"omitempty,oneof=public private"`
}
