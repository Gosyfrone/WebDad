// Package services porte la logique métier de l'authentification :
// hash/vérification des mots de passe, génération et validation des JWT,
// et accès aux données (repository simplifié pour ce squelette).
package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/webdad/auth-service/internal/eraser"
	"github.com/webdad/auth-service/internal/models"
)

// Erreurs métier (mappées vers des codes HTTP par les handlers).
var (
	ErrEmailTaken          = errors.New("email déjà utilisé")
	ErrInvalidCredentials  = errors.New("identifiant ou mot de passe invalide")
	ErrUserInactive        = errors.New("compte désactivé")
	ErrInvalidRefreshToken = errors.New("refresh token invalide ou expiré")
	ErrUserNotFound        = errors.New("utilisateur introuvable")
	ErrInvalidRole         = errors.New("rôle invalide (user, moderator ou admin)")
	// ErrInsufficientPrivilege : l'acteur n'a pas le niveau pour agir sur la
	// cible (ex. un modérateur tente de bannir un autre modérateur / un admin).
	ErrInsufficientPrivilege = errors.New("privilèges insuffisants pour cette cible")
	// ErrEmailNotVerified : login refusé tant que l'adresse n'est pas vérifiée.
	ErrEmailNotVerified = errors.New("adresse e-mail non vérifiée")
	// ErrInvalidToken : token de vérification inconnu / expiré / déjà consommé.
	ErrInvalidToken = errors.New("token invalide ou expiré")
	// ErrNoLocalPassword : tentative de login classique sur un compte créé via
	// un provider externe (password NULL). Le front doit rediriger vers OAuth.
	ErrNoLocalPassword = errors.New("ce compte se connecte via un fournisseur externe (Google) ; aucun mot de passe n'est défini")
	// ErrInvalidCurrentPassword : changement de mot de passe refusé car le mot de
	// passe actuel fourni ne correspond pas (ou le compte n'a pas de mot de passe local).
	ErrInvalidCurrentPassword = errors.New("mot de passe actuel invalide")
)

// Usages des account_tokens + TTL de la vérification d'e-mail.
const (
	purposeVerify  = "verify"
	purposeReset   = "reset"
	verifyTokenTTL = 24 * time.Hour
	resetTokenTTL  = 1 * time.Hour
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
	// MustChangePassword : propagé pour que le front impose un changement de mot
	// de passe bloquant (compte créé par un admin avec un mot de passe temporaire),
	// sans appel supplémentaire. Effacé au prochain token après le changement.
	MustChangePassword bool `json:"must_change_password,omitempty"`
	jwt.RegisteredClaims
}

// Mailer envoie un e-mail transactionnel (implémenté par internal/notify).
// Best-effort : l'appelant logge l'erreur sans la propager.
type Mailer interface {
	Send(to, subject, html, text string) error
}

// AuthService regroupe les dépendances (DB + paramètres JWT + mailer).
type AuthService struct {
	db            *sql.DB
	jwtSecret     []byte
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
	mailer        Mailer // nil = envoi d'e-mails désactivé (no-op loggé)
	appBaseURL    string // base URL du front (liens dans les e-mails)
	// adminCreateAutoVerify : DEV/LOCAL — marque les comptes créés par un admin
	// comme vérifiés d'office (court-circuit de la vérif e-mail). False en prod.
	adminCreateAutoVerify bool
}

