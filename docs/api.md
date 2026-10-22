# PicBerth HTTP API v1 — contract

This document is the single source of truth for the HTTP surface. The Go server and the
Svelte SPA both implement it; the SPA uses **only** endpoints listed here (design baseline §7.1).

## 1. Namespaces

| Prefix | Purpose | Auth |
|---|---|---|
| `/i/{id}.{ext}` | Public image bytes (GET/HEAD) | none; private / trashed / unknown → 404 |
| `/api/v1/...` | JSON API | session cookie (SPA) or `Authorization: Bearer <token>` |
| `/assets/...` | Hashed SPA assets | none, `Cache-Control: public, max-age=31536000, immutable` |
| `/healthz`, `/readyz` | Liveness / readiness (`200 ok` / `503`) | none |
| `/`, `/login`, `/setup`, `/library`, `/trash`, `/tokens`, `/settings`, `/status` | SPA entry HTML (exact paths only) | none (SPA checks session itself) |

Anything else → `404`.

## 2. Conventions

- JSON bodies, UTF-8. Field names are camelCase. Times are RFC 3339 UTC strings (`2026-10-21T08:15:30Z`).
- Unknown JSON fields in requests are rejected (`400 validation`); clients must ignore unknown response fields.
- **Session-authenticated unsafe requests** (POST/PATCH/DELETE) must send header `X-PicBerth-Client: web`
  and must pass Go's `http.CrossOriginProtection` (Sec-Fetch-Site / Origin). Token-authenticated requests need neither.
- Every response carries `X-PicBerth-Version: <version>`; the SPA prompts a reload when it changes.
- Tokens are accepted **only** in the `Authorization` header, never in the query string.

### Errors — RFC 9457 `application/problem+json`

```json
{ "type": "urn:picberth:problem:too-large", "title": "File too large", "status": 413,
  "detail": "The file exceeds the 20 MiB limit.", "code": "too-large" }
```

`code` is stable and machine-readable; `title`/`detail` are human text (may change).

| code | status | meaning |
|---|---|---|
| `validation` | 400 | malformed request / bad field |
| `unauthorized` | 401 | no or invalid credentials (also bad login) |
| `forbidden` | 403 | authenticated but not allowed (scope, cross-origin, missing client header) |
| `not-found` | 404 | resource does not exist **or is not visible to the caller** |
| `conflict` | 409 | state does not allow the action (e.g. purge an active image) |
| `too-large` | 413 | upload exceeds size limit |
| `unsupported-format` | 415 | content is not an allowed image format |
| `invalid-image` | 422 | image header/structure invalid or suspicious |
| `too-many-pixels` | 422 | width × height (per frame) exceeds the limit |
| `setup-required` | 409 | no admin exists yet |
| `setup-done` | 409 | setup already completed |
| `rate-limited` | 429 | + `Retry-After` header (seconds) |
| `insufficient-storage` | 507 | free disk space below watermark |
| `internal` | 500 | unexpected error (never includes internal details) |

## 3. Types

### Image

```json
{
  "id": "7k2m9q4xw8bd3f6hz1ca",
  "url": "https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png",
  "filename": "Screenshot 2026-10-21.png",
  "mime": "image/png",
  "ext": "png",
  "size": 183224,
  "width": 1440,
  "height": 900,
  "frames": 1,
  "sha256": "9f2c…(64 hex)",
  "visibility": "public",
  "state": "active",
  "hasGps": false,
  "hasThumb": true,
  "broken": false,
  "duplicate": false,
  "source": { "kind": "web" },
  "tags": ["截图", "工作"],
  "createdAt": "2026-10-21T08:15:30Z",
  "trashedAt": null,
  "links": {
    "url": "https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png",
    "markdown": "![Screenshot 2026-10-21](https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png)",
    "html": "<img src=\"https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png\" alt=\"Screenshot 2026-10-21\" width=\"1440\" height=\"900\">",
    "bbcode": "[img]https://img.example.com/i/7k2m9q4xw8bd3f6hz1ca.png[/img]"
  }
}
```

