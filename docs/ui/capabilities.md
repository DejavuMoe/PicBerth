# Current UI capabilities and states

Evidence-based inventory of PicBerth as of 2026-11-10. Sources: `internal/httpapi/api.go` (routes),
`internal/httpapi/handlers_*.go`, `web/src/lib/{api,types,upload.svelte,library.svelte,folders}.ts`, `docs/api.md`.
The production SPA implements revision 7 (`docs/ui/r7-revision.md`, design commit `01d0aea`) on top of the folders
revision approved as `c3bb4d0` (2026-11-09, `docs/ui/folders-revision.md`) and the redesign `ead4953`. Every image is
public; there is no visibility concept in the API or the console. Tags do not exist. The earlier 「暗房」 design was
**rejected** by the user on 2026-10-21 and no longer exists in the code; it has no authority.

## Users and primary jobs

- One administrator (single account created at setup) managing a personal image host.
- Upload images (web: drop / paste / pick; external tools: PicGo, PicList, ShareX, curl via API tokens) and copy
  their links in URL / Markdown / HTML / BBCode form.
- Find an existing image again and copy its link.
- Organise images in folders (at most 3 levels); rename, move, trash, restore, purge; act on many images at once.
- Administer: API tokens, upload / retention settings, health checks and snapshots, operation log.

## Surfaces

| Surface ID | Route | Purpose | Primary actions | Evidence |
|---|---|---|---|---|
| setup | `/setup` | create first admin | enter setup code (one input shown as three groups of four characters), username, password | `POST /api/v1/setup`, `screens/Setup.svelte` |
| login | `/login?next=` | sign in | username + password, then return to `next` | `POST /api/v1/session`, `screens/Login.svelte` |
| not-found | any unknown path | tell the user the page does not exist | go back to the library | `screens/NotFound.svelte` |
| library | `/library`, `/library/<folder path>` (`/` is replaced with `/library`) | browse the current folder as a grid (default) or a list; in the grid the direct subfolders are buttons above the images (hidden while searching / filtering; the list shows images only); search / filter it with its subfolders; uploads in a floating panel mounted in the shell | upload, search, filter, sort, select, batch, move, open detail, open viewer | `screens/Library.svelte`, `components/library/`, `components/UploadDock.svelte` |
| folder tree | sidebar top ≥ 1100 px on every route, popover below | folders of the library; root row 图片 with the create button; 还没有文件夹 + 新建 when empty | open, create, rename, move, delete (empty only), drop images onto a node | `components/library/FolderPanel.svelte`, `FolderMenu.svelte`, `lib/tree.svelte.ts`, `components/Shell.svelte` |
| image-detail | docked column / drawer / sheet over library/trash | one image; the preview opens the viewer | copy link (4 formats, last-used format remembered), rename, move (文件夹 row), download, trash / restore / purge, prev/next | `components/library/DetailPanel.svelte` |
| viewer | overlay over library/trash | one image at window size | previous / next, 复制链接 (active images), fit / 原始大小, Esc | `components/library/Viewer.svelte` |
| command-palette | `Ctrl/⌘ K` anywhere in the console | find images and folders, run commands, go to a page | images by name, ID or SHA-256 prefix via `GET /api/v1/images?folder=root&recursive=true&q=…&limit=6`; folders matched client-side; commands 上传图片, 新建文件夹 (≥ 1100 px), 新建令牌; go to any page and API 文档; theme; Enter opens, Ctrl/⌘ Enter copies the link | `components/Palette.svelte`, `lib/palette.svelte.ts` |
| trash | `/trash` | trashed images across folders, with a 文件夹 column | restore, purge (confirm), batch | `screens/Library.svelte` (trash mode) |
| tokens | `/tokens` | API tokens | create (name, scopes, expiry), reveal once + integration snippets, revoke (inline second press), open API 文档 | `screens/Tokens.svelte`, `components/admin/TokenCreate.svelte`, `TokenReveal.svelte` |
| settings | `/settings` | business settings | edit, save / discard, leave guard; the two read-only addresses copy on click | `screens/Settings.svelte` |
| status | `/status` | health and operation log | quick / full check, snapshot, warnings, figures (incl. 损坏) and one disk-usage bar (图片 / 数据库 / 其他 / 可用), operation log filter and 加载更多 | `screens/Status.svelte` |
| api-docs | `/docs/api` (no session needed) | API reference in the console shell: contents are the sidebar (tabs below 1100 px), current section marked | read, copy examples, open OpenAPI | `screens/DocsApi.svelte`, `GET /api/v1/openapi.json` |

