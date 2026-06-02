// Package services porte la logique métier de l'authentification :
// hash/vérification des mots de passe, génération et validation des JWT,
// et accès aux données (repository simplifié pour ce squelette).
package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/webdad/auth-service/internal/models"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	ErrEmailTaken         = errors.New("email déjà utilisé")
	ErrInvalidCredentials = errors.New("email ou mot de passe invalide")
	ErrUserInactive       = errors.New("compte désactivé")
)

// defaultAdminID : UUID figé de l'admin par défaut, partagé avec les autres
// services (user-init.sql, profil-init.js) pour désigner le même compte.
const defaultAdminID = "00000000-0000-0000-0000-000000000001"

// Claims : contenu du JWT. Partagé avec l'API Gateway et les autres
// services, qui vérifient les tokens avec le même JWT_SECRET.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthService regroupe les dépendances (DB + paramètres JWT).
type AuthService struct {
	db        *sql.DB
	jwtSecret []byte
	jwtExpiry time.Duration
}

// New construit le service.
func New(db *sql.DB, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

// Register crée un compte (role=user) et retourne l'utilisateur créé.
func (s *AuthService) Register(email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash mot de passe : %w", err)
	}

	const q = `
		INSERT INTO credentials (email, password, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, role, is_active, created_at`

	u := &models.User{}
	err = s.db.QueryRow(q, email, string(hash), models.RoleUser).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insertion utilisateur : %w", err)
	}
	return u, nil
}

// Login vérifie les credentials et retourne un JWT signé + l'utilisateur.
func (s *AuthService) Login(email, password string) (string, *models.User, error) {
	const q = `
		SELECT id, email, password, role, is_active, created_at
		FROM credentials WHERE email = $1`

	u := &models.User{}
	err := s.db.QueryRow(q, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, fmt.Errorf("lecture utilisateur : %w", err)
	}

	if !u.IsActive {
		return "", nil, ErrUserInactive
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.GenerateToken(u)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// EnsureDefaultAdmin crée le compte admin par défaut (UUID figé) s'il
// n'existe pas. Idempotent (ON CONFLICT DO NOTHING) : sans effet aux boots
// suivants. Destiné au dev/démo, activé via SEED_DEFAULT_ADMIN.
func (s *AuthService) EnsureDefaultAdmin(email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin : %w", err)
	}

	const q = `
		INSERT INTO credentials (id, email, password, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING`

	if _, err := s.db.Exec(q, defaultAdminID, email, string(hash), models.RoleAdmin); err != nil {
		return fmt.Errorf("seed admin : %w", err)
	}
	return nil
}

// GenerateToken signe un JWT HS256 pour l'utilisateur donné.
func (s *AuthService) GenerateToken(u *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("signature JWT : %w", err)
	}
	return signed, nil
}

// ParseToken valide la signature et l'expiration, puis retourne les claims.
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue : %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token invalide")
	}
	return claims, nil
}

// isUniqueViolation détecte l'erreur PostgreSQL 23505 (contrainte UNIQUE)
// sans dépendre directement du type concret du driver.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
