# PicBerth — implementation architecture

Companion to the design baseline (`PicBerth-设计研究基线-v0.1.md`), the API contract (`docs/api.md`) and the
UI brief (`docs/ui.md`). Section references like "§6.4" point into the baseline.

## 1. Repository layout

```
cmd/picberth/           main: subcommand dispatch (serve, admin, check, snapshot, restore, healthcheck, version)
internal/buildinfo/     Version / Commit set via -ldflags; GoVersion from runtime
internal/config/        infrastructure config: flags > env > file > defaults
internal/ids/           image IDs, secrets, session IDs (crypto/rand, Crockford base32)
internal/password/      Argon2id PHC hashing + verification, bounded concurrency
internal/netx/          trusted-proxy CIDR set + real client IP resolution
internal/ratelimit/     keyed token buckets with idle eviction
internal/datadir/       data-dir layout, instance lock, format marker, free-space query
internal/blob/          file store over os.Root: temp, commit, move between areas, open, walk
internal/imaging/       format sniffing, header/structure inspection, GPS detection, thumbnails
internal/meta/          SQLite: open (writer + reader pools), embedded migrations, repositories
internal/core/          application services, principals, authorization, background jobs
internal/httpapi/       net/http server, middleware, handlers (public, api, spa, health, openapi.json)
internal/webui/         //go:embed all:dist  — Vite build output lands in internal/webui/dist
web/                    Svelte 5 + Vite SPA source (no Go files)
deploy/                 Dockerfile helpers, Caddy / Nginx / systemd / launchd examples
docs/                   this file, api.md, ui.md
```

Dependency direction: `httpapi → core → {meta, blob, imaging, password, ids, ratelimit}`; `cmd → everything`.
`core` never imports `net/http`. Only `cmd/picberth` imports `internal/webui`.

Allowed runtime dependencies (AC-MAINT-01): `modernc.org/sqlite`, `golang.org/x/crypto` (argon2),
`golang.org/x/image` (webp, draw), `golang.org/x/sys` (locks, disk space, Windows service). Nothing else.

## 2. Data directory

```
<data>/
  FORMAT                  "picberth-data 1\n"  (refuse to start on unknown major)
  .lock                   instance lock (separate file — never lock the DB file itself, §12.5)
  picberth.db             SQLite (+ -wal, -shm)
  objects/ab/cd/<id>.<ext>
  trash/ab/cd/<id>.<ext>  same key as in objects/
  tmp/                    upload temp files  up-<random>.part
  cache/thumbs/ab/<id>.<jpg|png>
  quarantine/<unix-ms>-<name>   orphans found by full check (never auto-deleted)
  snapshots/<YYYYMMDDTHHMMSSZ>/  default snapshot destination
  backups/picberth-schema<N>-<UTC>.db   automatic copy taken before migrating an existing schema N (newest 5 kept)
```

- Object key = `<id[0:2]>/<id[2:4]>/<id>.<ext>`; stored in the DB with `/` separators; the area
  (`objects` / `trash`) is derived from the record state. Keys only contain `[0-9a-z./]`.
- All file operations go through one `*os.Root` opened on `<data>` (§9.1). Never build paths from user input.
- Unix: dirs 0700, files 0600. Windows: inherit ACLs.

## 3. Database (SQLite, STRICT tables, `PRAGMA user_version` migrations)