// New construit le service. mailer peut être nil (mail non configuré) : l'envoi
// devient alors un no-op loggé et auth reste pleinement fonctionnel.
func New(db *sql.DB, jwtSecret string, jwtExpiry, refreshExpiry time.Duration, mailer Mailer, appBaseURL string, adminCreateAutoVerify bool) *AuthService {
	return &AuthService{
		db:                    db,
		jwtSecret:             []byte(jwtSecret),
		jwtExpiry:             jwtExpiry,
		refreshExpiry:         refreshExpiry,
		mailer:                mailer,
		appBaseURL:            appBaseURL,
		adminCreateAutoVerify: adminCreateAutoVerify,
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

	// Envoi du mail de vérification (best-effort). Les tokens sont tout de même
	// émis : le BFF s'en sert UNIQUEMENT côté serveur pour le provisioning
	// (users+profils) puis les jette — le client n'obtient pas de session, le
	// blocage réel est appliqué au login (email_verified). Cf. DECISIONS.md.
	s.sendVerificationMail(u)

	return s.issueTokens(u)
}

// AdminCreateUser crée un compte (role=user) au nom d'un administrateur, avec un
// mot de passe TEMPORAIRE (must_change_password=true → changement imposé à la
// première connexion, cf. ChangePassword).
//
// Vérification d'e-mail OBLIGATOIRE en prod : le compte est créé NON vérifié
// (email_verified=false) et l'utilisateur reçoit un e-mail contenant le lien de
// vérification + son mot de passe temporaire ; cliquer le lien vérifie l'adresse
// ET ouvre la session (cf. VerifyEmail) → il atterrit sur le feed avec la modale
// de changement de mot de passe. En DEV/LOCAL (adminCreateAutoVerify=true), le
// compte est vérifié d'office (court-circuit) : connexion directe au mot de passe
// temporaire. Best-effort sur l'e-mail (ne casse jamais la création).
// Retourne le compte créé ; ErrEmailTaken si l'adresse est déjà utilisée.
func (s *AuthService) AdminCreateUser(email, password, username string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash mot de passe : %w", err)
	}

	const q = `
		INSERT INTO credentials (email, password, role, email_verified, must_change_password)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id, email, role, is_active, email_verified, must_change_password, created_at`

	u := &models.User{}
	err = s.db.QueryRow(q, email, string(hash), models.RoleUser, s.adminCreateAutoVerify).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified, &u.MustChangePassword, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insertion utilisateur (admin) : %w", err)
	}

	s.sendAdminWelcomeMail(u, username, password)
	return u, nil
}

// ChangePassword remplace le mot de passe d'un compte authentifié après
// vérification du mot de passe ACTUEL, lève le drapeau must_change_password
// (fin du flux « mot de passe temporaire »), révoque toutes les autres sessions
// puis ré-émet une paire de tokens (le nouveau JWT ne porte plus le drapeau).
// ErrInvalidCurrentPassword si le mot de passe actuel ne correspond pas (ou si
// le compte n'a pas de mot de passe local, ex. compte OAuth).
func (s *AuthService) ChangePassword(userID, currentPassword, newPassword string) (string, string, *models.User, error) {
	const sel = `
		SELECT id, email, password, role, is_active, email_verified, must_change_password, created_at
		FROM credentials WHERE id = $1`
	u := &models.User{}
	var pwHash sql.NullString
	err := s.db.QueryRow(sel, userID).
		Scan(&u.ID, &u.Email, &pwHash, &u.Role, &u.IsActive, &u.EmailVerified, &u.MustChangePassword, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrUserNotFound
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("lecture utilisateur : %w", err)
	}
	if !pwHash.Valid || pwHash.String == "" {
		return "", "", nil, ErrInvalidCurrentPassword // compte sans mot de passe local
	}
	if bcrypt.CompareHashAndPassword([]byte(pwHash.String), []byte(currentPassword)) != nil {
		return "", "", nil, ErrInvalidCurrentPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", "", nil, fmt.Errorf("hash du nouveau mot de passe : %w", err)
	}
	if _, err := s.db.Exec(
		`UPDATE credentials SET password = $1, must_change_password = false WHERE id = $2`,
		string(hash), userID,
	); err != nil {
		return "", "", nil, fmt.Errorf("mise à jour du mot de passe : %w", err)
	}

	// Révoque toutes les sessions ouvertes : un changement de mot de passe doit
	// déconnecter partout. La nouvelle paire émise juste après rouvre la session courante.
	if _, err := s.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return "", "", nil, fmt.Errorf("révocation des sessions : %w", err)
	}

	u.MustChangePassword = false // le nouveau token ne doit plus porter le drapeau
	return s.issueTokens(u)
}

