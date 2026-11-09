# PicBerth HTTP API v1 — contract

This document is the single source of truth for the HTTP surface. The Go server and the
Svelte SPA both implement it; the SPA uses **only** endpoints listed here (design baseline §7.1).
The machine-readable form is `GET /api/v1/openapi.json` (OpenAPI 3.1, embedded in the binary, no
authentication required); the human reference page is `/docs/api`. Both are kept in sync with this file.

## 1. Namespaces

| Prefix | Purpose | Auth |
|---|---|---|
| `/i/{id}.{ext}` | Public image bytes (GET/HEAD) | none; every active, healthy image is public; trashed/unknown → 404 |
| `/api/v1/...` | JSON API | session cookie (SPA) or `Authorization: Bearer <token>` |
| `/assets/...` | Hashed SPA assets | none, `Cache-Control: public, max-age=31536000, immutable` |
| `/healthz`, `/readyz` | Liveness / readiness (`200 ok` / `503`) | none |
| `/`, `/login`, `/setup`, `/library`, `/trash`, `/tokens`, `/settings`, `/status`, `/docs/api` | SPA entry HTML | none (SPA checks session itself) |
| `/library/{folder path}` | SPA entry HTML for any path below `/library` | none |

Anything else → `404`. Folder paths such as `/library/旅行/2026/云南` are client-side routes: the server
always answers with the SPA entry, and a missing folder is the SPA's own 文件夹不存在 state.

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
| `folder-exists` | 409 | a sibling folder already uses the name (case-insensitively) |
| `folder-not-empty` | 409 | the folder still has subfolders or active images |
| `folder-depth` | 422 | the operation would create a fourth folder level |
| `invalid-scope` | 422 | an unknown or removed token scope (for example `read`) |
| `too-large` | 413 | upload exceeds size limit |
| `unsupported-format` | 415 | content is not an allowed image format |
| `invalid-image` | 422 | image header/structure invalid or suspicious |
| `too-many-pixels` | 422 | width × height (per frame) exceeds 50 MP |
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
  "folderId": "3n7vq2k9xwbd4f6h",
  "mime": "image/png",
  "ext": "png",
  "size": 183224,
  "width": 1440,
  "height": 900,
  "frames": 1,
  "sha256": "9f2c…(64 hex)",
  "state": "active",
  "hasGps": false,
  "hasThumb": true,
  "broken": false,
  "duplicate": false,
  "source": { "kind": "web" },
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
- `folderId`: the folder containing the image, or `null` for the root (图片). Folder ids are opaque:
  16 chars of the same alphabet. Moving images or folders never changes `url`.
- Every uploaded image is public; there is no visibility field. Trashed images return 404 on `/i/...`.
- `state`: `active` | `trashed`. (`purged` images are never returned.)
- `source.kind`: `web` (session upload) | `token` (then `source.tokenId` and `source.tokenName`) | `cli`.
- `filename` can be changed with `PATCH` (see below); `id`/`url` never change, `links.markdown`/`html` alt text
  follows the new name.
- `duplicate`: only meaningful in upload responses (true when an existing image was returned). Deduplication
  is scoped to the destination folder (the root counts as a folder): same owner and same folder. An
  upload-only token deduplicates only against its own uploads in that folder.
- `frames` > 1 means animated.
- Markdown/HTML alt text = filename without extension; characters escaped for the target format.

### Folder

```json
{ "id": "3n7vq2k9xwbd4f6h", "name": "云南", "parentId": "k2m9q4xw8bd3f6hz", "depth": 3,
  "imageCount": 4, "createdAt": "2026-11-09T08:00:00Z" }
```

