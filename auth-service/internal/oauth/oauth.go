// Package oauth implémente la connexion via des fournisseurs externes. Deux
// familles de providers sont gérées, derrière une même interface (AuthURL +
// Exchange) :
//
//   - kindOIDC   (Google) : flux OIDC Authorization Code. L'échange
//     code→tokens renvoie un id_token signé que l'on VÉRIFIE ici, côté serveur
//     (signature via JWKS, issuer, audience), puis on en lit les claims.
//   - kindOAuth2 (Facebook, Spotify, GitHub) : OAuth2 « classique » sans
//     id_token. On échange le code contre un access_token, puis on appelle
//     l'endpoint userinfo du provider (Graph API, /v1/me, /user…) pour récupérer
//     {sub, email}. L'email y est considéré comme vérifié : le provider a
//     authentifié l'utilisateur et nous renvoie une adresse confirmée de son
//     côté. Certains providers n'exposent pas l'email dans userinfo (GitHub le
//     masque si privé) : un `emailsURL` optionnel sert alors de repli pour
//     récupérer l'adresse primaire vérifiée.
//
// Dans tous les cas, l'échange et la lecture d'identité se font ICI, jamais
// côté front. Le service n'expose que l'URL d'autorisation (avec un state
// anti-CSRF) et un endpoint d'échange.
//
// Ajouter un provider = une entrée dans `defs` (kind + issuer OU
// authURL/tokenURL/userInfoURL) + ses variables d'environnement (cf. config).
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// ErrUnknownProvider : provider non supporté ou non configuré (clé absente).
var ErrUnknownProvider = errors.New("provider OAuth inconnu ou non configuré")

// httpClient sert les appels userinfo (kindOAuth2), avec un timeout borné pour
// ne jamais bloquer une requête de login sur un provider lent.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// providerKind distingue les deux familles de flux (cf. doc du package).
type providerKind int

const (
	kindOIDC providerKind = iota
	kindOAuth2
)

// providerDef décrit les aspects STATIQUES d'un provider.
type providerDef struct {
	kind   providerKind
	scopes []string // scopes OAuth demandés

	// kindOIDC : URL de l'émetteur (sert au discovery .well-known).
	issuer string

	// kindOAuth2 : endpoints fixes + endpoint userinfo + extraction d'identité.
	authURL     string
	tokenURL    string
	userInfoURL string
	userInfoMap func(map[string]any) Identity // mappe la réponse userinfo → Identity

	// emailsURL (optionnel, kindOAuth2) : endpoint listant les adresses email,
	// utilisé en repli quand userInfoURL n'expose pas l'email (cas GitHub : email
	// privé). Appelé uniquement si l'identité extraite n'a pas d'email. Réponse
	// attendue : tableau d'objets {email, primary, verified}.
	emailsURL string
}

// defs : config générique des providers.
var defs = map[string]providerDef{
	"google": {
		kind:   kindOIDC,
		issuer: "https://accounts.google.com",
		scopes: []string{oidc.ScopeOpenID, "email", "profile"},
	},
	"github": {
		kind:        kindOAuth2,
		scopes:      []string{"read:user", "user:email"},
		authURL:     "https://github.com/login/oauth/authorize",
		tokenURL:    "https://github.com/login/oauth/access_token",
		userInfoURL: "https://api.github.com/user",
		// `id` GitHub est un nombre JSON → idField. `email` peut être null (privé)
		// → repli via emailsURL ci-dessous.
		userInfoMap: func(m map[string]any) Identity {
			return Identity{Subject: idField(m, "id"), Email: strField(m, "email"), EmailVerified: true}
		},
		emailsURL: "https://api.github.com/user/emails",
	},
	"facebook": {
		kind:        kindOAuth2,
		scopes:      []string{"email"},
		authURL:     "https://www.facebook.com/v21.0/dialog/oauth",
		tokenURL:    "https://graph.facebook.com/v21.0/oauth/access_token",
		userInfoURL: "https://graph.facebook.com/me?fields=id,email",
		userInfoMap: func(m map[string]any) Identity {
			return Identity{Subject: strField(m, "id"), Email: strField(m, "email"), EmailVerified: true}
		},
	},
	"spotify": {
		kind:        kindOAuth2,
		scopes:      []string{"user-read-email"},
		authURL:     "https://accounts.spotify.com/authorize",
		tokenURL:    "https://accounts.spotify.com/api/token",
		userInfoURL: "https://api.spotify.com/v1/me",
		userInfoMap: func(m map[string]any) Identity {
			return Identity{Subject: strField(m, "id"), Email: strField(m, "email"), EmailVerified: true}
		},
	},
}