## Domain objects and terminology

| Object | User-visible fields | Allowed actions (web session) | Rules | Evidence |
|---|---|---|---|---|
| Image (图片) | filename, id, url, mime/ext, size, width×height, frames (animated), sha256, broken, duplicate, source (网页 / 令牌名 / 命令行), folder, createdAt, trashedAt, links{url, markdown, html, bbcode} | upload, rename, move, trash, restore, purge, download raw | link bytes never change, also when moved; purged IDs never reused; dedupe is per folder; `hasGps` is returned by the API but not shown | `lib/types.ts` Image, `handlers_images.go` |
| Folder (文件夹) | name, parent, depth, imageCount (direct, active) | create, rename, move, delete when empty | depth ≤ 3 counting the moved subtree; names 1–64 characters, no `/` or `\`, unique among siblings case-insensitively; opaque ids, the address uses names | `lib/folders.ts`, `handlers_folders.go` |
| Upload item | file name, folder, progress, status queued/uploading/processing/done/error, error text, retryable | cancel, retry (retryable only), remove, copy link | concurrency 3; client pre-check of size/format; 429 auto-retry | `lib/upload.svelte.ts` |
| Token (令牌) | name, prefix, scopes 上传/管理, createdAt, lastUsedAt, expiresAt, revokedAt | create, revoke | secret shown once; tokens cannot manage tokens/settings | `lib/types.ts` Token, `api.md` |
| Settings (设置) | maxUploadBytes, allowedFormats, dedupe, trashRetentionDays, auditRetentionDays; read-only publicUrl, imageUrl, hardMaxUploadBytes | edit writable fields | upload rate (60/min), session lifetimes (7 days idle, 30 days absolute) and max pixels (50 MP) are fixed | `lib/types.ts` Settings |
| Status (状态) | images {active, trashed, bytes, broken}, database, disk, lastSnapshot, lastCheck, warnings, versions | run check (quick/full), create snapshot | — | `lib/types.ts` Status |
| Audit entry (操作记录) | at, actor, action, target, detail, ip | read, filter by category, page by 10 | read-only; pruned after `auditRetentionDays` | `GET /api/v1/audit` |

## Library query capabilities (what "find / filter" can do today)

`GET /api/v1/images` supports, all combined with AND: `state` (active | trashed), `folder` (`root` or a folder
id) with `recursive` (include descendants), `q` (filename substring, ID or SHA-256 prefix), `format` (any-of
jpeg | png | gif | webp | avif; repeatable and/or comma-separated), `source` (repeatable any-of: `web`, `cli`,
`token:<id>`), `since` / `until` (RFC 3339, upload time, `since` inclusive, `until` exclusive), `sort` (new | old |
large | small | name), cursor pagination (`limit`, `cursor`). No size range. The console browses a folder without
`recursive` and adds `recursive=true` when a search or filter is active.

`GET /api/v1/images/facets?state=active|trashed` returns counts for the whole visible set of that state
(ignoring other filters): total, bytes, per format and per source (web, each token with its name, cli). Use it
for filter counts and the source picker.

`PATCH /api/v1/images/{id}` accepts any non-empty subset of `filename` (rename; link and id never change) and
`folderId`. Batch (`POST /api/v1/images/batch`, ≤ 500 ids): `trash | restore | purge | move` — no batch renaming.

## Not supported by the backend (would need new API work)

Albums across folders, notes/descriptions, replacing an image's bytes, view / traffic statistics, size-range
filter, batch rename, multiple users.

## State matrix

| Flow | Ready | Loading | Empty | Error | Disabled | Permission | Progress / cancel / retry |
|---|---|---|---|---|---|---|---|
| library list | list with cursor | first page + load more | 还没有图片; 这里没有图片 in a folder; filtered-empty with 清除筛选 | load failed + retry; 文件夹不存在 | — | 401 → login | infinite scroll |
| folder tree | tree | 加载中 | 还没有文件夹 + 新建 | 文件夹加载失败 + 重试 | 新建子文件夹 at depth 3 (最多 3 层); 删除 while 不为空; move targets past depth 3 | — | inline create / rename (Enter / Esc); delete and move with 撤销 |
| upload | done with link | — | — | precheck rejection, 404 folder, 413/415/422/429/5xx messages | — | 401 → login | % progress, processing, cancel, retry (retryable), auto-retry on short 429 |
| batch actions | toast with 撤销 for trash/restore/move | — | — | partial failure counts | — | — | — |
| purge | confirm dialog | — | — | — | — | — | — |
| tokens | rows | — | 还没有令牌 | load failed | — | — | — |
| settings | form | — | — | validation per field, save error | save while unchanged | — | unsaved bar, leave guard |
| status | figures | — | — | load failed | buttons while running | — | check elapsed time |
| operation log | rows, 加载更多 | 加载中 | empty filter result | load failed | — | — | paging by 10 |

## Interaction and accessibility behavior

- Keyboard: `U` opens file picker, `/` focuses search, `Ctrl/⌘ K` opens the command palette, arrows move between
  images, Space selects, Enter opens, Esc clears selection / closes the palette, viewer or detail, ←/→ in the viewer
  and detail; Enter saves and Esc cancels inline folder names.
- Pointer: window-wide drag & drop and paste upload into the current folder; images (the whole selection when the
  dragged image is selected) can be dropped on a tree node or a breadcrumb ancestor; Ctrl/⌘-click toggle,
  Shift-click range; double-click opens the viewer; touch long-press select, swipe in detail. Folders are not
  draggable; grid folder buttons accept dropped images. Phones: 选择 in the
  header 显示 menu starts selection mode; batch actions float above the tab bar; the upload button is the last
  header control.
- Table row actions show on hover or focus and always on touch. Default library view is the grid on every width;
  the choice is remembered (`picberth.view` = `grid` | `list`; the old value `table` is read as unset).
- The uploads panel floats bottom right (above the tab bar on phones) and can be collapsed; the list does not move.
- Status figures are one card; the settings unsaved bar floats at the bottom (above the tab bar on phones).
- Live region announces upload completions/errors; icon buttons have labels.
- Overlays (modal, drawer, sheet) animate in and out; all motion is disabled under `prefers-reduced-motion`.
- zh-CN only; light and dark follow the system, with a manual override (跟随系统 / 浅色 / 深色, `picberth.theme`).

## Platform, viewport, and runtime constraints

- SPA embedded in a Go binary; strict CSP (`script-src 'self'`, images from self/blob/data/image origin).
- Svelte 5 + Vite, no UI libraries. Target Chrome/Edge 111+, Firefox 114+, Safari 16.4+.
- Breakpoints: full sidebar (248 px) with the folder tree inside it from 1100 px, icon rail and tree popover
  721–1099 px, full-bleed
  panel with a bottom tab bar at ≤ 720 px. Image detail docked column ≥ 1440 px, right drawer 721–1439 px, bottom
  sheet on phones. The header controls adapt to the width of the panel content, not the viewport.
- The server serves the SPA for its known routes and any `/library/...` path: an unknown URL requested directly
  gets a plain 404; unknown in-app paths show the not-found screen.

## Content authority currently evidenced

- Domain terminology in production code: 图片, 文件夹, 上传, 回收站, 令牌, 设置, 状态, 操作记录, 链接, 复制,
  移动, 移到回收站, 恢复, 永久删除, 撤销, 动图, 损坏, 上传 / 管理.
- State/error strings in `lib/api.ts` (`errorMessage`) and `lib/upload.svelte.ts` must stay compatible.
- Rejected metaphor copy (底片, 帧, 冲洗, 显影, 暗房已就绪, 图片的泊位 …) has no authority.
- Explicitly approved marketing copy: none.

## Known unknowns

- Which management capabilities the user expects beyond the current API (notes, batch renaming, size-range
  filter, statistics …).
- The approved prototype's API reference identified folders by path; the API identifies them by opaque id, and the
  production reference documents ids.