```
users(id INTEGER PK, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, created_at INTEGER NOT NULL)
sessions(id_hash TEXT PK, user_id INTEGER NOT NULL REFERENCES users ON DELETE CASCADE,
         created_at INTEGER, last_seen_at INTEGER) -- expiry from fixed 7 d idle / 30 d absolute lifetimes
tokens(id INTEGER PK, owner_id INTEGER NOT NULL REFERENCES users, name TEXT NOT NULL, prefix TEXT NOT NULL,
       secret_hash TEXT UNIQUE NOT NULL, scopes TEXT NOT NULL, created_at INTEGER, expires_at INTEGER NULL,
       last_used_at INTEGER NULL, revoked_at INTEGER NULL)
images(id TEXT PK, owner_id INTEGER NOT NULL REFERENCES users, token_id INTEGER NULL REFERENCES tokens,
       folder_id TEXT NULL REFERENCES folders,
       source TEXT NOT NULL CHECK(source IN ('web','token','cli')),
       object_key TEXT UNIQUE NOT NULL, original_name TEXT NOT NULL, mime TEXT NOT NULL, ext TEXT NOT NULL,
       size INTEGER NOT NULL, width INTEGER NOT NULL, height INTEGER NOT NULL, frames INTEGER NOT NULL,
       sha256 TEXT NOT NULL, has_gps INTEGER NOT NULL,
       state TEXT NOT NULL CHECK(state IN ('active','trashed','purged')),
       thumb TEXT NOT NULL DEFAULT '' CHECK(thumb IN ('','jpg','png','none')),
       created_at INTEGER NOT NULL, trashed_at INTEGER NULL, purged_at INTEGER NULL, broken_at INTEGER NULL)
  indexes: (state, created_at DESC, id DESC), (state, size, id), (folder_id, state, created_at DESC, id DESC),
           (owner_id, folder_id, sha256) WHERE state='active', (token_id, state, created_at DESC),
           (trashed_at) WHERE state='trashed', (created_at) WHERE thumb='' AND state='active'
folders(id TEXT PK, owner_id INTEGER NOT NULL REFERENCES users, parent_id TEXT NULL REFERENCES folders,
        name TEXT NOT NULL, name_key TEXT NOT NULL, depth INTEGER NOT NULL CHECK(depth BETWEEN 1 AND 3),
        created_at INTEGER NOT NULL)
  unique: (owner_id, COALESCE(parent_id,''), name_key); index (owner_id, parent_id)
settings(key TEXT PK, value TEXT NOT NULL)          -- JSON values; defaults live in code
audit(id INTEGER PK, at INTEGER NOT NULL, actor TEXT NOT NULL, action TEXT NOT NULL, target TEXT, detail TEXT, ip TEXT)
  index: (at)                                        -- schema 3, for retention pruning
kv(key TEXT PK, value TEXT NOT NULL)                -- last_snapshot, last_check
```

Times are unix milliseconds UTC. `application_id` = `0x50494342` ("PICB").

Schema history: 1 initial; 2 tags; 3 folders and removal of tags, visibility and the `read` token scope
(also drops the write-only session columns `expires_at`/`ip`/`user_agent`, rebuilds `images` with
`folder_id`, adds the audit `at` index, prunes removed settings, strips `read` from tokens and revokes
tokens left without a scope — with an audit row naming the migration reason, and rotates `backups/` to the
newest five files).

Connections (ADR-004): writer pool `SetMaxOpenConns(1)`, reader pool `SetMaxOpenConns(max(4, GOMAXPROCS))`,
readers open with `query_only=1`. Every connection gets — through the DSN `_pragma` parameters, never a one-off
Exec — `journal_mode=WAL`, `synchronous=FULL`, `foreign_keys=ON`, `busy_timeout=5000`. Write transactions start
with `BEGIN IMMEDIATE` (driver `_txlock=immediate`). Assert at open that `sqlite_version() >= 3.51.3` (AC-DATA-08).
Every folder write runs its checks — ownership, sibling-name uniqueness, cycles, subtree depth, emptiness
and the release of trashed images — inside the same write transaction as the change; an image INSERT or
folder UPDATE re-verifies its destination folder there too, so concurrent writes cannot create a cycle,
exceed three levels, or leave an active image in a folder that no longer exists.

Purged images keep a tombstone row (id, sha256, times, state='purged'); the other columns stay as they were
except `folder_id`, which purge clears, so a purged image never holds a deleted folder reference. The ID can
never be reused (§6.1).

## 4. Consistency protocol (§6.4–6.6, ADR-005)

The database is the source of truth for *where a file must be*: `active` → `objects/`, `trashed` → `trash/`,
`purged` → nowhere.

| Operation | Order |
|---|---|
| Upload | stream to `tmp/` (hash + count) → inspect → resolve the destination folder → dedupe (visible active, healthy records: same owner **and same folder**, the root counting as a folder; or own uploads for an upload-only token) → `fsync` file → close → rename to `objects/key` → fsync parent dir (Unix) → INSERT record (re-checking the destination folder and the same dedupe scope in one write transaction) → audit `image.upload` → enqueue thumbnail |
| Trash | UPDATE state=trashed → rename `objects/key` → `trash/key` (retry on Windows sharing violations) |
| Restore | rename `trash/key` → `objects/key` → UPDATE state=active |
| Purge | UPDATE state=purged → remove `trash/key` and thumbnail |

