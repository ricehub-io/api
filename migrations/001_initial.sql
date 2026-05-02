CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username CITEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password TEXT NOT NULL,
    avatar_path TEXT,
    is_admin BOOL NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(author_id, slug)
);

CREATE TABLE rice_dotfiles (
    rice_id UUID PRIMARY KEY REFERENCES rices(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL UNIQUE,
    download_count INTEGER NOT NULL DEFAULT 0 CHECK (download_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rice_previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rice_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rice_stars (
    rice_id UUID NOT NULL REFERENCES rices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    starred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(rice_id, user_id)
);

CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    rice_id UUID REFERENCES rices(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES rice_comments(id) ON DELETE CASCADE,
    is_closed BOOL NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- make sure that at least one object is referenced
    CHECK (
        (rice_id IS NOT NULL)::int + (comment_id IS NOT NULL)::int = 1
    ),
    -- create unique key to ensure users dont send duplicated reports
    UNIQUE(reporter_id, reason, is_closed)
);

-- logic behind updating the `updated_at` column for all tables
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    -- check if we're updating download_count
    IF to_jsonb(NEW) ? 'download_count' THEN
        if NEW.download_count > OLD.download_count THEN
            RETURN NEW;
        END IF;
    END IF;

    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_tags_updated_at
    BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rices_updated_at
    BEFORE UPDATE ON rices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rice_dotfiles_updated_at
    BEFORE UPDATE ON rice_dotfiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_rice_comments_updated_at
    BEFORE UPDATE ON rice_comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

INSERT INTO tags (name)
VALUES ('AwesomeWM'), ('Arch Linux'), ('KDE'), ('Hyprland'), ('i3'), ('bspwm');

ALTER TABLE rice_dotfiles
    ADD COLUMN file_size BIGINT NOT NULL CHECK (file_size > 0);

CREATE TABLE website_variables (
    key TEXT PRIMARY KEY CHECK (key ~ '^[a-z0-9_]+$'),
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER update_website_variables_updated_at
    BEFORE UPDATE ON website_variables
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE links (
    name TEXT PRIMARY KEY CHECK (name ~ '^[a-z]+$'),
    url TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER update_links_updated_at
    BEFORE UPDATE ON links
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

INSERT INTO website_variables (key, value)
VALUES
    ('terms_of_service_text', 'Lorem ipsum'),
    ('privacy_policy_text', 'Lorem ipsum');

INSERT INTO links (name, url)
VALUES
    ('discord', 'https://discord.com'),
    ('github', 'https://github.com');

CREATE TABLE user_bans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    admin_id UUID REFERENCES users(id) CHECK (admin_id != user_id),
    reason TEXT NOT NULL,
    is_revoked BOOL NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,
    banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE OR REPLACE FUNCTION update_revoked_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_revoked IS true THEN
        NEW.revoked_at = NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql'; 

-- update `revoked_at` column in `user_bans` if ban is revoked
CREATE TRIGGER update_user_ban_revoked_at
    BEFORE UPDATE OF is_revoked ON user_bans
    FOR EACH ROW EXECUTE FUNCTION update_revoked_at();

-- create a view for fetching user data + ban info
CREATE VIEW users_with_ban_status AS
SELECT
    u.*,
    EXISTS (
        SELECT 1
        FROM user_bans b
        WHERE
            b.user_id = u.id
            AND (b.expires_at > NOW() OR b.expires_at IS NULL)
            AND b.is_revoked = false
    ) AS is_banned
FROM users u;

-- add state column to rices for manual verification before being publicly visible 
CREATE TYPE rice_state AS ENUM (
    'waiting',
    'accepted'
);

ALTER TABLE rices
    ADD COLUMN "state" rice_state NOT NULL DEFAULT 'waiting';

-- create dotfiles type enum and add column to the table
CREATE TYPE dotfiles_type AS ENUM (
    'free',
    'one-time'
);

ALTER TABLE rice_dotfiles
    ADD COLUMN "type" dotfiles_type NOT NULL DEFAULT 'free';

-- add price column to dotfiles
ALTER TABLE rice_dotfiles
    ADD COLUMN price NUMERIC(5, 2) NOT NULL DEFAULT 1.0 CHECK (price > 0.0);

-- create table to keep track of dotfiles purchased by users
CREATE TABLE dotfiles_purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    rice_id UUID NOT NULL REFERENCES rices(id),
    price_paid NUMERIC(5, 2) NOT NULL CHECK (price_paid > 0.0),
    purchased_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- add polar product id to rice dotfiles
ALTER TABLE rice_dotfiles
    ADD COLUMN product_id UUID CHECK (product_id IS NOT NULL OR "type" = 'free');

-- create table to keep track of users' subscription
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) UNIQUE,
    current_period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now() 
);

CREATE TRIGGER update_user_subscriptions_updated_at
    BEFORE UPDATE ON user_subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- table to keep record of ALL webhooks that were received by API
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    polar_webhook_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    error TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- M-to-M table for rices and tags
CREATE TABLE rice_tag (
    rice_id UUID REFERENCES rices(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (rice_id, tag_id)
);

-- rice download events
CREATE TABLE rice_downloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rice_id UUID NOT NULL REFERENCES rices(id),
    user_id UUID REFERENCES users(id), -- null if triggered by not signed in user
    downloaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- leaderboard stuff
CREATE TYPE leaderboard_period AS ENUM (
    'week',
    'month',
    'year'
);

CREATE TABLE rice_leaderboard (
    rice_id UUID REFERENCES rices(id) ON DELETE CASCADE,
    period leaderboard_period NOT NULL,
    position INT NOT NULL CHECK (position > 0),
    score BIGINT NOT NULL CHECK (score >= 0),
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (rice_id, period)
);

-- user subscription status
CREATE TYPE subscription_status AS ENUM (
    'active',
    'canceled'
);

ALTER TABLE user_subscriptions ADD COLUMN status subscription_status NOT NULL;

-- previews => screenshots refactor
ALTER TABLE rice_previews RENAME TO rice_screenshots;