// Package models contient les structures de données du service profil.
package models

import "time"

// Rôles valides (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// Profil représente un document de la collection `profiles` : uniquement les
// données « décoratives » d'un utilisateur. L'identité (username), les rôles
// et le graphe social vivent dans user-service / le JWT — pas ici (une seule
// source de vérité par donnée). La vue agrégée d'un profil (identité +
// compteurs + décoratif) est composée à la lecture par l'appelant.
type Profil struct {
	UserID      string     `json:"user_id"               bson:"user_id"`
	DisplayName string     `json:"display_name"          bson:"display_name"`
	Bio         string     `json:"bio"                   bson:"bio"`
	AvatarURL   string     `json:"avatar_url"            bson:"avatar_url"`
	BannerURL   string     `json:"banner_url"            bson:"banner_url"`
	Website     string     `json:"website"               bson:"website"`
	Location    string     `json:"location"              bson:"location"`
	BirthDate   *time.Time `json:"birth_date,omitempty"  bson:"birth_date,omitempty"`
	Gender      string     `json:"gender,omitempty"      bson:"gender,omitempty"` // "male" | "female"
	CreatedAt   time.Time  `json:"created_at"            bson:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"            bson:"updated_at"`
}

// UpdateProfilRequest : payload de PATCH /profils/me. Champs optionnels
// (pointeurs) : seuls les champs fournis sont modifiés (nil = inchangé).
type UpdateProfilRequest struct {
	DisplayName *string    `json:"display_name" binding:"omitempty,max=100"`
	Bio         *string    `json:"bio"          binding:"omitempty,max=160"`
	AvatarURL   *string    `json:"avatar_url"   binding:"omitempty"`
	BannerURL   *string    `json:"banner_url"   binding:"omitempty"`
	Website     *string    `json:"website"      binding:"omitempty"`
	Location    *string    `json:"location"     binding:"omitempty"`
	BirthDate   *time.Time `json:"birth_date"   binding:"omitempty"`
	Gender      *string    `json:"gender"       binding:"omitempty,oneof=male female"`
}