// sendAdminWelcomeMail envoie l'e-mail de bienvenue d'un compte créé par un
// admin : il porte TOUJOURS le mot de passe temporaire, et — quand la
// vérification est requise (prod) — un lien de vérification d'e-mail qui ouvre la
// session à la volée (cf. VerifyEmail). En auto-vérification (dev), le CTA pointe
// directement vers la connexion. Best-effort : toute erreur est loggée, jamais
// propagée. Le gabarit échappe le contenu (mot de passe arbitraire sûr en HTML).
func (s *AuthService) sendAdminWelcomeMail(u *models.User, username, tempPassword string) {
	if s.mailer == nil {
		slog.Debug("mail désactivé : e-mail bienvenue admin non envoyé", "user_id", u.ID)
		return
	}

	who := username
	if who == "" {
		who = u.Email
	}
	base := strings.TrimRight(s.appBaseURL, "/")

	// Compte vérifié d'office (dev) : connexion directe, pas de lien de vérif.
	if u.EmailVerified {
		loginURL := base + "/login"
		subject := "Ton compte Breezy a été créé — mot de passe temporaire"
		text := fmt.Sprintf(
			"Bonjour %s,\n\n"+
				"Un administrateur vient de créer ton compte Breezy.\n\n"+
				"Identifiants de connexion :\n"+
				"  E-mail : %s\n"+
				"  Mot de passe temporaire : %s\n\n"+
				"Connecte-toi ici : %s\n\n"+
				"Pour des raisons de sécurité, tu devras choisir un nouveau mot de "+
				"passe dès ta première connexion.",
			who, u.Email, tempPassword, loginURL)
		htmlBody := brandedEmailHTML(s.appBaseURL,
			"Ton compte Breezy est prêt 🎉",
			fmt.Sprintf("Un administrateur a créé ton compte. Connecte-toi avec l'e-mail %s et le mot de passe temporaire ci-dessous — tu devras le changer dès ta première connexion.",
				u.Email),
			tempPassword,
			"Se connecter", loginURL,
			"Si tu n'attendais pas cet e-mail, ignore-le ou contacte l'administrateur.")
		if err := s.mailer.Send(u.Email, subject, htmlBody, text); err != nil {
			slog.Warn("envoi e-mail bienvenue admin échoué (best-effort)", "user_id", u.ID, "error", err)
		}
		return
	}

	// Prod : vérification d'e-mail obligatoire. Le lien vérifie l'adresse ET
	// ouvre la session → l'utilisateur arrive sur le feed avec la modale de
	// changement de mot de passe (drapeau porté par le token, cf. VerifyEmail).
	raw, err := s.createAccountToken(u.ID, purposeVerify, verifyTokenTTL)
	if err != nil {
		slog.Error("création token vérification (bienvenue admin) échouée", "user_id", u.ID, "error", err)
		return
	}
	link := fmt.Sprintf("%s/verify-email?token=%s", base, url.QueryEscape(raw))

	subject := "Ton compte Breezy a été créé — vérifie ton adresse"
	text := fmt.Sprintf(
		"Bonjour %s,\n\n"+
			"Un administrateur vient de créer ton compte Breezy.\n\n"+
			"E-mail : %s\n"+
			"Mot de passe temporaire : %s\n\n"+
			"Vérifie ton adresse e-mail pour te connecter en ouvrant ce lien :\n%s\n\n"+
			"Ce lien expire dans 24 heures. À ta première connexion, tu devras "+
			"choisir un nouveau mot de passe.",
		who, u.Email, tempPassword, link)
	htmlBody := brandedEmailHTML(s.appBaseURL,
		"Ton compte Breezy est prêt 🎉",
		"Un administrateur a créé ton compte. Ton mot de passe temporaire est ci-dessous. Vérifie ton adresse e-mail pour te connecter — tu devras ensuite choisir un nouveau mot de passe.",
		tempPassword,
		"Vérifier mon adresse e-mail", link,
		"Ce lien expire dans 24 heures. Si tu n'attendais pas cet e-mail, ignore-le ou contacte l'administrateur.")

	if err := s.mailer.Send(u.Email, subject, htmlBody, text); err != nil {
		slog.Warn("envoi e-mail bienvenue admin (vérif) échoué (best-effort)", "user_id", u.ID, "error", err)
	}
}

// Login vérifie les credentials et retourne un access token + un refresh
// token + l'utilisateur.
func (s *AuthService) Login(email, password string) (string, string, *models.User, error) {
	const q = `
		SELECT id, email, password, role, is_active, email_verified, must_change_password, created_at
		FROM credentials WHERE email = $1`

	return s.loginWithQuery(q, email, password)
}

// LoginByUserID vérifie les credentials à partir de l'id auth. Utilisé par le
// BFF après résolution d'un username dans user-service.
func (s *AuthService) LoginByUserID(userID, password string) (string, string, *models.User, error) {
	const q = `
		SELECT id, email, password, role, is_active, email_verified, must_change_password, created_at
		FROM credentials WHERE id = $1`

	return s.loginWithQuery(q, userID, password)
}

