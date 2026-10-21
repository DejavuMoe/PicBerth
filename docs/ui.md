# PicBerth web console

The console is the Svelte 5 SPA in `web/`, embedded in the Go binary. Its visual and interaction truth is the
approved prototype in `designs/picberth/`, **direction A** (`ShellA` in `app.jsx`; styles in `styles.css`,
admin pages in `admin.jsx`). Approved copy is the prototype's copy; strings it does not cover come from the
production code (`web/src/lib/api.ts` error messages, sign-in field labels). Behaviour, states and keyboard
support are inventoried in `docs/ui/capabilities.md`; durable constraints are in `docs/product/constraints.md`.
The earlier 「暗房」 design was rejected and no longer exists in the code.

## Shell and routes

Sidebar (brand, navigation groups 图片 / 接入 / 系统 with counts and the 状态 warning dot, usage meter), top bar
(breadcrumbs, theme toggle, account menu). Up to 900 px the sidebar becomes a left drawer opened from a menu
button. Routes: `/library` (`/` redirects), `/trash`, `/tokens`, `/settings`, `/status`, `/login`, `/setup`,
unknown paths show a not-found page. Sign-in and setup are plain centered cards.

## Library and trash

- Page head with count and size, toolbar (search, filter menus 格式 / 可见性 / 上传时间 / 来源 / 标签, 含位置
  toggle, sort, table / grid toggle), filter chips, bulk bar, table / grid / phone rows (≤ 720 px), detail
  drawer, upload dialog, purge confirm (the only confirmation in the app; trash and visibility changes offer
  撤销 instead).
- Filters and sort live in the URL: `q`, `format` (repeatable), `visibility`, `gps`, `date`
  (`today | 7d | 30d | older`), `source` (repeatable), `tag` (repeatable), `sort`. Dates are turned into
  `since` / `until` when the list is requested. The list is paged with `cursor` and loads more on scroll.
- Counts in menus, sidebar and page head come from `GET /api/v1/images/facets`, refreshed after uploads and
  mutations.
- The table / grid choice (`picberth.view`), the link format (`picberth.linkFormat`) and the theme
  (`picberth.theme`) are remembered in `localStorage`.
- Keyboard: `/` search, `U` file picker, Esc clears the selection or closes the detail drawer, ← / → in the
  detail drawer, Enter opens and Space selects on a focused row or tile; Ctrl / ⌘ and Shift click select. Touch:
  long-press starts selection. Window-wide drag and drop and paste open the upload dialog.

## Code layout

| Path | Content |
|---|---|
| `web/src/app.css` | tokens (light and dark through `data-theme` on `<html>`), primitives, shell, admin pages |
| `web/src/lib/` | `api.ts` (endpoints), `upload.svelte.ts` (queue), `library.svelte.ts` (list state and actions), `facets`, `health`, `session`, `router`, `toast`, `theme`, `modal` (overlay stack, focus trap), formatting helpers |
| `web/src/components/` | shell, menus, overlays, upload dialog, shared controls |
| `web/src/components/library/` | toolbar, filters, table, grid, rows, detail drawer |
| `web/src/screens/` | route screens (library and trash, tokens, settings, status, login, setup, not found) |

System fonts only, no external assets; the CSP in `internal/httpapi/server.go` applies (no inline scripts or
event-handler strings).
