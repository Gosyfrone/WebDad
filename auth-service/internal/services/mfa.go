package services

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/webdad/auth-service/internal/models"
)

// Erreurs MFA (mappées vers des codes HTTP par les handlers).
var (
	// ErrMFANotConfigured : MFA_ENCRYPTION_KEY absente → la feature est désactivée.
	ErrMFANotConfigured = errors.New("MFA non configurée sur le serveur")
	// ErrMFAAlreadyEnabled : tentative de (re)setup alors que la MFA est déjà active.
	ErrMFAAlreadyEnabled = errors.New("la double authentification est déjà activée")
	// ErrMFANotEnabled : opération nécessitant une MFA active alors qu'elle ne l'est pas.
	ErrMFANotEnabled = errors.New("la double authentification n'est pas activée")
	// ErrMFANotPending : enable sans setup préalable (aucun secret en attente).
	ErrMFANotPending = errors.New("aucune configuration MFA en attente — lance d'abord le setup")
	// ErrInvalidMFACode : code TOTP (ou mot de passe de désactivation) invalide.
	ErrInvalidMFACode = errors.New("code de vérification invalide")
)

const (
	// mfaIssuer : nom affiché dans l'app d'authentification (Microsoft/Google Authenticator…).
	mfaIssuer = "Breezy"
	// purposeMFAChallenge : usage des account_tokens pour le challenge post-mot-de-passe.
	purposeMFAChallenge = "mfa_challenge"
	// mfaChallengeTTL : fenêtre pour saisir le second facteur après le mot de passe.
	mfaChallengeTTL = 5 * time.Minute
)

// MFAConfigured indique si la MFA est utilisable (clé de chiffrement présente).
func (s *AuthService) MFAConfigured() bool { return s.mfaCipher != nil }