func (s *AuthService) loginWithQuery(query, identifier, password string) (string, string, *models.User, error) {
	u := &models.User{}
	var pwHash sql.NullString // NULL pour les comptes OAuth (sans mot de passe)
	err := s.db.QueryRow(query, identifier).
		Scan(&u.ID, &u.Email, &pwHash, &u.Role, &u.IsActive, &u.EmailVerified, &u.MustChangePassword, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("lecture utilisateur : %w", err)
	}

	if !u.IsActive {
		return "", "", nil, ErrUserInactive
	}
	// Compte sans mot de passe local (créé via OAuth) : login classique refusé
	// avec un message clair plutôt qu'un 401 générique.
	if !pwHash.Valid || pwHash.String == "" {
		return "", "", nil, ErrNoLocalPassword
	}
	if bcrypt.CompareHashAndPassword([]byte(pwHash.String), []byte(password)) != nil {
		return "", "", nil, ErrInvalidCredentials
	}
	// Blocage dur : un compte non vérifié ne peut pas se connecter (aucun token
	// émis). Vérifié APRÈS le bcrypt pour ne pas révéler l'existence du compte.
	if !u.EmailVerified {
		return "", "", nil, ErrEmailNotVerified
	}

	return s.issueTokens(u)
}

// LoginWithOAuth connecte un utilisateur à partir d'une identité OIDC vérifiée
// (email + subject du provider). Rapprochement par email :
//   - compte existant → on le connecte et on renseigne provider_subject s'il
//     n'est pas déjà lié (le compte local conserve son mot de passe) ;
//   - aucun compte → création d'un compte (role=user) SANS mot de passe
//     (password NULL), avec provider + provider_subject renseignés.
//
// Émet ensuite NOS tokens (access + refresh), exactement comme Login.
func (s *AuthService) LoginWithOAuth(provider, subject, email string) (string, string, *models.User, error) {
	u := &models.User{}

	const sel = `
		SELECT id, email, role, is_active, created_at, provider
		FROM credentials WHERE email = $1`
	err := s.db.QueryRow(sel, email).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.Provider)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Création : compte OAuth sans mot de passe local.
		const ins = `
			INSERT INTO credentials (email, password, role, provider, provider_subject)
			VALUES ($1, NULL, $2, $3, $4)
			RETURNING id, email, role, is_active, created_at, provider`
		if err := s.db.QueryRow(ins, email, models.RoleUser, provider, subject).
			Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.Provider); err != nil {
			return "", "", nil, fmt.Errorf("création compte OAuth : %w", err)
		}

	case err != nil:
		return "", "", nil, fmt.Errorf("lecture utilisateur : %w", err)

	default:
		if !u.IsActive {
			return "", "", nil, ErrUserInactive
		}
		// Rapprochement : renseigne provider_subject uniquement s'il est absent
		// (on ne réécrase pas un lien existant → pas de prise de contrôle via un
		// autre compte externe partageant l'email).
		const upd = `
			UPDATE credentials SET provider_subject = $1
			WHERE id = $2 AND provider_subject IS NULL`
		if _, err := s.db.Exec(upd, subject, u.ID); err != nil {
			return "", "", nil, fmt.Errorf("rapprochement compte OAuth : %w", err)
		}
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
		SELECT c.id, c.email, c.role, c.is_active, c.must_change_password, c.created_at, rt.expires_at
		FROM refresh_tokens rt
		JOIN credentials c ON c.id = rt.user_id
		WHERE rt.token = $1`

	u := &models.User{}
	var expiresAt time.Time
	err := s.db.QueryRow(q, tokenHash).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.MustChangePassword, &u.CreatedAt, &expiresAt)
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

// VerifyEmail consomme un token de vérification, marque l'adresse vérifiée et
// ouvre une session dans la foulée (access + refresh) : cliquer le lien prouve
// la possession de la boîte, l'utilisateur entre donc directement dans l'app
// sans se reconnecter. Renvoie ErrInvalidToken si le token est inconnu /
// expiré / déjà utilisé, ErrUserInactive si le compte a été désactivé entre-temps.
func (s *AuthService) VerifyEmail(rawToken string) (string, string, *models.User, error) {
	userID, err := s.consumeAccountToken(rawToken, purposeVerify)
	if err != nil {
		return "", "", nil, err
	}

	// UPDATE ... RETURNING : on marque vérifié ET on récupère l'utilisateur en
	// une requête, pour pouvoir lui émettre une session immédiatement.
	const q = `
		UPDATE credentials SET email_verified = true
		WHERE id = $1
		RETURNING id, email, role, is_active, email_verified, must_change_password, created_at`
	u := &models.User{}
	if err := s.db.QueryRow(q, userID).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified, &u.MustChangePassword, &u.CreatedAt); err != nil {
		return "", "", nil, fmt.Errorf("activation email_verified : %w", err)
	}
	if !u.IsActive {
		return "", "", nil, ErrUserInactive // compte banni : pas de session.
	}

	return s.issueTokens(u)
}

// ResendVerification renvoie un mail de vérification. ANTI-ÉNUMÉRATION : renvoie
// TOUJOURS nil — l'appelant ne peut pas distinguer un compte inexistant, déjà
// vérifié, ou non vérifié. Le mail n'est (ré)envoyé que dans ce dernier cas.
func (s *AuthService) ResendVerification(email string) error {
	const q = `
		SELECT id, email, role, is_active, email_verified, created_at
		FROM credentials WHERE email = $1`

	u := &models.User{}
	err := s.db.QueryRow(q, email).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // compte inexistant : no-op silencieux (anti-énumération).
	}
	if err != nil {
		slog.Error("resend vérification : lecture DB échouée", "error", err)
		return nil
	}
	if u.EmailVerified {
		return nil // déjà vérifié : rien à faire.
	}

	s.sendVerificationMail(u)
	return nil
}

// sendVerificationMail crée un token de vérification (TTL 24h) et envoie le
// lien par mail. Best-effort : toute erreur est loggée, jamais propagée (ne
// casse ni le register ni le resend).
func (s *AuthService) sendVerificationMail(u *models.User) {
	if s.mailer == nil {
		slog.Debug("mail désactivé : e-mail de vérification non envoyé", "user_id", u.ID)
		return
	}

	raw, err := s.createAccountToken(u.ID, purposeVerify, verifyTokenTTL)
	if err != nil {
		slog.Error("création token vérification échouée", "user_id", u.ID, "error", err)
		return
	}

	link := fmt.Sprintf("%s/verify-email?token=%s",
		strings.TrimRight(s.appBaseURL, "/"), url.QueryEscape(raw))

	subject := "Confirme ton adresse e-mail — Breezy"
	text := fmt.Sprintf(
		"Bienvenue sur Breezy !\n\n"+
			"Confirme ton adresse e-mail en ouvrant ce lien :\n%s\n\n"+
			"Ce lien expire dans 24 heures. Si tu n'es pas à l'origine de cette "+
			"inscription, ignore ce message.",
		link)
	htmlBody := brandedEmailHTML(s.appBaseURL,
		"Bienvenue sur Breezy 👋",
		"Plus qu'une étape : confirme ton adresse e-mail pour activer ton compte et rejoindre la conversation.",
		"",
		"Vérifier mon adresse e-mail", link,
		"Ce lien expire dans 24 heures. Si tu n'es pas à l'origine de cette inscription, ignore simplement ce message.")

	if err := s.mailer.Send(u.Email, subject, htmlBody, text); err != nil {
		slog.Warn("envoi e-mail vérification échoué (best-effort)", "user_id", u.ID, "error", err)
	}
}

// ForgotPassword déclenche l'envoi d'un mail de réinitialisation. ANTI-
// ÉNUMÉRATION : renvoie TOUJOURS nil — l'appelant ne peut pas distinguer un
// compte inexistant ou inactif. Le mail n'est envoyé que pour un compte actif.
func (s *AuthService) ForgotPassword(email string) error {
	const q = `
		SELECT id, email, role, is_active, email_verified, created_at
		FROM credentials WHERE email = $1`

	u := &models.User{}
	err := s.db.QueryRow(q, email).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // compte inexistant : no-op silencieux (anti-énumération).
	}
	if err != nil {
		slog.Error("forgot password : lecture DB échouée", "error", err)
		return nil
	}
	if !u.IsActive {
		return nil // compte désactivé : pas de reset.
	}

	s.sendResetMail(u)
	return nil
}

// ResetPassword consomme un token de reset puis remplace le mot de passe. Le
// lien reset prouvant la possession de l'adresse, on en profite pour marquer
// l'e-mail vérifié (débloque un compte non vérifié). Toutes les sessions sont
// révoquées (DELETE refresh_tokens). Renvoie ErrInvalidToken si le token est
// inconnu / expiré / déjà utilisé.
func (s *AuthService) ResetPassword(rawToken, newPassword string) error {
	userID, err := s.consumeAccountToken(rawToken, purposeReset)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash du nouveau mot de passe : %w", err)
	}

	if _, err := s.db.Exec(
		`UPDATE credentials SET password = $1, email_verified = true WHERE id = $2`,
		string(hash), userID,
	); err != nil {
		return fmt.Errorf("mise à jour du mot de passe : %w", err)
	}

	// Révoque toutes les sessions ouvertes : un reset doit déconnecter partout.
	if _, err := s.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("révocation des sessions : %w", err)
	}
	return nil
}

// sendResetMail crée un token de reset (TTL 1h) et envoie le lien par mail.
// Best-effort : toute erreur est loggée, jamais propagée (ne casse pas le flux
// forgot, qui reste anti-énumération).
func (s *AuthService) sendResetMail(u *models.User) {
	if s.mailer == nil {
		slog.Debug("mail désactivé : e-mail reset non envoyé", "user_id", u.ID)
		return
	}

	raw, err := s.createAccountToken(u.ID, purposeReset, resetTokenTTL)
	if err != nil {
		slog.Error("création token reset échouée", "user_id", u.ID, "error", err)
		return
	}

	link := fmt.Sprintf("%s/reset-password?token=%s",
		strings.TrimRight(s.appBaseURL, "/"), url.QueryEscape(raw))

	subject := "Réinitialise ton mot de passe — Breezy"
	text := fmt.Sprintf(
		"Tu as demandé à réinitialiser ton mot de passe Breezy.\n\n"+
			"Choisis un nouveau mot de passe en ouvrant ce lien :\n%s\n\n"+
			"Ce lien expire dans 1 heure. Si tu n'es pas à l'origine de cette "+
			"demande, ignore ce message : ton mot de passe reste inchangé.",
		link)
	htmlBody := brandedEmailHTML(s.appBaseURL,
		"Réinitialise ton mot de passe 🔒",
		"Tu as demandé à changer ton mot de passe Breezy. Choisis-en un nouveau en un clic — c'est rapide et sécurisé.",
		"",
		"Choisir un nouveau mot de passe", link,
		"Ce lien expire dans 1 heure. Si tu n'es pas à l'origine de cette demande, ignore ce message : ton mot de passe reste inchangé.")

	if err := s.mailer.Send(u.Email, subject, htmlBody, text); err != nil {
		slog.Warn("envoi e-mail reset échoué (best-effort)", "user_id", u.ID, "error", err)
	}
}

// createAccountToken invalide d'abord les tokens non consommés du même
// (user_id, purpose), puis crée un nouveau jeton opaque haché (SHA-256) et
// retourne sa valeur EN CLAIR (à insérer dans le lien). TTL via expires_at.
func (s *AuthService) createAccountToken(userID, purpose string, ttl time.Duration) (string, error) {
	if _, err := s.db.Exec(
		`DELETE FROM account_tokens WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL`,
		userID, purpose,
	); err != nil {
		return "", fmt.Errorf("invalidation anciens tokens : %w", err)
	}

	raw, err := randomToken()
	if err != nil {
		return "", err
	}

	const q = `INSERT INTO account_tokens (user_id, purpose, token_hash, expires_at) VALUES ($1, $2, $3, $4)`
	if _, err := s.db.Exec(q, userID, purpose, hashToken(raw), time.Now().Add(ttl)); err != nil {
		return "", fmt.Errorf("création account token : %w", err)
	}
	return raw, nil
}

// consumeAccountToken valide un token (lookup par hash) puis le marque consommé
// (used_at = NOW()). Usage unique : refuse un token inconnu, déjà utilisé,
// expiré ou de mauvais purpose. Renvoie le user_id associé.
func (s *AuthService) consumeAccountToken(rawToken, purpose string) (string, error) {
	const q = `
		SELECT id, user_id, expires_at, used_at
		FROM account_tokens WHERE token_hash = $1 AND purpose = $2`

	var id, userID string
	var expiresAt time.Time
	var usedAt sql.NullTime
	err := s.db.QueryRow(q, hashToken(rawToken), purpose).
		Scan(&id, &userID, &expiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("lecture account token : %w", err)
	}
	if usedAt.Valid || time.Now().After(expiresAt) {
		return "", ErrInvalidToken
	}

	if _, err := s.db.Exec(`UPDATE account_tokens SET used_at = NOW() WHERE id = $1`, id); err != nil {
		return "", fmt.Errorf("consommation account token : %w", err)
	}
	return userID, nil
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

	// email_verified=true : l'admin de démo n'a pas de vraie boîte mail, on le
	// garde donc utilisable malgré le blocage login des comptes non vérifiés
	// (Phase 1). DO UPDATE rend le marquage idempotent même sur une base où
	// l'admin existait déjà avant l'ajout de la colonne.
	const q = `
		INSERT INTO credentials (id, email, password, role, email_verified)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (id) DO UPDATE SET email_verified = true`

	if _, err := s.db.Exec(q, defaultAdminID, email, string(hash), models.RoleAdmin); err != nil {
		return fmt.Errorf("seed admin : %w", err)
	}
	return nil
}

// ─── Administration (réservé aux comptes admin, garde côté handler/route) ───

// ListUsers retourne une page de comptes (annuaire admin). `query` filtre par
// email (sous-chaîne, insensible à la casse) ; vide = tous les comptes. Inclut
// les comptes désactivés (l'admin doit voir les bannis).
func (s *AuthService) ListUsers(limit, offset int, query string) ([]models.User, error) {
	const q = `
		SELECT id, email, role, is_active, deactivated_at, created_at
		FROM credentials
		WHERE ($1 = '' OR email ILIKE '%' || $1 || '%')
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.Query(q, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("liste comptes : %w", err)
	}
	defer func() { _ = rows.Close() }()

	users := make([]models.User, 0, limit)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.DeactivatedAt, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("lecture compte : %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours comptes : %w", err)
	}
	return users, nil
}

