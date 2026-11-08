# Current UI capabilities and states

Evidence-based inventory of PicBerth as of 2026-10-22. Sources: `internal/httpapi/api.go` (routes),
`internal/httpapi/handlers_images.go`, `web/src/lib/{api,types,upload.svelte}.ts`, `docs/api.md`.
The production SPA implements the prototype approved as `d4827e1`. On 2026-11-08 `designs/picberth/` was
redesigned from scratch and awaits review; until it is approved, `d4827e1` stays the visual truth for production. The redesign drops 公开 / 私有 (every image is public); production and the API still carry visibility. The earlier 「暗房」 design was
**rejected** by the user on 2026-10-21 and no longer exists in the code; it has no authority.

## Users and primary jobs

- One administrator (single account created at setup) managing a personal image host.
- Upload images (web: drop / paste / pick; external tools: PicGo, PicList, ShareX, curl via API tokens) and copy
  their links in URL / Markdown / HTML / BBCode form.
- Find an existing image again and copy its link.
- Manage images: change visibility, move to trash, restore, purge; act on many images at once.
- Administer: API tokens, upload/retention/session settings, health checks and snapshots, audit log.

## Surfaces

| Surface ID | Route | Purpose | Primary actions | Evidence |
|---|---|---|---|---|
| setup | `/setup` | create first admin | enter setup code, username, password | `POST /api/v1/setup`, `screens/Setup.svelte` |
| login | `/login` | sign in | username + password | `POST /api/v1/session`, `screens/Login.svelte` |
| not-found | any unknown path | tell the user the page does not exist | go back to the library | `screens/NotFound.svelte` |
| library | `/`, `/library` | browse/search active images, upload queue | upload, search, filter, sort, select, batch, open detail | `screens/Library.svelte`, `components/library/` |
| image-detail | docked column / drawer / sheet over library/trash | one image | copy link (4 formats), visibility toggle, rename, edit tags, download, trash / restore / purge, prev/next | `components/library/DetailPanel.svelte` |
| trash | `/trash` | trashed images | restore, purge (confirm), batch | `screens/Library.svelte` (trash mode) |
| tokens | `/tokens` | API tokens | create (name, scopes, expiry), reveal once + integration snippets, revoke (inline second press) | `screens/Tokens.svelte`, `TokenDialog.svelte` |
| settings | `/settings` | business settings | edit, save / discard, leave guard | `screens/Settings.svelte` |
| status | `/status` | health | quick / full check, snapshot, view warnings, figures, audit log | `screens/Status.svelte` |

## Domain objects and terminology

| Object | User-visible fields | Allowed actions (web session) | Rules | Evidence |
|---|---|---|---|---|
| Image (图片) | filename, id, url, mime/ext, size, width×height, frames (animated), sha256, visibility 公开/私有, broken, duplicate, source (网页 / 令牌名 / 命令行), tags, createdAt, trashedAt, links{url, markdown, html, bbcode} | upload, set visibility, rename, edit tags, trash, restore, purge, download raw | link bytes never change; purged IDs never reused; private images not publicly reachable; `hasGps` is returned by the API but not shown in the console | `lib/types.ts` Image, `handlers_images.go` |
| Upload item | file name, progress, status queued/uploading/processing/done/error, error text, retryable | cancel, retry (retryable only), remove, copy link | concurrency 3; client pre-check of size/format; 429 auto-retry | `lib/upload.svelte.ts` |
| Token (令牌) | name, prefix, scopes 上传/只读/管理, createdAt, lastUsedAt, expiresAt, revokedAt | create, revoke | secret shown once; tokens cannot manage tokens/settings | `lib/types.ts` Token, `api.md` |
| Settings (设置) | maxUploadBytes, maxPixels, allowedFormats, dedupe, defaultVisibility, trashRetentionDays, uploadsPerMinute, sessionIdleMinutes, sessionAbsoluteHours; read-only publicUrl, imageUrl, hardMaxUploadBytes | edit writable fields | — | `lib/types.ts` Settings |
| Status (状态) | images {active, trashed, bytes, broken}, database, disk, lastSnapshot, lastCheck, warnings, versions | run check (quick/full), create snapshot | — | `lib/types.ts` Status |
| Audit entry | at, actor, action, target, detail, ip | read, paginate | — | `GET /api/v1/audit` |

## Library query capabilities (what "find / filter" can do today)

`GET /api/v1/images` supports, all combined with AND: `state` (active | trashed), `q` (filename or tag
substring, ID or SHA-256 prefix), `format` (any-of jpeg | png | gif | webp | avif; repeatable and/or
comma-separated), `tag` (repeatable, image must have all), `source` (repeatable any-of: `web`, `cli`,
`token:<id>`), `visibility` (public | private), `since` / `until` (RFC 3339, upload time,
`since` inclusive, `until` exclusive), `sort` (new | old | large | small | name), cursor pagination
(`limit`, `cursor`). No size range.

