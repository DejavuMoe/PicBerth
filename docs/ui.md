# PicBerth web console

The console is the Svelte 5 SPA in `web/`, embedded in the Go binary. Its visual and interaction truth is the
approved prototype in `designs/picberth/` (single console shell `Shell` in `app.jsx`; styles in `styles.css`,
admin pages in `admin.jsx`; revised and approved 2026-10-22, implemented in `8952c1e`). Approved copy is the prototype's copy; strings it does not cover come from the
production code (`web/src/lib/api.ts` error messages, sign-in field labels). Behaviour, states and keyboard
support are inventoried in `docs/ui/capabilities.md`; durable constraints are in `docs/product/constraints.md`.
The earlier 「暗房」 design was rejected and no longer exists in the code.

## Shell and routes

From 1100 px: sidebar (brand, navigation groups 图片 / 接入 / 系统 with counts and the 状态 warning badge, 磁盘剩余
meter at the bottom) and top bar (breadcrumbs, theme toggle, account menu). 721–1099 px: the sidebar is an icon
rail (tooltips, warning dot). ≤ 720 px: no sidebar; top bar with brand and account menu, and a fixed bottom tab
bar 图片 / 回收站 / 令牌 / 更多, where 更多 opens a sheet with 设置, 状态 (warning count) and a 深色模式 switch.
Routes: `/library` (`/` redirects), `/trash`, `/tokens`, `/settings`, `/status`, `/login`, `/setup`; unknown
in-app paths show a not-found page. The server serves the SPA only for these routes, so a direct request for any
other URL gets a plain 404. Sign-in and setup are plain centered cards. Overlays (modal, drawer, sheet) animate in
and out; all motion is disabled under `prefers-reduced-motion`.

## Library and trash

- Page head with count and size. Toolbar adapts to the width the page content gets: from 940 px one menu per
  filter (格式 / 可见性 / 上传时间 / 来源 / 标签 / 含位置), narrower a single 筛选 popover; sort is icon-only below
  720 px; table / grid toggle. Phones: search, 筛选 button (sort and filters in a sheet), 选择 toggle, view toggle.
- Filter chips; bulk bar (on desktop it replaces the toolbar while images are selected; on phones it floats
  above the tab bar; actions 复制链接, 可见性, 移到回收站 / 恢复 / 永久删除, 全选 / 取消全选).
- Table (default view on every width): columns hide as the table narrows (尺寸 below 900 px, 可见性 below 760 px,
  time below 560 px) and that information moves into the line under the file name; sticky header; row actions
  (copy in the current link format, ⋯ menu with 复制为 URL / Markdown / HTML / BBCode, 打开原图, 下载, 设为私有 /
  公开, 移到回收站) show on hover or focus and
  always on touch. Grid and phone rows (≤ 720 px) as before.
- Image detail: docked column ≥ 1280 px beside the list, right drawer 721–1279 px, full-height bottom sheet on
  phones. Order: preview (打开原图), facts line, 链接 (format + copy), 公开访问 switch, 文件名 (rename), 标签 (up
  to 20 tags of 32 characters; a leading `#` is stripped), 信息 (来源, 上传时间, 删除时间 in trash, 位置信息, ID,
  SHA-256), footer 下载 / 移到回收站 (trash: 恢复 / 永久删除).
- Upload dialog; purge confirm (the only confirmation in the app; trash and visibility changes offer 撤销
  instead).
- Filters and sort live in the URL: `q`, `format` (repeatable), `visibility`, `gps`, `date`
  (`today | 7d | 30d | older`), `source` (repeatable), `tag` (repeatable), `sort`. Dates are turned into
  `since` / `until` when the list is requested. The list is paged with `cursor` and loads more on scroll.
- Counts in menus, sidebar and page head come from `GET /api/v1/images/facets`, refreshed after uploads and
  mutations.
- The table / grid choice (`picberth.view`, default table), the link format (`picberth.linkFormat`) and the theme
  (`picberth.theme`) are remembered in `localStorage`.
- Keyboard: `/` search, `U` file picker, Esc clears the selection, then closes the image detail, ← / → move
  between images in the detail, Enter opens and Space selects on a focused row or tile; Ctrl / ⌘ and Shift click
  select. Touch: long-press or the 选择 toggle starts selection on phones. Window-wide drag and drop and paste
  open the upload dialog.

## Status and settings

- Status figures are one card with cells. Settings show an unsaved-changes bar that floats at the bottom of the
  screen, above the tab bar on phones.

## Code layout

| Path | Content |
|---|---|
| `web/src/app.css` | tokens (light and dark through `data-theme` on `<html>`), primitives, shell, admin pages |
| `web/src/lib/` | `api.ts` (endpoints), `upload.svelte.ts` (queue), `library.svelte.ts` (list state and actions), `facets`, `health`, `session`, `router`, `toast`, `theme`, `modal` (overlay stack, focus trap), formatting helpers |
| `web/src/components/` | shell, menus, overlays, upload dialog, shared controls |
| `web/src/components/library/` | toolbar, filters, table, grid, rows, detail panel |
| `web/src/screens/` | route screens (library and trash, tokens, settings, status, login, setup, not found) |

System fonts only, no external assets; the CSP in `internal/httpapi/server.go` applies (no inline scripts or
event-handler strings).
