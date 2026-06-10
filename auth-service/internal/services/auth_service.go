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
	"log"
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
	ErrInvalidCredentials  = errors.New("email ou mot de passe invalide")
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
}

// New construit le service. mailer peut être nil (mail non configuré) : l'envoi
// devient alors un no-op loggé et auth reste pleinement fonctionnel.
func New(db *sql.DB, jwtSecret string, jwtExpiry, refreshExpiry time.Duration, mailer Mailer, appBaseURL string) *AuthService {
	return &AuthService{
		db:            db,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
		mailer:        mailer,
		appBaseURL:    appBaseURL,
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

// Login vérifie les credentials et retourne un access token + un refresh
// token + l'utilisateur.
func (s *AuthService) Login(email, password string) (string, string, *models.User, error) {
	const q = `
		SELECT id, email, password, role, is_active, email_verified, created_at
		FROM credentials WHERE email = $1`

	u := &models.User{}
	err := s.db.QueryRow(q, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.EmailVerified, &u.CreatedAt)
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
	// Blocage dur : un compte non vérifié ne peut pas se connecter (aucun token
	// émis). Vérifié APRÈS le bcrypt pour ne pas révéler l'existence du compte.
	if !u.EmailVerified {
		return "", "", nil, ErrEmailNotVerified
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
		RETURNING id, email, role, is_active, email_verified, created_at`
	u := &models.User{}
	if err := s.db.QueryRow(q, userID).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified, &u.CreatedAt); err != nil {
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
		log.Printf("[auth-service] resend vérif (lecture) pour %s : %v", email, err)
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
		log.Printf("[auth-service] mail désactivé : vérif pour %s non envoyée", u.Email)
		return
	}

	raw, err := s.createAccountToken(u.ID, purposeVerify, verifyTokenTTL)
	if err != nil {
		log.Printf("[auth-service] création token vérif pour %s : %v", u.Email, err)
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
		"Vérifier mon adresse e-mail", link,
		"Ce lien expire dans 24 heures. Si tu n'es pas à l'origine de cette inscription, ignore simplement ce message.")

	if err := s.mailer.Send(u.Email, subject, htmlBody, text); err != nil {
		log.Printf("[auth-service] envoi mail vérif à %s : %v", u.Email, err)
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
		log.Printf("[auth-service] forgot password (lecture) pour %s : %v", email, err)
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
		log.Printf("[auth-service] mail désactivé : reset pour %s non envoyé", u.Email)
		return
	}

	raw, err := s.createAccountToken(u.ID, purposeReset, resetTokenTTL)
	if err != nil {
		log.Printf("[auth-service] création token reset pour %s : %v", u.Email, err)
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
		"Choisir un nouveau mot de passe", link,
		"Ce lien expire dans 1 heure. Si tu n'es pas à l'origine de cette demande, ignore ce message : ton mot de passe reste inchangé.")

	if err := s.mailer.Send(u.Email, subject, htmlBody, text); err != nil {
		log.Printf("[auth-service] envoi mail reset à %s : %v", u.Email, err)
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
			log.Printf("[account-purge] %s : échec partiel %v", u.ID, failed)
		}
		if err := s.DeleteAccount(u.ID); err != nil {
			log.Printf("[account-purge] suppression credentials %s : %v", u.ID, err)
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
			log.Printf("[account-purge] balayage : %v", err)
		} else if n > 0 {
			log.Printf("[account-purge] %d comptes bannis effacés (RGPD)", n)
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