`GET /api/v1/images/facets?state=active|trashed` returns counts for the whole visible set of that state
(ignoring other filters): total, bytes, per format, per visibility, per source (web, each token with
its name, cli) and per tag. Use it for filter counts and tag/source pickers.

Images carry `tags` (up to 20, 1–32 characters each; no comma, no leading `#`). `PATCH /api/v1/images/{id}`
accepts any non-empty subset of `visibility`, `filename` (rename; link and id never change) and `tags` (replaces
the full set). Upload accepts repeated `tag` parts. Batch (`POST /api/v1/images/batch`, ≤ 500 ids):
`trash | restore | purge | visibility` — no batch tagging or renaming.

## Not supported by the backend (would need new API work)

Albums / folders, notes/descriptions, replacing an image's bytes, view / traffic statistics, size-range
filter, batch tag / rename, tag rename or merge across images (tags are only edited per image), multiple users.

## State matrix

| Flow | Ready | Loading | Empty | Error | Disabled | Permission | Progress / cancel / retry |
|---|---|---|---|---|---|---|---|
| library list | list with cursor | first page + load more | none at all; filtered-empty with clear filters | load failed + retry | — | 401 → login | infinite scroll |
| upload | done with link | — | — | precheck rejection, 413/415/422/429/5xx messages | — | 401 → login | % progress, processing, cancel, retry (retryable), auto-retry on short 429 |
| batch actions | toast with 撤销 for trash/restore | — | — | partial failure counts | — | — | — |
| purge | confirm dialog | — | — | — | — | — | — |
| tokens | rows | — | 还没有令牌 | load failed | — | — | — |
| settings | form | — | — | validation per field, save error | save while unchanged | — | unsaved bar, leave guard |
| status | figures | — | — | load failed | buttons while running | — | check elapsed time |

## Interaction and accessibility behavior

- Keyboard: `U` opens file picker, `/` focuses search, arrows move between images, Space selects, Enter opens,
  Esc clears selection / closes detail, ←/→ in detail.
- Pointer: window-wide drag & drop and paste; Ctrl/⌘-click toggle, Shift-click range; touch long-press select,
  swipe in detail. Phones: 选择 toggle starts selection mode; the upload button is the last header control.
- Table row actions show on hover or focus and always on touch. Default library view is the table on every
  width; the choice is remembered (`picberth.view`).
- Tag editor: up to 20 tags, 32 characters each (server limit); a leading `#` is stripped. The add field is a
  combobox listing existing tags with counts.
- Status figures are one card; the settings unsaved bar floats at the bottom (above the tab bar on phones).
- Live region announces upload completions/errors; icon buttons have labels.
- Overlays (modal, drawer, sheet) animate in and out; all motion is disabled under `prefers-reduced-motion`.
- zh-CN only; light and dark follow the system, with a manual override (跟随系统 / 浅色 / 深色, `picberth.theme`).

## Platform, viewport, and runtime constraints

- SPA embedded in a Go binary; strict CSP (`script-src 'self'`, images from self/blob/data/image origin).
- Svelte 5 + Vite, no UI libraries. Target Chrome/Edge 111+, Firefox 114+, Safari 16.4+.
- Breakpoints: full sidebar from 1100 px, icon rail 721–1099 px, full-bleed panel with a bottom tab bar at
  ≤ 720 px. Library phone rows at ≤ 720 px; image detail docked column ≥ 1280 px (380 px), right drawer
  721–1279 px, bottom sheet on phones. The header controls adapt to the width of the panel content, not the
  viewport.
- The server serves the SPA only for its known routes: an unknown URL requested directly gets a plain 404; unknown
  in-app paths show the not-found screen.

## Content authority currently evidenced

- Domain terminology in production code: 图片, 上传, 图库, 回收站, 令牌, 设置, 状态, 公开, 私有, 链接, 复制,
  移到回收站, 恢复, 永久删除, 撤销, 动图, 损坏, 上传 / 只读 / 管理.
- State/error strings in `lib/api.ts` (`errorMessage`) and `lib/upload.svelte.ts` must stay compatible.
- Rejected metaphor copy (底片, 帧, 冲洗, 显影, 暗房已就绪, 图片的泊位 …) has no authority.
- Explicitly approved marketing copy: none.

## Known unknowns

- Which management capabilities the user expects beyond the current API (albums, notes, batch tagging and
  renaming, tag merge, size-range filter, statistics …).