Check (quick): for each active/trashed record, stat the expected path; if missing but present in the other area,
move it (`pending-move`, repaired); if missing in both → set `broken_at`; size mismatch → `broken_at`;
remove files left in `trash/` for purged records; delete `tmp/` files older than the maximum upload duration.
Check (full): quick + re-hash each file (`hash-mismatch` → broken) + walk `objects/` and `trash/` for files
with no active/trashed record → move to `quarantine/` (`orphan-file`). Only a successful full check of an image
clears its `broken_at` flag: restoring a file or passing a quick check alone does not prove its bytes are intact.

Background jobs (in-process tickers): quick check 30 s after start and every 24 h; purge expired trash hourly;
delete expired sessions hourly; prune the audit log hourly in batches of 1000 by `auditRetentionDays`
(2 min after start, no audit row, log only); thumbnail worker (concurrency 1) fed by upload events and a
startup backfill.

## 5. Snapshot / restore (§12.2–12.3)

`Snapshot(dest)`: take the purge lock (pauses purging, not uploads) → `VACUUM INTO dest/picberth.db` → open the
snapshot read-only and list active + trashed records → for each, hard-link (fallback: copy) the file into
`dest/objects|trash/key`, hashing and comparing with the recorded sha256 (any mismatch fails the snapshot and
removes `dest`) → write `dest/MANIFEST.json` (`format`, `version`, `schemaVersion`, `createdAt`, `images`, `bytes`)
→ release lock → record in `kv`. Thumbnails are not included (derived data).

`restore --from SNAP --data DIR`: DIR must be empty or absent; copy snapshot contents, write FORMAT, run a full
check; non-zero exit on any issue.

## 6. Security mechanics

- Principal = `{Kind: anonymous|session|token|cli, UserID, TokenID, Scopes, Name}`. `core` checks every
  operation against the matrix in api.md §3 (Token scopes) and baseline §7.3. Tokens can never manage tokens or
  settings; `upload`-scoped tokens only see images they uploaded. Scopes are only `upload` and `manage`;
  folder reads are allowed for every authenticated principal of the owner, folder writes need a session or
  `manage`.
- Sessions: 32 random bytes, base64url in the cookie, SHA-256 hex in the DB. Cookie `__Host-pb_session`
  (`Secure`) when the public URL is https, else `pb_session`; `HttpOnly; SameSite=Strict; Path=/`. Idle + absolute
  expiry are fixed (7 days / 30 days); `last_seen_at` written at most once per minute.
- Tokens: `pb_` + 32 Crockford base32 chars (160 bits). DB stores SHA-256 hex. Display prefix = first 9 chars.
  `last_used_at` written at most once per minute per token.
- Passwords: Argon2id m=19456 KiB, t=2, p=1, 16-byte salt, 32-byte key, PHC string. At most 2 concurrent hashes.
  Unknown users still run one dummy verification (timing).
- Setup: if no user exists at `serve` start, generate a code `XXXX-XXXX-XXXX` (Crockford), log it at WARN with
  instructions; single use, constant-time compare; the admin INSERT is conditional on `NOT EXISTS (SELECT 1 FROM users)`.
- CSRF: API mux wrapped in `http.CrossOriginProtection`; session-authenticated unsafe requests also require
  `X-PicBerth-Client: web`.
- Real IP: peer address unless the peer is in `trusted-proxies`; then walk `X-Forwarded-For` right→left skipping
  trusted entries. Requests from an untrusted peer carrying `X-Forwarded-For` set the `untrusted-forwarded` warning.
- Rate limits: login 5/min burst 5 per IP and 10 per 10 min per username; uploads 60/min per principal
  (token id, or user for sessions), a fixed constant; 429 + `Retry-After`.