- `id`: 20 chars of lowercase Crockford base32 (`0-9a-hjkmnp-tv-z`), 100 bits of entropy.
- `visibility`: `public` | `private`. For private images `url`/`links` still contain the would-be public URL,
  but it returns 404 until made public; the SPA shows them via `/api/v1/images/{id}/raw`.
- `state`: `active` | `trashed`. (`purged` images are never returned.)
- `source.kind`: `web` (session upload) | `token` (then `source.tokenId` and `source.tokenName`) | `cli`.
- `tags`: always an array (never `null`), sorted ascending (Go `sort.Strings`, i.e. by UTF-8 bytes). Tag rules:
  trimmed, 1–32 characters, no control characters, no comma, must not start with `#`; exact duplicates are
  dropped; at most 20 per image. Violations are `400 validation`.
- `filename` can be changed with `PATCH` (see below); `id`/`url` never change, `links.markdown`/`html` alt text
  follows the new name.
- `duplicate`: only meaningful in upload responses (true when an existing image was returned).
- `frames` > 1 means animated.
- Markdown/HTML alt text = filename without extension; characters escaped for the target format.

### Token

```json
{ "id": 3, "name": "PicGo on laptop", "prefix": "pb_7k2m9q", "scopes": ["upload"],
  "createdAt": "…", "expiresAt": null, "lastUsedAt": "…", "revokedAt": null }
```

Scopes: `upload` (upload; list/read only images it uploaded), `read` (list/read all images incl. private bytes),
`manage` (read + upload + change visibility, delete, restore, purge). No scope can manage tokens or settings.

### Settings

```json
{
  "maxUploadBytes": 20971520,
  "maxPixels": 50000000,
  "allowedFormats": ["jpeg", "png", "gif", "webp", "avif"],
  "dedupe": true,
  "defaultVisibility": "public",
  "trashRetentionDays": 30,
  "uploadsPerMinute": 60,
  "sessionIdleMinutes": 30,
  "sessionAbsoluteHours": 12,
  "publicUrl": "https://pics.example.com",
  "imageUrl": "https://img.example.com",
  "hardMaxUploadBytes": 104857600
}
```

`publicUrl`, `imageUrl`, `hardMaxUploadBytes` are read-only (infrastructure config).
`maxUploadBytes` ∈ [1 KiB, hardMaxUploadBytes]; `maxPixels` ∈ [1e6, 2e8]; `trashRetentionDays` ∈ [0, 365];
`uploadsPerMinute` ∈ [1, 10000]; `sessionIdleMinutes` ∈ [5, 10080]; `sessionAbsoluteHours` ∈ [1, 8760].

## 4. Session & setup (browser only)

| Method & path | Request | Response |
|---|---|---|
| `GET /api/v1/session` | — | `200 {"authenticated": bool, "setupRequired": bool, "user": {"name": "admin"} \| null, "limits": {"maxUploadBytes", "allowedFormats", "defaultVisibility"} \| null, "version": "0.1.0"}` |
| `POST /api/v1/session` | `{"username", "password"}` | `204` + session cookie; `401 unauthorized`; `429 rate-limited` |
| `DELETE /api/v1/session` | — | `204`, cookie cleared |
| `POST /api/v1/setup` | `{"setupCode", "username", "password"}` | `204` + session cookie; `409 setup-done`; `403 forbidden` (bad code) |

- Password: 10–256 characters. Username: 1–64 chars `[A-Za-z0-9._-]`.
- The setup code is printed to the server log on start while no admin exists; it is single-use.
- Login errors never reveal whether the user exists.

## 5. Images