- Flat list: `GET /folders` returns every folder with `parentId` linking the tree.
- `depth` is 1–3. Names are 1–64 characters (trimmed, no `/` or `\`, no control characters) and unique among
  siblings case-insensitively.
- `imageCount` counts **active images directly in the folder** (subfolders and trashed images excluded).
- Folders are organisation only. An image is in exactly one folder or in the root.

### Token

```json
{ "id": 3, "name": "PicGo on laptop", "prefix": "pb_7k2m9q", "scopes": ["upload"],
  "createdAt": "…", "expiresAt": null, "lastUsedAt": "…", "revokedAt": null }
```

Scopes: `upload` (upload; list/read only images it uploaded) and `manage` (read, upload, rename, move,
delete, restore, purge images; create, rename, move and delete folders). No scope can manage tokens or
settings. The `read` scope was removed in schema 3; `POST /tokens` answers `422 invalid-scope` for it.

### Settings

```json
{
  "maxUploadBytes": 20971520,
  "allowedFormats": ["jpeg", "png", "gif", "webp", "avif"],
  "dedupe": true,
  "trashRetentionDays": 30,
  "auditRetentionDays": 180,
  "publicUrl": "https://pics.example.com",
  "imageUrl": "https://img.example.com",
  "hardMaxUploadBytes": 104857600
}
```

`publicUrl`, `imageUrl`, `hardMaxUploadBytes` are read-only (infrastructure config); `PATCH` accepts them
only when they equal the current value, so a client may echo a whole `GET` response back.
`maxUploadBytes` ∈ [1 KiB, hardMaxUploadBytes]; `allowedFormats` is a non-empty subset of the five formats;
`trashRetentionDays` ∈ [0, 365]; `auditRetentionDays` ∈ [7, 3650].

Fixed constants (not settings): uploads 60/min per principal, session idle 7 days, session absolute 30 days,
max 50 MP per frame.

## 4. Session & setup (browser only)

| Method & path | Request | Response |
|---|---|---|
| `GET /api/v1/session` | — | `200 {"authenticated": bool, "setupRequired": bool, "user": {"name": "admin"} \| null, "limits": {"maxUploadBytes", "allowedFormats"} \| null, "version": "v0.0.1"}` |
| `POST /api/v1/session` | `{"username", "password"}` | `204` + session cookie; `401 unauthorized`; `429 rate-limited` |
| `DELETE /api/v1/session` | — | `204`, cookie cleared |
| `POST /api/v1/setup` | `{"setupCode", "username", "password"}` | `204` + session cookie; `409 setup-done`; `403 forbidden` (bad code) |

- Password: 10–256 characters. Username: 1–64 chars `[A-Za-z0-9._-]`.
- The setup code is printed to the server log on start while no admin exists; it is single-use.
- Login errors never reveal whether the user exists.
- Sessions expire after 7 days idle or 30 days after creation, whichever comes first.

## 5. Images

| Method & path | Auth | Request | Response |
|---|---|---|---|
| `POST /api/v1/images` | session, token `upload`/`manage` | `multipart/form-data`: exactly one file part named `file`; optional text part `folder` (folder id; empty = root) | `201 Image` (new) / `200 Image` with `duplicate:true` (an identical image already exists in the folder) |
| `GET /api/v1/images` | session, token (`upload` sees own only) | query below | `200 {"items": [Image], "nextCursor": string\|null}` |
| `GET /api/v1/images/facets` | same as list (`state=trashed` needs `manage`) | `?state=active\|trashed` (default `active`) | `200 Facets` |
| `GET /api/v1/images/{id}` | same | — | `200 Image` |
| `PATCH /api/v1/images/{id}` | session, `manage` | any non-empty subset of `{"filename": string, "folderId": string\|null}` | `200 Image` |
| `DELETE /api/v1/images/{id}` | session, `manage` | — | `204` (moves to trash; idempotent) |
| `POST /api/v1/images/{id}/restore` | session, `manage` | — | `200 Image`; `409 conflict` if not trashed |
| `POST /api/v1/images/{id}/purge` | session, `manage` | — | `204`; `409 conflict` if not trashed |
| `POST /api/v1/images/batch` | session, `manage` | `{"action": "trash"\|"restore"\|"purge"\|"move", "ids": [≤500], "folderId"?: string\|null}` | `200 {"results": [{"id", "ok": bool, "code"?: string}]}` |
| `GET /api/v1/images/{id}/raw` | session, token (or `upload` for own) | — | image bytes, `Cache-Control: private, no-store` |
| `GET /api/v1/images/{id}/thumb` | same | — | `image/jpeg` or `image/png` ≤ 480 px; `404` if none (client falls back to raw) |

List query parameters:

| param | values | default |
|---|---|---|
| `state` | `active` \| `trashed` | `active` |
| `folder` | `root`, or a folder id: images directly in that folder | — (every folder) |
| `recursive` | `true` \| `false`: with `folder`, include all descendants (at the root: the whole library) | `false` |
| `q` | substring of filename (case-insensitive), exact id, or sha256 prefix (≥ 8 hex). Never matches folders. | — |
| `format` | any of `jpeg` \| `png` \| `gif` \| `webp` \| `avif`; repeatable and/or comma-separated | — |
| `source` | repeatable, any-of: `web`, `cli`, `token:<id>` (decimal token id) | — |
| `tokenId` | integer | — |
| `since`, `until` | RFC 3339 timestamps (URL-encode `+`): `createdAt >= since` and `createdAt < until` | — |
| `sort` | `new` \| `old` \| `large` \| `small` \| `name` (filename ascending, ASCII case-insensitive, then id) | `new` |
| `limit` | 1–200 | 60 |
| `cursor` | opaque string from `nextCursor` (bound to `sort`, `state`, `folder` and `recursive`) | — |

All params combine with AND; within `format` and `source` the values are alternatives. Invalid values are
`400 validation`; an unknown or foreign `folder` is `404 not-found`. Upload-only tokens keep seeing only
their own uploads whatever the filters say.

`GET /api/v1/images/facets` counts the caller's whole visible set for the given `state`, ignoring all list filters:

```json
{ "total": 26, "bytes": 23400000,
  "formats": { "jpeg": 7, "png": 16, "gif": 1, "webp": 2, "avif": 0 },
  "sources": [ { "key": "web", "kind": "web", "count": 13 },
               { "key": "token:1", "kind": "token", "tokenId": 1, "tokenName": "PicGo · 笔记本", "count": 5 },
               { "key": "cli", "kind": "cli", "count": 2 } ] }
```

- `formats` always has all five keys. `sources` lists only kinds with `count > 0`: `web` first, then tokens
  by count descending (ties by id; a revoked token keeps its name), then `cli`. `sources` is an array even
  when empty.
- `PATCH` applies every given field in one transaction. `filename`: trimmed, 1–200 characters, no control
  characters, no `/` or `\`. `folderId: null` moves the image to the root; an omitted field is left alone; an
  unknown folder is `404 not-found`. A body without any known field is `400 validation`. Purged images answer
  `404`, trashed images can still be edited and moved.
- Batch `move` resolves `folderId` once; an unknown folder fails the whole request with `404 not-found`.
  `folderId` null or absent means the root.
- Audit actions written by image changes: `image.upload` (target = image id, detail = destination folder
  id or empty; a dedupe hit stores nothing and writes no row), `image.rename` (detail = new name),
  `image.move` (detail = `old -> new`, `root` when there is no folder); rename and move only when the
  value actually changed. Existing: `image.trash`, `image.restore`, `image.purge`.
- Purging an image clears its folder reference; the tombstone keeps no folder.

Upload notes for third-party clients (PicGo web-uploader, ShareX, scripts):
- field name `file`; URL at JSON path `url` (also `links.markdown` etc.); optional field `folder` with the
  destination folder id (omit it for the root). Unknown folders are 404 and are never created implicitly.
  A legacy `tag` part is ignored and does not count against the 8-part limit (tags were removed in schema 3).
- errors carry a human `detail`; ShareX can use `{json:detail}` as ErrorMessage.

Example:

```sh
curl -H "Authorization: Bearer pb_…" -F file=@shot.png https://pics.example.com/api/v1/images
```

## 6. Folders

| Method & path | Auth | Request | Response |
|---|---|---|---|
| `GET /api/v1/folders` | any authenticated principal (upload-only tokens included) | — | `200 {"items": [Folder]}` flat list, ordered by name |
| `POST /api/v1/folders` | session, `manage` | `{"name": string, "parentId": string\|null}` | `201 Folder` |
| `PATCH /api/v1/folders/{id}` | session, `manage` | any non-empty subset of `{"name": string, "parentId": string\|null}` | `200 Folder` |
| `DELETE /api/v1/folders/{id}` | session, `manage` | — | `204`; `409 folder-not-empty` |

- Names: trimmed, 1–64 characters, no `/` or `\`, no control characters; unique among siblings
  case-insensitively (`409 folder-exists`). A sibling of another owner never conflicts.
- Depth: at most 3 levels. Creating below a depth-3 folder or moving a folder whose subtree would then
  exceed 3 levels answers `422 folder-depth`. Moving a folder into itself or one of its descendants is
  `400 validation`.
- `parentId: null` moves a folder to the root. Renaming and moving are one transaction.
- `DELETE` is allowed only when the folder has no subfolders and no active images. Trashed images that were
  in the deleted folder return to the root when restored (their `folderId` becomes `null` immediately).
- Folder writes are audited as `folder.create`, `folder.rename`, `folder.move`, `folder.delete`
  (target = folder id, detail = name or `old -> new`).
- Any authenticated principal of the owner may list folders, including upload-only tokens; only sessions and
  `manage` tokens may write.

## 7. Tokens (session only)

| Method & path | Request | Response |
|---|---|---|
| `GET /api/v1/tokens` | — | `200 {"items": [Token]}` (newest first, revoked included) |
| `POST /api/v1/tokens` | `{"name": 1–80 chars, "scopes": [≥1 of upload\|manage], "expiresInDays": 1–3650 \| null}` | `201 {"token": Token, "secret": "pb_…"}` — secret shown once; `422 invalid-scope` for unknown/removed scopes |
| `DELETE /api/v1/tokens/{id}` | — | `204` (revoke; idempotent) |

## 8. Settings, status, operations (session only)

| Method & path | Request | Response |
|---|---|---|
| `GET /api/v1/settings` | — | `200 Settings` |
| `PATCH /api/v1/settings` | partial Settings (writable fields only) | `200 Settings`; removed keys are unknown fields → `400 validation` |
| `GET /api/v1/status` | — | `200 Status` |
| `POST /api/v1/checks` | `{"mode": "quick"\|"full"}` | `200 CheckReport` |
| `POST /api/v1/snapshots` | — | `201 {"path", "images", "bytes", "createdAt"}` |
| `GET /api/v1/audit` | `?kind=<category>&action=<exact action>&limit=1–100&cursor=` | `200 {"items": [{"id", "at", "actor", "action", "target", "detail", "ip"}], "nextCursor"}` |
| `GET /api/v1/audit/actions` | — | `200 {"items": ["image.trash", …]}` distinct action strings, ascending |
| `GET /api/v1/openapi.json` | — (no auth) | `200` OpenAPI 3.1 document |

- Audit paging: newest first, `limit` defaults to 10 (max 100), `cursor` is the value of `nextCursor`.
  `action` keeps exactly that action string (unknown values simply return an empty page). `kind` keeps one
  category: `image` (`image.*`), `folder` (`folder.*`), `token` (`token.*`), `settings` (`settings.*`),
  `system` (`check.*`, `admin.*`, `snapshot`) or `session` (`login.*`, `logout`, `setup`, `setup.failure`);
  any other value is `400 validation`. `kind` and `action` combine.
- Audit retention: `auditRetentionDays`, enforced hourly in batches. The prune itself writes no audit row;
  it is only logged.
- Stored settings for removed keys (`maxPixels`, `defaultVisibility`, `uploadsPerMinute`,
  `sessionIdleMinutes`, `sessionAbsoluteHours`) are deleted by schema migration 3 and ignored if they reappear.

### Status

```json
{
  "version": "v0.0.1", "commit": "…", "goVersion": "go1.27.3", "sqliteVersion": "3.53.4",
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
