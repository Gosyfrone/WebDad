// Package models contient les structures de données du service user.
package models

import "time"

// Rôles valides (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// User représente une ligne de la table `users` (identité publique).
type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserDetails enrichit User des compteurs du graphe social, pour les vues
// « profil » (un seul utilisateur). Les compteurs sont calculés (COUNT) côté
// repository ; à grande échelle on les dénormaliserait en colonnes dédiées.
type UserDetails struct {
	User
	FollowerCount  int `json:"follower_count"`
	FollowingCount int `json:"following_count"`
}

// CreateUserRequest : payload de POST /users.
// L'id n'est pas dans le corps : il provient du JWT (claims) ou est généré.
type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	DisplayName string `json:"display_name" binding:"max=100"`
}

// UpdateUserRequest : payload de PATCH /users/me.
// Champs optionnels (pointeurs) : seuls les champs fournis sont modifiés.
type UpdateUserRequest struct {
	Username    *string `json:"username" binding:"omitempty,min=3,max=50"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
}