| Method & path | Auth | Request | Response |
|---|---|---|---|
| `POST /api/v1/images` | session, token `upload`/`manage` | `multipart/form-data`: exactly one file part named `file`; optional text part `visibility`; optional repeated text part `tag` | `201 Image` (new) / `200 Image` with `duplicate:true` (the existing image keeps its own tags; sent `tag` parts are ignored) |
| `GET /api/v1/images` | session, token (`upload` sees own only) | query below | `200 {"items": [Image], "nextCursor": string\|null}` |
| `GET /api/v1/images/facets` | same as list (`state=trashed` needs `manage`) | `?state=active\|trashed` (default `active`) | `200 Facets` |
| `GET /api/v1/images/{id}` | same | — | `200 Image` |
| `PATCH /api/v1/images/{id}` | session, `manage` | any non-empty subset of `{"visibility": "public"\|"private", "filename": string, "tags": [string]}` | `200 Image` |
| `DELETE /api/v1/images/{id}` | session, `manage` | — | `204` (moves to trash; idempotent) |
| `POST /api/v1/images/{id}/restore` | session, `manage` | — | `200 Image`; `409 conflict` if not trashed |
| `POST /api/v1/images/{id}/purge` | session, `manage` | — | `204`; `409 conflict` if not trashed |
| `POST /api/v1/images/batch` | session, `manage` | `{"action": "trash"\|"restore"\|"purge"\|"visibility", "ids": [≤500], "visibility"?}` | `200 {"results": [{"id", "ok": bool, "code"?: string}]}` |
| `GET /api/v1/images/{id}/raw` | session, `read`/`manage` (or `upload` for own) | — | image bytes, `Cache-Control: private, no-store` |
| `GET /api/v1/images/{id}/thumb` | same | — | `image/jpeg` or `image/png` ≤ 480 px; `404` if none (client falls back to raw) |

List query parameters:

| param | values | default |
|---|---|---|
| `state` | `active` \| `trashed` | `active` |
| `q` | substring of filename or of a tag (case-insensitive), or exact id, or sha256 prefix (≥ 8 hex) | — |
| `format` | any of `jpeg` \| `png` \| `gif` \| `webp` \| `avif`; repeatable and/or comma-separated (`format=png&format=jpeg`, `format=png,jpeg`) | — |
| `tag` | repeatable; the image must have **all** given tags (exact match) | — |
| `source` | repeatable, any-of: `web`, `cli`, `token:<id>` (decimal token id) | — |
| `visibility` | `public` \| `private` | — |
| `tokenId` | integer | — |
| `since`, `until` | RFC 3339 timestamps (URL-encode `+`): `createdAt >= since` and `createdAt < until` | — |
| `sort` | `new` \| `old` \| `large` \| `small` \| `name` (filename ascending, ASCII case-insensitive, then id) | `new` |
| `limit` | 1–200 | 60 |
| `cursor` | opaque string from `nextCursor` (bound to `sort` and `state`) | — |

All params combine with AND; within `format` and `source` the values are alternatives. Invalid values are
`400 validation`. Upload-only tokens keep seeing only their own uploads whatever the filters say.

`GET /api/v1/images/facets` counts the caller's whole visible set for the given `state`, ignoring all list filters:

```json
{ "total": 26, "bytes": 23400000,
  "formats": { "jpeg": 7, "png": 16, "gif": 1, "webp": 2, "avif": 0 },
  "visibility": { "public": 21, "private": 5 },
  "sources": [ { "key": "web", "kind": "web", "count": 13 },
               { "key": "token:1", "kind": "token", "tokenId": 1, "tokenName": "PicGo · 笔记本", "count": 5 },
               { "key": "cli", "kind": "cli", "count": 2 } ],
  "tags": [ { "tag": "截图", "count": 7 } ] }
```

- `formats` always has all five keys; `visibility` both keys. `sources` lists only kinds with `count > 0`: `web`
  first, then tokens by count descending (ties by id; a revoked token keeps its name), then `cli`. `tags` are
  sorted by count descending, then tag ascending. `sources` and `tags` are arrays even when empty.