- Uploads: `MaxBytesReader(maxUploadBytes + 64 KiB)`; `r.MultipartReader()` streaming (never ParseMultipartForm);
  at most 8 parts; exactly one part named `file`; optional `folder` part with a folder id (empty = root,
  unknown/foreign = 404, never created implicitly); max 50 MP per frame (constant); process-wide semaphore of
  4 concurrent uploads (excess → 429 with Retry-After 2); disk watermark `max(1 GiB, 2% of volume)` checked
  before writing.
- Response headers:
  - SPA HTML: `Content-Security-Policy: default-src 'self'; img-src 'self' blob: data: <image origin>; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'`,
    `Referrer-Policy: same-origin`, `X-Frame-Options: DENY`, `Cross-Origin-Opener-Policy: same-origin`, `Cache-Control: no-cache`.
  - API: `Cache-Control: no-store`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`.
  - Public image: `Content-Type` from DB, `Cache-Control: public, max-age=31536000, immutable`, strong `ETag` =
    `"<sha256>"`, `Last-Modified`, `Content-Security-Policy: default-src 'none'; sandbox`,
    `Cross-Origin-Resource-Policy: cross-origin`, `Content-Disposition: inline; filename="<id>.<ext>"` (the
    original filename is never exposed publicly). `If-None-Match` hit → 304 without opening the file.
  - Raw/thumb (authenticated): `Cache-Control: private, no-store`, `Cross-Origin-Resource-Policy: same-origin`.
  - All responses: `X-Content-Type-Options: nosniff`, `X-PicBerth-Version`.
- Server: `ReadHeaderTimeout 10s`, `IdleTimeout 120s`, `MaxHeaderBytes 64 KiB`; per-request deadlines via
  `http.ResponseController`: 30 s read for normal requests, `max(2 min, maxUpload / 64 KiB/s)` for uploads.
- Graceful shutdown: readiness → 503, `Shutdown` with 8 s deadline, stop jobs, close DB, release lock.

## 7. Configuration (ADR-011)

| Env / file key | Flag | Default |
|---|---|---|
| `PICBERTH_LISTEN` | `--listen` | `127.0.0.1:12433` |
| `PICBERTH_DATA_DIR` | `--data-dir` | Linux `$XDG_DATA_HOME/picberth` (`~/.local/share/picberth`), macOS `~/Library/Application Support/PicBerth`, Windows `%LOCALAPPDATA%\PicBerth` |
| `PICBERTH_PUBLIC_URL` | `--public-url` | `http://<listen>` (+ warning `public-url-unset`) |
| `PICBERTH_IMAGE_URL` | `--image-url` | = public URL |
| `PICBERTH_TRUSTED_PROXIES` | `--trusted-proxies` | empty (comma-separated IPs/CIDRs) |
| `PICBERTH_HARD_MAX_UPLOAD` | `--hard-max-upload` | `104857600` |
| `PICBERTH_SNAPSHOT_DIR` | `--snapshot-dir` | `<data>/snapshots` |
| `PICBERTH_LOG_LEVEL` / `PICBERTH_LOG_FORMAT` | `--log-level` / `--log-format` | `info` / `text` |
| `PICBERTH_DEV_WEB_DIR` | `--dev-web-dir` | empty (serve embedded SPA) |
| `PICBERTH_CONFIG` | `--config` | none — optional file of `KEY=value` lines using the env names, `#` comments |

The Docker image sets `PICBERTH_LISTEN=0.0.0.0:12433` and `PICBERTH_DATA_DIR=/data`; the program has no
"am I in a container" logic (§10.1).

## 8. Build & dev

- `mise run build` → `pnpm --dir web build` (outputs to `internal/webui/dist`) → `go build -trimpath
  -ldflags "-s -w -X …buildinfo.Version=…" ./cmd/picberth` with `CGO_ENABLED=0`.
- `mise run dev` builds the embedded SPA, then starts the Go server on 127.0.0.1:12433. Run `mise run web:dev`
  separately for Vite on 127.0.0.1:5173, proxying `/api` and `/i`.
- `mise run test` builds the embedded SPA, then runs `go vet ./...`, `go test ./...` and `pnpm --dir web check`.
  Run `mise run web:install` first. Direct Go commands that include `cmd/picberth` or `internal/webui` also need
  `mise run web:build` first on a clean checkout.
