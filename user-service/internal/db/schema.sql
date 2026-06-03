-- Schéma canonique du service user — SOURCE DE VÉRITÉ UNIQUE.
--
-- Embarqué dans le binaire (go:embed) et appliqué au démarrage de façon
-- idempotente (EnsureSchema). Le service crée et maintient donc sa propre
-- base : il tourne en standalone (`make run` contre un Postgres nu) comme
-- en stack Docker, sans dépendre d'un script d'init externe.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table des utilisateurs (identité publique, PAS les credentials).
-- id = même UUID que credentials.id (auth-service), porté par le JWT.
CREATE TABLE IF NOT EXISTS users (
    id           UUID PRIMARY KEY,             -- même UUID que credentials.id
    username     VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100),
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Graphe de follows (relations entre utilisateurs).
CREATE TABLE IF NOT EXISTS follows (
    follower_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id),
    CONSTRAINT no_self_follow CHECK (follower_id != following_id)
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows(follower_id);
CREATE INDEX IF NOT EXISTS idx_follows_following ON follows(following_id);

-- Trigger updated_at automatique (CREATE OR REPLACE → idempotent, PG ≥ 14).
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Utilisateur admin par défaut (UUID figé, partagé avec auth-service via
-- SEED_DEFAULT_ADMIN). Idempotent : sans effet aux boots suivants.
INSERT INTO users (id, username, display_name)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'Administrateur Breezy'
) ON CONFLICT (id) DO NOTHING;