- `PATCH` applies every given field in one transaction. `filename`: trimmed, 1–200 characters, no control
  characters, no `/` or `\`. `tags` replaces the whole tag set. `null` fields count as absent; a body without any
  known field is `400 validation`. Purged images answer `404`, trashed images can still be edited.
- Audit actions written by image changes: `image.visibility` (`old -> new`), `image.rename` (target = id,
  detail = new name), `image.tags` (detail = comma-joined tags, empty when cleared); each only when the value
  actually changed. Existing: `image.trash`, `image.restore`, `image.purge`.
- Purging an image deletes its tags together with the tombstone transition.

Upload notes for third-party clients (PicGo web-uploader, ShareX, scripts):
- field name `file`; URL at JSON path `url` (also `links.markdown` etc.); optional repeated field `tag`
  (same rules as `tags`; `tag` parts do not count towards the 8-part limit, more than 20 is `400 validation`);
- errors carry a human `detail`; ShareX can use `{json:detail}` as ErrorMessage.

Example:

```sh
curl -H "Authorization: Bearer pb_…" -F file=@shot.png https://pics.example.com/api/v1/images
```

## 6. Tokens (session only)

| Method & path | Request | Response |
|---|---|---|
| `GET /api/v1/tokens` | — | `200 {"items": [Token]}` (newest first, revoked included) |
| `POST /api/v1/tokens` | `{"name": 1–80 chars, "scopes": [≥1], "expiresInDays": 1–3650 \| null}` | `201 {"token": Token, "secret": "pb_…"}` — secret shown once |
| `DELETE /api/v1/tokens/{id}` | — | `204` (revoke; idempotent) |

## 7. Settings, status, operations (session only)

| Method & path | Request | Response |
|---|---|---|
| `GET /api/v1/settings` | — | `200 Settings` |
| `PATCH /api/v1/settings` | partial Settings (writable fields only) | `200 Settings` |
| `GET /api/v1/status` | — | `200 Status` |
| `POST /api/v1/checks` | `{"mode": "quick"\|"full"}` | `200 CheckReport` |
| `POST /api/v1/snapshots` | — | `201 {"path", "images", "bytes", "createdAt"}` |
| `GET /api/v1/audit` | `?limit=1–200&cursor=` | `200 {"items": [{"id", "at", "actor", "action", "target", "detail", "ip"}], "nextCursor"}` |

### Status

```json
{
  "version": "0.1.0", "commit": "…", "goVersion": "go1.27.3", "sqliteVersion": "3.53.4",
  "startedAt": "…", "uptimeSeconds": 3600,
  "images": { "active": 1204, "trashed": 12, "bytes": 734003200, "broken": 0 },
  "database": { "bytes": 2490368, "walBytes": 41200 },
  "disk": { "freeBytes": 51539607552, "totalBytes": 107374182400, "watermarkBytes": 2147483648 },
  "dataDir": "/var/lib/picberth",
  "lastSnapshot": { "at": "…", "path": "…" } ,
  "lastCheck": { "at": "…", "mode": "quick", "ok": true, "issues": 0 },
  "warnings": [ { "code": "untrusted-forwarded", "message": "…" } ]
}
```

Warning codes: `public-url-unset`, `insecure-public-url`, `untrusted-forwarded`, `low-disk`,
`snapshot-stale`, `check-failed`, `non-loopback-listen`, `wal-large`.

### CheckReport

```json
{ "mode": "full", "startedAt": "…", "durationMs": 812, "ok": false,
  "checked": 1216,
  "issues": [ { "kind": "missing-file" | "size-mismatch" | "hash-mismatch" | "orphan-file" | "pending-move" | "stale-temp",
                "imageId": "…" | null, "path": "objects/7k/2m/…" | null, "repaired": bool } ] }
```

Quick check: every active/trashed record's file exists with the right size; completes pending moves; removes stale temp files.
Full check: quick + re-hash every file + directory walk for orphans (orphans are moved to `quarantine/`, never deleted).