// SetRole change le rôle d'un compte (source de vérité du rôle = ce service).
// La prise d'effet côté JWT est différée au prochain /refresh de l'utilisateur
// (le token courant porte l'ancien rôle jusqu'à expiration ≤ jwtExpiry).
func (s *AuthService) SetRole(id, role string) error {
	if !validRole(role) {
		return ErrInvalidRole
	}
	res, err := s.db.Exec(`UPDATE credentials SET role = $2 WHERE id = $1`, id, role)
	if err != nil {
		return fmt.Errorf("changement de rôle : %w", err)
	}
	return errIfNoRows(res)
}

// RoleOf renvoie le rôle courant d'un compte (ErrUserNotFound si absent). Sert
// à protéger la hiérarchie : un modérateur ne doit bannir qu'un simple
// utilisateur, pas un autre modérateur ni un admin.
func (s *AuthService) RoleOf(id string) (string, error) {
	var role string
	err := s.db.QueryRow(`SELECT role FROM credentials WHERE id = $1`, id).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lecture du rôle : %w", err)
	}
	return role, nil
}

// SetActive active/désactive un compte. Désactiver = bannir : bloque le login
// ET le /refresh (cf. Login/Refresh), et on révoque immédiatement les refresh
// tokens du compte pour tuer sa session courante au plus vite (sa session ne
// survit alors qu'à l'access token en cours, ≤ jwtExpiry).
func (s *AuthService) SetActive(id string, active bool) error {
	// deactivated_at : posée au bannissement (point de départ de la purge RGPD à
	// 5 ans), effacée à la réactivation. `CASE` pour ne pas écraser une date déjà
	// posée si on rebannit (improbable mais sûr).
	res, err := s.db.Exec(`
		UPDATE credentials
		SET is_active = $2,
		    deactivated_at = CASE
		        WHEN $2 = false AND deactivated_at IS NULL THEN NOW()
		        WHEN $2 = true THEN NULL
		        ELSE deactivated_at
		    END
		WHERE id = $1`, id, active)
	if err != nil {
		return fmt.Errorf("changement d'état du compte : %w", err)
	}
	if err := errIfNoRows(res); err != nil {
		return err
	}
	if !active {
		// Révocation best-effort : l'échec ne doit pas annuler le bannissement.
		_, _ = s.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, id)
	}
	return nil
}

