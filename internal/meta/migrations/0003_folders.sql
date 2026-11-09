-- Schema 3: folders, per-folder dedupe, and removal of tags, visibility and the read token scope.
--
-- Token scopes: `read` is removed. A token left without any scope cannot authenticate for anything
-- useful, so the migration revokes it and records that reason in the audit log (actor `system`).

CREATE TABLE folders (
    id         TEXT    PRIMARY KEY,
    owner_id   INTEGER NOT NULL REFERENCES users(id),
    parent_id  TEXT    REFERENCES folders(id),
    name       TEXT    NOT NULL,
    name_key   TEXT    NOT NULL, -- strings.ToLower(name), for sibling uniqueness
    depth      INTEGER NOT NULL CHECK (depth BETWEEN 1 AND 3),
    created_at INTEGER NOT NULL
) STRICT;
CREATE UNIQUE INDEX folders_sibling_name ON folders(owner_id, COALESCE(parent_id, ''), name_key);
CREATE INDEX folders_parent ON folders(owner_id, parent_id);

DROP TABLE image_tags;

-- Images: drop the visibility column, add folder_id, and replace the owner-wide duplicate index
-- with a per-folder one. Every row is copied unchanged.
CREATE TABLE images_new (
    id            TEXT    PRIMARY KEY,
    owner_id      INTEGER NOT NULL REFERENCES users(id),
    token_id      INTEGER REFERENCES tokens(id),
    folder_id     TEXT    REFERENCES folders(id),
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
    state         TEXT    NOT NULL CHECK (state IN ('active', 'trashed', 'purged')),
    thumb         TEXT    NOT NULL DEFAULT '' CHECK (thumb IN ('', 'jpg', 'png', 'none')),
    created_at    INTEGER NOT NULL,
    trashed_at    INTEGER,
    purged_at     INTEGER,
    broken_at     INTEGER
) STRICT;
INSERT INTO images_new (id, owner_id, token_id, folder_id, source, object_key, original_name, mime, ext, size,
                        width, height, frames, sha256, has_gps, state, thumb, created_at, trashed_at, purged_at, broken_at)
SELECT id, owner_id, token_id, NULL, source, object_key, original_name, mime, ext, size,
       width, height, frames, sha256, has_gps, state, thumb, created_at, trashed_at, purged_at, broken_at
FROM images;
DROP TABLE images;
ALTER TABLE images_new RENAME TO images;
CREATE INDEX images_state_created ON images(state, created_at DESC, id DESC);
CREATE INDEX images_state_size    ON images(state, size, id);
CREATE INDEX images_owner_folder_sha ON images(owner_id, folder_id, sha256) WHERE state = 'active';
CREATE INDEX images_token         ON images(token_id, state, created_at DESC);
CREATE INDEX images_trashed_at    ON images(trashed_at) WHERE state = 'trashed';
CREATE INDEX images_thumb_pending ON images(created_at) WHERE thumb = '' AND state = 'active';
CREATE INDEX images_folder        ON images(folder_id, state, created_at DESC, id DESC);

-- Sessions: expires_at, ip and user_agent were written but never read; expiry is derived from the
-- fixed idle/absolute lifetimes. Every row is copied unchanged.
CREATE TABLE sessions_new (
    id_hash      TEXT    PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL
) STRICT;
INSERT INTO sessions_new (id_hash, user_id, created_at, last_seen_at)
SELECT id_hash, user_id, created_at, last_seen_at FROM sessions;
DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;
CREATE INDEX sessions_user ON sessions(user_id);

CREATE INDEX audit_at ON audit(at);

-- Remove the read scope from every token; canonically stored scopes are comma-separated, ascending.
UPDATE tokens SET scopes = trim(replace(replace(replace(scopes, 'read', ''), ',,', ','), ',,', ','), ',');
INSERT INTO audit (at, actor, action, target, detail)
SELECT CAST(strftime('%s', 'now') AS INTEGER) * 1000, 'system', 'token.revoke', CAST(id AS TEXT),
       'no scope left after the read scope was removed by schema migration 0003'
FROM tokens
WHERE scopes = '' AND revoked_at IS NULL;
UPDATE tokens SET revoked_at = CAST(strftime('%s', 'now') AS INTEGER) * 1000
WHERE scopes = '' AND revoked_at IS NULL;

-- Removed settings no longer exist; stale rows would otherwise linger forever.
DELETE FROM settings
WHERE key IN ('maxPixels', 'defaultVisibility', 'uploadsPerMinute', 'sessionIdleMinutes', 'sessionAbsoluteHours');