// SetupMFA génère un nouveau secret TOTP, le stocke CHIFFRÉ (mfa_enabled reste
// false tant qu'il n'est pas confirmé par EnableMFA), et renvoie le secret en
// clair + l'URI otpauth:// + un QR PNG (data URI) à scanner. Refuse si la MFA
// est déjà active (il faut d'abord la désactiver).
func (s *AuthService) SetupMFA(userID string) (*models.MFASetupData, error) {
	if s.mfaCipher == nil {
		return nil, ErrMFANotConfigured
	}

	var email string
	var enabled bool
	err := s.db.QueryRow(`SELECT email, mfa_enabled FROM credentials WHERE id = $1`, userID).
		Scan(&email, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lecture utilisateur : %w", err)
	}
	if enabled {
		return nil, ErrMFAAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{Issuer: mfaIssuer, AccountName: email})
	if err != nil {
		return nil, fmt.Errorf("génération secret TOTP : %w", err)
	}

	enc, err := s.mfaCipher.encrypt(key.Secret())
	if err != nil {
		return nil, fmt.Errorf("chiffrement secret TOTP : %w", err)
	}
	// Écrase tout secret en attente d'un précédent setup non confirmé (idempotent).
	if _, err := s.db.Exec(
		`UPDATE credentials SET mfa_secret = $1, mfa_enabled = false WHERE id = $2`,
		enc, userID,
	); err != nil {
		return nil, fmt.Errorf("enregistrement secret MFA : %w", err)
	}

	img, err := key.Image(256, 256)
	if err != nil {
		return nil, fmt.Errorf("génération QR : %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encodage QR PNG : %w", err)
	}

	return &models.MFASetupData{
		Secret:     key.Secret(),
		OtpauthURL: key.URL(),
		QRDataURI:  "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// EnableMFA confirme la configuration : valide le code TOTP courant contre le
// secret en attente, puis passe mfa_enabled à true. ErrMFANotPending si aucun
// setup n'a été lancé, ErrMFAAlreadyEnabled si déjà active, ErrInvalidMFACode
// si le code ne correspond pas.
func (s *AuthService) EnableMFA(userID, code string) error {
	if s.mfaCipher == nil {
		return ErrMFANotConfigured
	}

	var enc sql.NullString
	var enabled bool
	err := s.db.QueryRow(`SELECT mfa_secret, mfa_enabled FROM credentials WHERE id = $1`, userID).
		Scan(&enc, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lecture utilisateur : %w", err)
	}
	if enabled {
		return ErrMFAAlreadyEnabled
	}
	if !enc.Valid || enc.String == "" {
		return ErrMFANotPending
	}

	secret, err := s.mfaCipher.decrypt(enc.String)
	if err != nil {
		return fmt.Errorf("déchiffrement secret MFA : %w", err)
	}
	if !validateTOTP(secret, code) {
		return ErrInvalidMFACode
	}

	if _, err := s.db.Exec(`UPDATE credentials SET mfa_enabled = true WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("activation MFA : %w", err)
	}
	return nil
}

// DisableMFA désactive la MFA après preuve d'identité : un code TOTP courant OU
// le mot de passe du compte suffit. Efface le secret et le drapeau. ErrMFANotEnabled
// si la MFA n'était pas active, ErrInvalidMFACode si aucune preuve n'est valide.
func (s *AuthService) DisableMFA(userID, code, password string) error {
	if s.mfaCipher == nil {
		return ErrMFANotConfigured
	}

	var enc sql.NullString
	var enabled bool
	var pwHash sql.NullString
	err := s.db.QueryRow(`SELECT mfa_secret, mfa_enabled, password FROM credentials WHERE id = $1`, userID).
		Scan(&enc, &enabled, &pwHash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lecture utilisateur : %w", err)
	}
	if !enabled {
		return ErrMFANotEnabled
	}

	authorized := false
	if c := strings.TrimSpace(code); c != "" && enc.Valid && enc.String != "" {
		if secret, derr := s.mfaCipher.decrypt(enc.String); derr == nil && validateTOTP(secret, c) {
			authorized = true
		}
	}
	if !authorized && password != "" && pwHash.Valid && pwHash.String != "" {
		if bcrypt.CompareHashAndPassword([]byte(pwHash.String), []byte(password)) == nil {
			authorized = true
		}
	}
	if !authorized {
		return ErrInvalidMFACode
	}

	if _, err := s.db.Exec(
		`UPDATE credentials SET mfa_secret = NULL, mfa_enabled = false WHERE id = $1`, userID,
	); err != nil {
		return fmt.Errorf("désactivation MFA : %w", err)
	}
	return nil
}

// VerifyMFA valide le second facteur lors d'un login : vérifie le challenge
// (sans le consommer), valide le code TOTP, PUIS consomme le challenge et émet
// la session. Un code erroné ne brûle donc pas le challenge (l'utilisateur peut
// réessayer pendant la fenêtre). Renvoie ErrInvalidToken si le challenge est
// inconnu/expiré/consommé, ErrInvalidMFACode si le code TOTP est faux.
func (s *AuthService) VerifyMFA(challenge, code string) (string, string, *models.User, error) {
	if s.mfaCipher == nil {
		return "", "", nil, ErrMFANotConfigured
	}

	tx, err := s.db.Begin()
	if err != nil {
		return "", "", nil, fmt.Errorf("début transaction MFA : %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lecture verrouillée du challenge (sans le marquer consommé tout de suite).
	const tokenQuery = `
		SELECT id, user_id, expires_at, used_at
		FROM account_tokens
		WHERE token_hash = $1 AND purpose = $2
		FOR UPDATE`
	var tokenID, userID string
	var expiresAt time.Time
	var usedAt sql.NullTime
	if err := tx.QueryRow(tokenQuery, hashToken(challenge), purposeMFAChallenge).
		Scan(&tokenID, &userID, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, ErrInvalidToken
		}
		return "", "", nil, fmt.Errorf("lecture challenge MFA : %w", err)
	}
	if usedAt.Valid || time.Now().After(expiresAt) {
		return "", "", nil, ErrInvalidToken
	}

	u := &models.User{}
	var enc sql.NullString
	const userQuery = `
		SELECT id, email, role, is_active, email_verified, must_change_password, mfa_enabled, created_at, mfa_secret
		FROM credentials WHERE id = $1`
	if err := tx.QueryRow(userQuery, userID).Scan(
		&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified,
		&u.MustChangePassword, &u.MFAEnabled, &u.CreatedAt, &enc,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, ErrUserNotFound
		}
		return "", "", nil, fmt.Errorf("lecture utilisateur : %w", err)
	}
	if !u.IsActive {
		return "", "", nil, ErrUserInactive
	}
	if !u.MFAEnabled || !enc.Valid || enc.String == "" {
		return "", "", nil, ErrMFANotEnabled
	}

	secret, err := s.mfaCipher.decrypt(enc.String)
	if err != nil {
		return "", "", nil, fmt.Errorf("déchiffrement secret MFA : %w", err)
	}
	if !validateTOTP(secret, code) {
		return "", "", nil, ErrInvalidMFACode
	}

	// Code valide : on consomme le challenge puis on valide la transaction.
	if _, err := tx.Exec(`UPDATE account_tokens SET used_at = NOW() WHERE id = $1`, tokenID); err != nil {
		return "", "", nil, fmt.Errorf("consommation challenge MFA : %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", "", nil, fmt.Errorf("validation MFA : %w", err)
	}

	return s.issueTokens(u)
}

// MFAStatus renvoie l'état d'activation de la MFA d'un compte (pour l'affichage
// de la section Sécurité côté front).
func (s *AuthService) MFAStatus(userID string) (bool, error) {
	var enabled bool
	err := s.db.QueryRow(`SELECT mfa_enabled FROM credentials WHERE id = $1`, userID).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrUserNotFound
	}
	if err != nil {
		return false, fmt.Errorf("lecture statut MFA : %w", err)
	}
	return enabled, nil
}

// validateTOTP vérifie un code TOTP à 6 chiffres (période 30s) avec une
// tolérance de ±1 fenêtre (skew) pour absorber la dérive d'horloge.
func validateTOTP(secret, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	ok, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && ok
}
