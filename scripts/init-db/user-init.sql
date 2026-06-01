-- Init schema : user-service
-- Exécuté automatiquement au premier démarrage du conteneur PostgreSQL

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table des utilisateurs (données publiques, pas les credentials)
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY,              -- même UUID que credentials.id
    username    VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table des follows (graphe de relations)
CREATE TABLE IF NOT EXISTS follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id),
    CONSTRAINT no_self_follow CHECK (follower_id != following_id)
);

-- Index
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows(follower_id);
CREATE INDEX IF NOT EXISTS idx_follows_following ON follows(following_id);

-- Trigger updated_at automatique
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Utilisateur admin par défaut
INSERT INTO users (id, username, display_name)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'Administrateur Breezy'
) ON CONFLICT (id) DO NOTHING;