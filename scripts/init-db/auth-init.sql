-- Init schema : auth-service
-- Exécuté automatiquement au premier démarrage du conteneur PostgreSQL

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enum pour les rôles
CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin');

-- Table principale des credentials
CREATE TABLE IF NOT EXISTS credentials (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,          -- bcrypt hash
    role        user_role NOT NULL DEFAULT 'user',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table des refresh tokens (Fx bonus)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index
CREATE INDEX IF NOT EXISTS idx_credentials_email ON credentials(email);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);

-- Trigger updated_at automatique
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_credentials_updated_at
    BEFORE UPDATE ON credentials
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Admin par défaut.
-- ⚠️ UUID hardcodé pour rester aligné avec user-init.sql et profil-init.js.
-- ⚠️ Le hash ci-dessous est un PLACEHOLDER bcrypt — à REMPLACER avant tout déploiement.
--    Génération : `python3 -c "import bcrypt; print(bcrypt.hashpw(b'MOT_DE_PASSE', bcrypt.gensalt(12)).decode())"`
INSERT INTO credentials (id, email, password, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@webdad.local',
    '$2a$12$REPLACE_BEFORE_DEPLOY___this_is_a_dev_only_placeholder_xxxx',
    'admin'
) ON CONFLICT (id) DO NOTHING;
