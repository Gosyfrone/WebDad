-- Schéma canonique du service auth — SOURCE DE VÉRITÉ UNIQUE.
--
-- Embarqué dans le binaire (go:embed) et appliqué au démarrage de façon
-- idempotente. Le service crée et maintient donc sa propre base : il tourne
-- en standalone (`make run` contre un Postgres nu) comme en stack Docker,
-- sans dépendre d'un script d'init externe.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum des rôles (CREATE TYPE n'a pas d'IF NOT EXISTS → garde via DO).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS credentials (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email          VARCHAR(255) NOT NULL UNIQUE,
    password       VARCHAR(255) NOT NULL,          -- hash bcrypt
    role           user_role NOT NULL DEFAULT 'user',
    is_active      BOOLEAN NOT NULL DEFAULT true,
    email_verified BOOLEAN NOT NULL DEFAULT false, -- vérif e-mail (blocage login en Phase 1)
    -- deactivated_at : date du bannissement (is_active passé à false). Sert de
    -- point de départ à la purge RGPD automatique des comptes bannis (5 ans).
    -- NULL quand le compte est actif.
    deactivated_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE credentials ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ;
-- must_change_password : compte créé par un admin avec un mot de passe temporaire.
-- Tant qu'il est à true, le front impose un changement de mot de passe bloquant
-- à la première connexion (cf. POST /auth/password/change qui le repasse à false).
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false;
-- Changement d'adresse : l'adresse courante reste valable tant que la nouvelle
-- n'a pas été prouvée via le lien envoyé dans sa boîte.
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS pending_email VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_credentials_pending_email
    ON credentials(pending_email) WHERE pending_email IS NOT NULL;

-- MFA TOTP (optionnelle, opt-in). mfa_enabled a un DÉFAUT → rétro-compatible
-- (aucun backfill requis, règle 5b) : en PostgreSQL, ADD COLUMN ... DEFAULT false
-- pose la valeur sur TOUTES les lignes existantes au déploiement → tous les
-- comptes déjà en prod démarrent avec la MFA DÉSACTIVÉE (aucun verrouillage
-- possible). NB : surtout pas d'UPDATE inconditionnel ici, il réinitialiserait
-- la MFA des comptes qui l'auront activée à chaque redémarrage.
-- mfa_secret est NULLABLE et contient le
-- secret TOTP CHIFFRÉ at-rest (AES-256-GCM, clé MFA_ENCRYPTION_KEY) :
--   - setup non confirmé  → secret présent + mfa_enabled = false
--   - MFA active          → secret présent + mfa_enabled = true
--   - MFA désactivée      → secret NULL    + mfa_enabled = false
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS mfa_secret TEXT;

-- Acceptation des CGU (consentement versionné). terms_accepted_version porte la
-- version des CGU acceptée par le compte ; CurrentTermsVersion (côté Go) est la
-- version EN VIGUEUR. Si version_acceptée < version_en_vigueur → le front impose
-- une modale d'acceptation bloquante post-login (claim JWT `terms_accepted`).
-- Rétro-compatible (règle 5b) : ADD COLUMN ... DEFAULT 0 pose 0 sur TOUTES les
-- lignes existantes au déploiement → tous les comptes déjà en prod (qui n'ont
-- jamais consenti explicitement) repassent sous la version courante et doivent
-- accepter à leur prochaine connexion. Surtout PAS d'UPDATE inconditionnel ici :
-- il ré-imposerait l'acceptation à chaque redémarrage. Rebumper CurrentTermsVersion
-- lors d'une MAJ des CGU re-déclenche le consentement de tout le monde.
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS terms_accepted_version INT NOT NULL DEFAULT 0;
-- terms_accepted_at : horodatage du dernier consentement (audit RGPD). NULL tant
-- qu'aucune acceptation explicite n'a eu lieu.
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS terms_accepted_at TIMESTAMPTZ;

-- Refresh tokens (préparé pour la feature bonus).
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tokens à usage unique pour la vérification d'e-mail et le reset de mot de
-- passe. Jeton opaque aléatoire, stocké HACHÉ (SHA-256 hex, jamais en clair),
-- consommé une seule fois (`used_at`), avec TTL (`expires_at`). Même principe
-- que refresh_tokens : une fuite de la table ne livre aucun token utilisable.
CREATE TABLE IF NOT EXISTS account_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    purpose     VARCHAR(16) NOT NULL CHECK (purpose IN ('verify', 'reset', 'email_change', 'mfa_challenge')),
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,                    -- NULL tant que non consommé
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Une base créée avant le flux de changement d'e-mail possède encore la
-- contrainte à deux valeurs. La remplacer au boot est idempotent.
ALTER TABLE account_tokens DROP CONSTRAINT IF EXISTS account_tokens_purpose_check;
ALTER TABLE account_tokens ADD CONSTRAINT account_tokens_purpose_check
    CHECK (purpose IN ('verify', 'reset', 'email_change', 'mfa_challenge'));

-- ─── Connexion via fournisseurs externes (Google / GitHub) ──
-- ALTER idempotents : la base existante est migrée au boot sans script externe.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'auth_provider') THEN
        CREATE TYPE auth_provider AS ENUM ('local', 'google', 'microsoft');
    END IF;
END$$;

-- Nouveaux providers : ADD VALUE IF NOT EXISTS est idempotent et hors
-- transaction (EnsureSchema applique le fichier en autocommit), donc sûr au boot.
-- NB : 'facebook'/'spotify' ont pu être ajoutés sur des bases existantes ; on ne
-- les retire pas (PostgreSQL n'autorise pas DROP VALUE), ils restent inertes.
ALTER TYPE auth_provider ADD VALUE IF NOT EXISTS 'github';

-- provider : origine du compte ('local' par défaut → comportement inchangé).
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS provider auth_provider NOT NULL DEFAULT 'local';
-- provider_subject : claim `sub` du provider (identifiant stable côté Google/MS).
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS provider_subject TEXT;
-- Comptes OAuth : aucun mot de passe local → password devient nullable.
ALTER TABLE credentials ALTER COLUMN password DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_credentials_email ON credentials(email);
-- Unicité (provider, subject) quand renseigné : un même compte externe ne peut
-- être lié qu'une fois (les lignes sans subject ne sont pas contraintes).
CREATE UNIQUE INDEX IF NOT EXISTS idx_credentials_provider_subject
    ON credentials(provider, provider_subject)
    WHERE provider_subject IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
-- Lookup à la consommation (token_hash, déjà UNIQUE) + invalidation des tokens
-- précédents d'un même usage pour un utilisateur.
CREATE INDEX IF NOT EXISTS idx_account_tokens_user_purpose ON account_tokens(user_id, purpose);

-- Inscription OAuth en attente : après validation de l'identité Google, aucun
-- compte n'est créé tant que l'utilisateur n'a pas finalisé l'inscription côté
-- front (username/date/CGU). Token opaque haché, usage unique, TTL court.
CREATE TABLE IF NOT EXISTS oauth_signup_tokens (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider         auth_provider NOT NULL,
    provider_subject TEXT NOT NULL,
    email            VARCHAR(255) NOT NULL,
    token_hash       TEXT NOT NULL UNIQUE,
    expires_at       TIMESTAMPTZ NOT NULL,
    used_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_signup_tokens_hash
    ON oauth_signup_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_signup_tokens_provider_subject
    ON oauth_signup_tokens(provider, provider_subject)
    WHERE used_at IS NULL;

-- Trigger updated_at automatique (CREATE OR REPLACE → idempotent, PG ≥ 14).
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_credentials_updated_at
    BEFORE UPDATE ON credentials
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