// strField lit une valeur string d'une map décodée depuis du JSON (absente ou
// d'un autre type → "").
func strField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// idField lit un identifiant qui peut être une string OU un nombre JSON (cas
// GitHub : `id` numérique, décodé en float64). Renvoie "" si absent/autre type.
func idField(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	}
	return ""
}

// Credentials : secrets d'un provider (issus de la config/env).
type Credentials struct {
	ClientID     string
	ClientSecret string
}

// Identity : informations extraites APRÈS vérification (OIDC) ou via userinfo
// (OAuth2).
type Identity struct {
	Subject       string // identifiant stable côté provider (claim `sub` / champ `id`)
	Email         string // email
	EmailVerified bool   // email_verified (OIDC) ou true de confiance (OAuth2)
}

// Provider : un provider prêt à l'emploi (config OAuth + vérificateur OIDC OU
// endpoint userinfo selon le kind).
type Provider struct {
	name  string
	kind  providerKind
	oauth *oauth2.Config

	// kindOIDC
	verifier *oidc.IDTokenVerifier

	// kindOAuth2
	userInfoURL string
	userInfoMap func(map[string]any) Identity
	emailsURL   string // repli email (optionnel, cf. providerDef.emailsURL)
}

// AuthURL construit l'URL d'autorisation pour le `state` donné (anti-CSRF).
func (p *Provider) AuthURL(state string) string {
	return p.oauth.AuthCodeURL(state)
}

// Exchange échange le `code` reçu au callback contre les tokens du provider,
// puis en extrait l'identité — par vérification de l'id_token (OIDC) ou par
// appel de l'endpoint userinfo (OAuth2).
func (p *Provider) Exchange(ctx context.Context, code string) (*Identity, error) {
	tok, err := p.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("échange code OAuth : %w", err)
	}

	if p.kind == kindOIDC {
		return p.identityFromIDToken(ctx, tok)
	}
	return p.identityFromUserInfo(ctx, tok)
}

// identityFromIDToken : flux OIDC — vérifie l'id_token (signature JWKS, issuer,
// audience=client_id) et en lit les claims email / email_verified.
func (p *Provider) identityFromIDToken(ctx context.Context, tok *oauth2.Token) (*Identity, error) {
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return nil, errors.New("réponse du provider sans id_token")
	}

	idToken, err := p.verifier.Verify(ctx, rawID)
	if err != nil {
		return nil, fmt.Errorf("vérification id_token : %w", err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"` // pointeur : absent ≠ false
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("lecture des claims id_token : %w", err)
	}

	verified := false // par défaut quand le claim est absent
	if claims.EmailVerified != nil {
		verified = *claims.EmailVerified
	}

	return &Identity{
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: verified,
	}, nil
}

// identityFromUserInfo : flux OAuth2 — appelle l'endpoint userinfo du provider
// avec l'access_token, puis mappe la réponse JSON → Identity.
func (p *Provider) identityFromUserInfo(ctx context.Context, tok *oauth2.Token) (*Identity, error) {
	var raw map[string]any
	if err := p.authedGetJSON(ctx, p.userInfoURL, tok, &raw); err != nil {
		return nil, fmt.Errorf("userinfo %s : %w", p.name, err)
	}

	id := p.userInfoMap(raw)
	if id.Subject == "" {
		return nil, errors.New("userinfo sans identifiant stable")
	}

	// Repli email : certains providers (GitHub) ne renvoient pas l'email dans
	// userinfo s'il est privé. On le récupère alors via emailsURL.
	if id.Email == "" && p.emailsURL != "" {
		email, err := p.fetchPrimaryEmail(ctx, tok)
		if err != nil {
			return nil, fmt.Errorf("emails %s : %w", p.name, err)
		}
		id.Email = email
	}
	return &id, nil
}

