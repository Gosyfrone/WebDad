// Package oauth implémente le flux OIDC Authorization Code pour la connexion
// via des fournisseurs externes (Google, Microsoft). L'échange code→tokens et
// la vérification de l'ID token (signature via JWKS, issuer, audience) se font
// ICI, côté serveur — jamais côté front. Le service n'expose que l'URL
// d'autorisation (avec un state anti-CSRF) et un endpoint d'échange.
//
// Ajouter un provider = une entrée dans `defs` (issuer + scopes) + ses
// variables d'environnement (cf. package config). Microsoft est déjà câblé :
// il suffit de renseigner MICROSOFT_CLIENT_ID / MICROSOFT_CLIENT_SECRET.
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// ErrUnknownProvider : provider non supporté ou non configuré (clé absente).
var ErrUnknownProvider = errors.New("provider OAuth inconnu ou non configuré")

// providerDef décrit les aspects STATIQUES d'un provider OIDC. Le tenant
// Microsoft (et donc son issuer) est résolu à la construction via un override
// de credentials.
type providerDef struct {
	issuer string   // URL de l'émetteur (sert au discovery .well-known/openid-configuration)
	scopes []string // scopes OAuth demandés (openid + identité)
}

// defs : config générique des providers. Microsoft = une entrée, comme Google.
var defs = map[string]providerDef{
	"google": {
		issuer: "https://accounts.google.com",
		scopes: []string{oidc.ScopeOpenID, "email", "profile"},
	},
	"microsoft": {
		// issuer par défaut (multi-tenant) ; surchargé par MICROSOFT_TENANT.
		issuer: "https://login.microsoftonline.com/common/v2.0",
		scopes: []string{oidc.ScopeOpenID, "email", "profile"},
	},
}

// Credentials : secrets d'un provider (issus de la config/env). Issuer permet
// de surcharger l'émetteur par défaut (ex. tenant Microsoft précis).
type Credentials struct {
	ClientID     string
	ClientSecret string
	Issuer       string // override optionnel de l'issuer
}

// Identity : informations extraites de l'ID token APRÈS vérification.
type Identity struct {
	Subject       string // claim `sub` — identifiant stable côté provider
	Email         string // claim `email`
	EmailVerified bool   // claim `email_verified`
}

// Provider : un provider OIDC prêt à l'emploi (config OAuth + vérificateur JWKS).
type Provider struct {
	name     string
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// AuthURL construit l'URL d'autorisation pour le `state` donné (anti-CSRF).
func (p *Provider) AuthURL(state string) string {
	return p.oauth.AuthCodeURL(state)
}

// Exchange échange le `code` reçu au callback contre les tokens du provider,
// vérifie l'ID token (signature, issuer, audience=client_id) et en extrait
// l'identité (sub + email).
func (p *Provider) Exchange(ctx context.Context, code string) (*Identity, error) {
	tok, err := p.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("échange code OAuth : %w", err)
	}

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
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("lecture des claims id_token : %w", err)
	}

	return &Identity{
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, nil
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

	issuer := def.issuer
	if cred.Issuer != "" {
		issuer = cred.Issuer
	}

	oidcProvider, err := oidc.NewProvider(r.ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discovery OIDC %s : %w", name, err)
	}

	p := &Provider{
		name: name,
		oauth: &oauth2.Config{
			ClientID:     cred.ClientID,
			ClientSecret: cred.ClientSecret,
			Endpoint:     oidcProvider.Endpoint(),
			// Redirect URI front : doit correspondre à celui déclaré chez le
			// provider, ex. http://localhost:3000/auth/callback/google.
			RedirectURL: r.redirectBase + "/auth/callback/" + name,
			Scopes:      def.scopes,
		},
		// audience = ClientID : rejette un id_token émis pour une autre app.
		verifier: oidcProvider.Verifier(&oidc.Config{ClientID: cred.ClientID}),
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
