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
// Le nom affiché (display_name) et les autres champs décoratifs vivent dans
// profil-service (Mongo) — user-service ne porte que l'identité immuable
// (username) + l'état du compte.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// UsernameChangedAt : date du dernier changement effectif du handle
	// (nil = jamais changé). Sert de base au cooldown de changement de
	// username (cf. service). Symétrique de display_name_changed_at (profil).
	UsernameChangedAt *time.Time `json:"username_changed_at,omitempty"`

	// UsernamePending : true si le username a été attribué d'office avec un
	// suffixe (compte créé par un admin, handle demandé déjà pris). Le front
	// impose alors un changement de username (repassé à false dès qu'un handle
	// libre est choisi via PATCH /users/me).
	UsernamePending bool `json:"username_pending"`
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
// Le nom affiché n'est plus ici (→ profil-service).
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}

// AdminCreateUserRequest : payload de POST /users/admin (admin). Crée la ligne
// `users` pour un id donné (= credentials.id retourné par auth-service). Si le
// username demandé est déjà pris, un suffixe `_<hex>` est ajouté et
// username_pending passe à true.
type AdminCreateUserRequest struct {
	ID       string `json:"id"       binding:"required"`
	Username string `json:"username" binding:"required,min=3,max=50"`
}

// UpdateUserRequest : payload de PATCH /users/me.
// Champs optionnels (pointeurs) : seuls les champs fournis sont modifiés.
// Le nom affiché se modifie via profil-service (PATCH /profils/me).
type UpdateUserRequest struct {
	Username *string `json:"username" binding:"omitempty,min=3,max=50"`
}

// UpdateStatusRequest : payload de PATCH /users/:id/status (admin).
// Pointeur + required : force la présence explicite de `is_active`.
type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}
