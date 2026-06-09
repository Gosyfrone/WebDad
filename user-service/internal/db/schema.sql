-- Schéma canonique du service user — SOURCE DE VÉRITÉ UNIQUE.
--
-- Embarqué dans le binaire (go:embed) et appliqué au démarrage de façon
-- idempotente (EnsureSchema). Le service crée et maintient donc sa propre
-- base : il tourne en standalone (`make run` contre un Postgres nu) comme
-- en stack Docker, sans dépendre d'un script d'init externe.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table des utilisateurs (identité publique, PAS les credentials).
-- id = même UUID que credentials.id (auth-service), porté par le JWT.
-- display_name et les autres champs décoratifs vivent dans profil-service
-- (Mongo) : user-service ne porte que l'identité immuable + l'état du compte.
CREATE TABLE IF NOT EXISTS users (
    id                  UUID PRIMARY KEY,        -- même UUID que credentials.id
    username            VARCHAR(50) NOT NULL UNIQUE,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    -- Date du dernier changement EFFECTIF de username (NULL = jamais changé
    -- depuis le provisioning). Base d'un cooldown « X jours entre deux
    -- changements de handle » (cf. config UsernameCooldown ; enforcement
    -- désactivé par défaut). Capturée dès maintenant pour avoir une baseline.
    -- Symétrique de profiles.display_name_changed_at (profil-service).
    username_changed_at TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ajout idempotent de la colonne pour les bases déjà créées (avant cette
-- migration) : CREATE TABLE IF NOT EXISTS ne modifie pas une table existante.
ALTER TABLE users ADD COLUMN IF NOT EXISTS username_changed_at TIMESTAMPTZ;

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

-- Demandes de follow vers profils privés. La relation réelle n'est créée dans
-- `follows` qu'après acceptation par le propriétaire du profil privé.
CREATE TABLE IF NOT EXISTS follow_requests (
    follower_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id),
    CONSTRAINT no_self_follow_request CHECK (follower_id != following_id)
);

CREATE INDEX IF NOT EXISTS idx_follow_requests_follower ON follow_requests(follower_id);
CREATE INDEX IF NOT EXISTS idx_follow_requests_following ON follow_requests(following_id);

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
INSERT INTO users (id, username)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin'
) ON CONFLICT (id) DO NOTHING;
