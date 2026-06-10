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
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,          -- hash bcrypt
    role        user_role NOT NULL DEFAULT 'user',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    -- deactivated_at : date du bannissement (is_active passé à false). Sert de
    -- point de départ à la purge RGPD automatique des comptes bannis (5 ans).
    -- NULL quand le compte est actif.
    deactivated_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotent pour les volumes créés avant l'ajout de la colonne.
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ;

-- Refresh tokens (préparé pour la feature bonus).
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credentials_email ON credentials(email);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);

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