// DeleteAccount efface DÉFINITIVEMENT les identifiants d'un compte (effacement
// RGPD) : ses refresh tokens puis sa ligne `credentials`. La purge des données
// applicatives (profil, posts, messages, médias, graphe social) est orchestrée
// par l'appelant (admin) sur les autres services. ErrUserNotFound si absent.
func (s *AuthService) DeleteAccount(id string) error {
	// Best-effort sur les tokens (révocation), impératif sur les credentials.
	_, _ = s.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, id)
	res, err := s.db.Exec(`DELETE FROM credentials WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("suppression du compte : %w", err)
	}
	return errIfNoRows(res)
}

// systemActorID : id « système » porté par le token admin minté pour le
// balayage automatique (aucun compte réel — sert uniquement aux gardes admin
// des services en aval).
const systemActorID = "00000000-0000-0000-0000-000000000000"

// ListBannedBefore renvoie les comptes bannis (is_active=false) dont le
// bannissement remonte à avant `before`, du plus ancien au plus récent.
func (s *AuthService) ListBannedBefore(before time.Time, limit int) ([]models.User, error) {
	const q = `
		SELECT id, email, role, is_active, deactivated_at, created_at
		FROM credentials
		WHERE is_active = false AND deactivated_at IS NOT NULL AND deactivated_at < $1
		ORDER BY deactivated_at ASC
		LIMIT $2`
	rows, err := s.db.Query(q, before, limit)
	if err != nil {
		return nil, fmt.Errorf("liste comptes bannis : %w", err)
	}
	defer func() { _ = rows.Close() }()

	users := make([]models.User, 0, limit)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.DeactivatedAt, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("lecture compte : %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// mintSystemAdminToken signe un JWT admin éphémère pour les appels
// serveur-à-serveur du balayage (auth est l'émetteur de tokens).
func (s *AuthService) mintSystemAdminToken() (string, error) {
	return s.GenerateToken(&models.User{ID: systemActorID, Email: "system@auth", Role: models.RoleAdmin})
}

// SweepBannedAccounts efface (RGPD) les comptes bannis depuis plus de `after` :
// purge des données applicatives sur les autres services (token admin minté),
// puis suppression des identifiants. Renvoie le nombre de comptes effacés.
func (s *AuthService) SweepBannedAccounts(ctx context.Context, e *eraser.Eraser, after time.Duration) (int, error) {
	if after <= 0 {
		return 0, nil
	}
	banned, err := s.ListBannedBefore(time.Now().Add(-after), 100)
	if err != nil {
		return 0, err
	}
	if len(banned) == 0 {
		return 0, nil
	}
	bearer, err := s.mintSystemAdminToken()
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, u := range banned {
		if failed := e.Erase(ctx, bearer, u.ID); len(failed) > 0 {
			slog.Warn("purge RGPD : services non effacés", "user_id", u.ID, "failed_services", failed)
		}
		if err := s.DeleteAccount(u.ID); err != nil {
			slog.Error("purge RGPD : suppression credentials échouée", "user_id", u.ID, "error", err)
			continue
		}
		purged++
	}
	return purged, nil
}

// RunAccountPurgeSweeper balaye périodiquement les comptes bannis à purger
// (RGPD) jusqu'à annulation du contexte. À lancer en goroutine. No-op si
// rétention ou intervalle <= 0.
func (s *AuthService) RunAccountPurgeSweeper(ctx context.Context, e *eraser.Eraser, after, interval time.Duration) {
	if after <= 0 || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if n, err := s.SweepBannedAccounts(ctx, e, after); err != nil {
			slog.Error("purge RGPD : balayage échoué", "error", err)
		} else if n > 0 {
			slog.Info("purge RGPD : comptes bannis effacés", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// validRole vérifie qu'un rôle fait partie de l'enum autorisé.
func validRole(role string) bool {
	switch role {
	case models.RoleUser, models.RoleModerator, models.RoleAdmin:
		return true
	default:
		return false
	}
}

// errIfNoRows mappe « 0 ligne affectée » vers ErrUserNotFound.
func errIfNoRows(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("lignes affectées : %w", err)
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GenerateToken signe un JWT HS256 pour l'utilisateur donné.
func (s *AuthService) GenerateToken(u *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:             u.ID,
		Email:              u.Email,
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword,
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