// fetchPrimaryEmail appelle l'endpoint emailsURL et retourne l'adresse primaire
// ET vérifiée (à défaut la première vérifiée). Erreur si aucune adresse
// vérifiée : on refuse de provisionner un compte sur un email non confirmé.
func (p *Provider) fetchPrimaryEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := p.authedGetJSON(ctx, p.emailsURL, tok, &emails); err != nil {
		return "", err
	}

	fallback := ""
	for _, e := range emails {
		if !e.Verified {
			continue
		}
		if e.Primary {
			return e.Email, nil
		}
		if fallback == "" {
			fallback = e.Email
		}
	}
	if fallback == "" {
		return "", errors.New("aucune adresse email vérifiée")
	}
	return fallback, nil
}

// authedGetJSON exécute un GET authentifié (Bearer) et décode la réponse JSON
// dans `out`. Pose User-Agent (exigé par api.github.com) et Accept JSON.
func (p *Provider) authedGetJSON(ctx context.Context, url string, tok *oauth2.Token, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("requête : %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "breezy-auth")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("appel : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("statut %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("décodage : %w", err)
	}
	return nil
}

// Options configure le Registry.
type Options struct {
	RedirectBaseURL string                 // base du redirect URI front (sans slash final)
	Providers       map[string]Credentials // credentials par provider (ClientID vide = ignoré)
}

// Registry construit PARESSEUSEMENT et met en cache les providers configurés.
// La construction effective (discovery OIDC = appel réseau) est différée au
// premier usage : le service démarre même si un provider est momentanément
// injoignable, et ne dépend pas du réseau au boot.
type Registry struct {
	ctx          context.Context
	redirectBase string
	creds        map[string]Credentials
	mu           sync.Mutex
	cache        map[string]*Provider
}

// NewRegistry retient les providers dont le ClientID est renseigné.
func NewRegistry(ctx context.Context, opts Options) *Registry {
	creds := make(map[string]Credentials)
	for name, c := range opts.Providers {
		if c.ClientID == "" {
			continue // provider non configuré → absent du registre
		}
		creds[name] = c
	}
	return &Registry{
		ctx:          ctx,
		redirectBase: strings.TrimRight(opts.RedirectBaseURL, "/"),
		creds:        creds,
		cache:        make(map[string]*Provider),
	}
}

// Get retourne le provider demandé, en le construisant au premier appel.
// Renvoie ErrUnknownProvider si le provider n'est pas supporté/configuré.
func (r *Registry) Get(name string) (*Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.cache[name]; ok {
		return p, nil
	}

	cred, ok := r.creds[name]
	if !ok {
		return nil, ErrUnknownProvider
	}
	def, ok := defs[name]
	if !ok {
		return nil, ErrUnknownProvider
	}

	// Redirect URI front : doit correspondre à celui déclaré chez le provider,
	// ex. http://localhost:3000/auth/callback/github.
	redirectURL := r.redirectBase + "/auth/callback/" + name

	var p *Provider
	switch def.kind {
	case kindOIDC:
		oidcProvider, err := oidc.NewProvider(r.ctx, def.issuer)
		if err != nil {
			return nil, fmt.Errorf("discovery OIDC %s : %w", name, err)
		}
		p = &Provider{
			name: name,
			kind: kindOIDC,
			oauth: &oauth2.Config{
				ClientID:     cred.ClientID,
				ClientSecret: cred.ClientSecret,
				Endpoint:     oidcProvider.Endpoint(),
				RedirectURL:  redirectURL,
				Scopes:       def.scopes,
			},
			// audience = ClientID : rejette un id_token émis pour une autre app.
			verifier: oidcProvider.Verifier(&oidc.Config{ClientID: cred.ClientID}),
		}
	case kindOAuth2:
		p = &Provider{
			name: name,
			kind: kindOAuth2,
			oauth: &oauth2.Config{
				ClientID:     cred.ClientID,
				ClientSecret: cred.ClientSecret,
				Endpoint:     oauth2.Endpoint{AuthURL: def.authURL, TokenURL: def.tokenURL},
				RedirectURL:  redirectURL,
				Scopes:       def.scopes,
			},
			userInfoURL: def.userInfoURL,
			userInfoMap: def.userInfoMap,
			emailsURL:   def.emailsURL,
		}
	default:
		return nil, ErrUnknownProvider
	}

	r.cache[name] = p
	return p, nil
}

// NewState génère un state anti-CSRF opaque (256 bits, base64url). Le front le
// stocke et le revérifie au callback avant d'appeler /exchange.
func NewState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("génération state : %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
