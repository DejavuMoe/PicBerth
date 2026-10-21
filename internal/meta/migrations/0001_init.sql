CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT    NOT NULL,
    created_at    INTEGER NOT NULL
) STRICT;

CREATE TABLE sessions (
    id_hash      TEXT    PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    ip           TEXT    NOT NULL DEFAULT '',
    user_agent   TEXT    NOT NULL DEFAULT ''
) STRICT;
CREATE INDEX sessions_user ON sessions(user_id);

CREATE TABLE tokens (
    id           INTEGER PRIMARY KEY,
    owner_id     INTEGER NOT NULL REFERENCES users(id),
    name         TEXT    NOT NULL,
    prefix       TEXT    NOT NULL,
    secret_hash  TEXT    NOT NULL UNIQUE,
    scopes       TEXT    NOT NULL,
    created_at   INTEGER NOT NULL,
    expires_at   INTEGER,
    last_used_at INTEGER,
    revoked_at   INTEGER
) STRICT;

CREATE TABLE images (
    id            TEXT    PRIMARY KEY,
    owner_id      INTEGER NOT NULL REFERENCES users(id),
    token_id      INTEGER REFERENCES tokens(id),
    source        TEXT    NOT NULL CHECK (source IN ('web', 'token', 'cli')),
    object_key    TEXT    NOT NULL UNIQUE,
    original_name TEXT    NOT NULL,
    mime          TEXT    NOT NULL,
    ext           TEXT    NOT NULL,
    size          INTEGER NOT NULL,
    width         INTEGER NOT NULL,
    height        INTEGER NOT NULL,
    frames        INTEGER NOT NULL,
    sha256        TEXT    NOT NULL,
    has_gps       INTEGER NOT NULL CHECK (has_gps IN (0, 1)),
    visibility    TEXT    NOT NULL CHECK (visibility IN ('public', 'private')),
    state         TEXT    NOT NULL CHECK (state IN ('active', 'trashed', 'purged')),
    thumb         TEXT    NOT NULL DEFAULT '' CHECK (thumb IN ('', 'jpg', 'png', 'none')),
    created_at    INTEGER NOT NULL,
    trashed_at    INTEGER,
    purged_at     INTEGER,
    broken_at     INTEGER
) STRICT;
CREATE INDEX images_state_created ON images(state, created_at DESC, id DESC);
CREATE INDEX images_state_size    ON images(state, size, id);
CREATE INDEX images_owner_sha     ON images(owner_id, sha256) WHERE state = 'active';
CREATE INDEX images_token         ON images(token_id, state, created_at DESC);
CREATE INDEX images_trashed_at    ON images(trashed_at) WHERE state = 'trashed';
CREATE INDEX images_thumb_pending ON images(created_at) WHERE thumb = '' AND state = 'active';

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;

CREATE TABLE audit (
    id     INTEGER PRIMARY KEY,
    at     INTEGER NOT NULL,
    actor  TEXT    NOT NULL,
    action TEXT    NOT NULL,
    target TEXT,
    detail TEXT,
    ip     TEXT
) STRICT;

CREATE TABLE kv (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;
