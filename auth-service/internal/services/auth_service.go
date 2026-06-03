// Package services porte la logique métier de l'authentification :
// hash/vérification des mots de passe, génération et validation des JWT,
// et accès aux données (repository simplifié pour ce squelette).
package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/webdad/auth-service/internal/models"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	ErrEmailTaken          = errors.New("email déjà utilisé")
	ErrInvalidCredentials  = errors.New("email ou mot de passe invalide")
	ErrUserInactive        = errors.New("compte désactivé")
	ErrInvalidRefreshToken = errors.New("refresh token invalide ou expiré")
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
	db            *sql.DB
	jwtSecret     []byte
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

// New construit le service.
func New(db *sql.DB, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *AuthService {
	return &AuthService{
		db:            db,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// Register crée un compte (role=user), puis connecte l'utilisateur dans la
// foulée : il retourne un access token + un refresh token + l'utilisateur créé
// (symétrique de Login).
func (s *AuthService) Register(email, password string) (string, string, *models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", nil, fmt.Errorf("hash mot de passe : %w", err)
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
			return "", "", nil, ErrEmailTaken
		}
		return "", "", nil, fmt.Errorf("insertion utilisateur : %w", err)
	}

	return s.issueTokens(u)
}

// Login vérifie les credentials et retourne un access token + un refresh
// token + l'utilisateur.
func (s *AuthService) Login(email, password string) (string, string, *models.User, error) {
	const q = `
		SELECT id, email, password, role, is_active, created_at
		FROM credentials WHERE email = $1`

	u := &models.User{}
	err := s.db.QueryRow(q, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("lecture utilisateur : %w", err)
	}

	if !u.IsActive {
		return "", "", nil, ErrUserInactive
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	return s.issueTokens(u)
}

// Refresh échange un refresh token valide contre une NOUVELLE paire
// (access + refresh). Rotation : l'ancien refresh token est révoqué, donc
// rejouable une seule fois. Renvoie ErrInvalidRefreshToken si le token est
// inconnu/expiré, ErrUserInactive si le compte a été désactivé entre-temps.
func (s *AuthService) Refresh(rawToken string) (string, string, *models.User, error) {
	tokenHash := hashToken(rawToken)

	const q = `
		SELECT c.id, c.email, c.role, c.is_active, c.created_at, rt.expires_at
		FROM refresh_tokens rt
		JOIN credentials c ON c.id = rt.user_id
		WHERE rt.token = $1`

	u := &models.User{}
	var expiresAt time.Time
	err := s.db.QueryRow(q, tokenHash).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("lecture refresh token : %w", err)
	}

	// Expiré : on le purge et on refuse (le front redirigera vers /login).
	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec(`DELETE FROM refresh_tokens WHERE token = $1`, tokenHash)
		return "", "", nil, ErrInvalidRefreshToken
	}
	if !u.IsActive {
		return "", "", nil, ErrUserInactive
	}

	// Rotation : révocation de l'ancien token avant d'en émettre un nouveau.
	if _, err := s.db.Exec(`DELETE FROM refresh_tokens WHERE token = $1`, tokenHash); err != nil {
		return "", "", nil, fmt.Errorf("révocation refresh token : %w", err)
	}

	return s.issueTokens(u)
}

// Logout révoque un refresh token (suppression en base). Idempotent : un
// token absent/vide n'est pas une erreur (déconnexion = best-effort).
func (s *AuthService) Logout(rawToken string) error {
	if rawToken == "" {
		return nil
	}
	if _, err := s.db.Exec(`DELETE FROM refresh_tokens WHERE token = $1`, hashToken(rawToken)); err != nil {
		return fmt.Errorf("révocation refresh token : %w", err)
	}
	return nil
}

// issueTokens signe un access token (court) et crée un refresh token (long,
// persisté haché). Retourné par Register/Login/Refresh.
func (s *AuthService) issueTokens(u *models.User) (string, string, *models.User, error) {
	access, err := s.GenerateToken(u)
	if err != nil {
		return "", "", nil, err
	}
	refresh, err := s.createRefreshToken(u.ID)
	if err != nil {
		return "", "", nil, err
	}
	return access, refresh, u, nil
}

// createRefreshToken génère un jeton opaque aléatoire, persiste son HASH
// (SHA-256) en base et retourne la valeur EN CLAIR (à poser en cookie). Le
// clair n'est jamais stocké : une fuite de la table ne livre aucun token
// utilisable.
func (s *AuthService) createRefreshToken(userID string) (string, error) {
	raw, err := randomToken()
	if err != nil {
		return "", err
	}

	const q = `INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	expiresAt := time.Now().Add(s.refreshExpiry)
	if _, err := s.db.Exec(q, userID, hashToken(raw), expiresAt); err != nil {
		return "", fmt.Errorf("création refresh token : %w", err)
	}
	return raw, nil
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

// randomToken produit un jeton opaque de 256 bits encodé en base64url.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("génération token aléatoire : %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken hache un refresh token en clair (SHA-256 hex) pour stockage/lookup.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
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
