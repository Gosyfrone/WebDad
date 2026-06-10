// Package models contient les structures de données du service auth.
package models

import "time"

// Rôles valides (alignés sur l'enum SQL user_role).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// User représente une ligne de la table `credentials`.
// PasswordHash n'est jamais sérialisé en JSON (tag `json:"-"`).
type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"`        // colonne `password` (hash bcrypt, NULL si compte OAuth)
	Provider      string     `json:"provider"` // 'local' | 'google' | 'microsoft'
	Role          string     `json:"role"`
	IsActive      bool       `json:"is_active"`
	EmailVerified bool       `json:"email_verified"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"` // date du bannissement (NULL si actif)
	CreatedAt     time.Time  `json:"created_at"`
}

// AuthUser : vue publique d'un utilisateur renvoyée par l'API d'auth.
// Découple le contrat HTTP du modèle DB : on n'expose que l'identité utile
// au front (le rôle pilote la nav), sans `is_active`/`created_at`.
type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// NewAuthUser projette un User (modèle DB) vers sa vue publique.
func NewAuthUser(u *User) AuthUser {
	return AuthUser{ID: u.ID, Email: u.Email, Role: u.Role}
}

// RegisterRequest : payload de POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest : payload de POST /auth/login. Le BFF envoie `email` pour le
// login classique, ou `user_id` après résolution d'un username via user-service.
type LoginRequest struct {
	Email    string `json:"email,omitempty" binding:"omitempty,email"`
	UserID   string `json:"user_id,omitempty"`
	Password string `json:"password" binding:"required"`
}

// VerifyEmailRequest : payload de POST /auth/verify-email/confirm (token en clair
// extrait du lien reçu par e-mail).
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// RequestVerifyRequest : payload de POST /auth/verify-email/request (renvoi du
// mail de vérification). Réponse générique (anti-énumération).
type RequestVerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPasswordRequest : payload de POST /auth/password/forgot. Réponse
// TOUJOURS générique (anti-énumération), que le compte existe ou non.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest : payload de POST /auth/password/reset (token en clair
// extrait du lien reçu par e-mail + nouveau mot de passe).
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// OAuthExchangeRequest : payload de POST /auth/oauth/:provider/exchange.
// `code` = code d'autorisation renvoyé par le provider au callback front.
// `state` est facultatif côté serveur (le front l'a déjà revérifié) ; il est
// accepté pour symétrie avec /url.
type OAuthExchangeRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state"`
}

// RefreshRequest : payload de POST /auth/refresh et /auth/logout. Le refresh
// token transite dans le corps (transmis par le BFF Next depuis le cookie
// httpOnly) — pas de binding `required` pour que /logout reste best-effort.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// UpdateRoleRequest : payload de PATCH /auth/users/:id/role (admin).
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateStatusRequest : payload de PATCH /auth/users/:id/status (admin).
// Pointeur + required : force la présence explicite de `is_active` (sinon un
// `false` omis serait indistinct d'un champ absent).
type UpdateStatusRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}
