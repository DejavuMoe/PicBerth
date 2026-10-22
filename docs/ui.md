# PicBerth web console

The console is the Svelte 5 SPA in `web/`, embedded in the Go binary. Its visual and interaction truth is the
approved prototype in `designs/picberth/` (single console shell `Shell` in `app.jsx`; styles in `styles.css`,
admin pages in `admin.jsx`; design approved 2026-10-22 as `fe9ca37`, implemented in `16297e6`). Approved copy is
the prototype's copy; strings it does not cover come from the production code (`web/src/lib/api.ts` error
messages, sign-in field labels). Behaviour, states and keyboard support are inventoried in
`docs/ui/capabilities.md`; durable constraints are in `docs/product/constraints.md`.
The earlier 「暗房」 design was rejected and no longer exists in the code.

## Visual system

- Linear-like. Near-black app background; every screen sits in an inset content panel (radius 12, hairline
  border). Hairlines replace shadows.
- Light and dark themes follow the system by default. The user can override with 跟随系统 / 浅色 / 深色, stored
  in `picberth.theme` (absent = system).
- Fonts: self-hosted Inter Variable (Latin subsets, OFL) with system CJK fonts.
- Lime `#e4f222` appears only on the single primary action of a view: 上传, 新建令牌, 保存, 创建, the sign-in and
  setup submit buttons, and the phone upload button.
- Menus, selects, comboboxes and checkboxes are drawn by the product in a floating layer. There are no native
  select popups or datalists.

## Shell and routes

From 1100 px: sidebar with brand, navigation without group headings, a warning dot on 状态 when there are
warnings, and an account row at the bottom with the theme options and 退出登录. 721–1099 px: the sidebar is an
icon rail (tooltips, warning dot). ≤ 720 px: a full-bleed panel with a sticky header and a fixed bottom tab bar
图片 / 回收站 / 令牌 / 更多, where 更多 opens a sheet with 设置, 状态, the theme options and 退出登录. There is no
top bar, no breadcrumbs, no navigation counts and no disk meter.
Routes: `/library` (`/` redirects), `/trash`, `/tokens`, `/settings`, `/status`, `/login`, `/setup`; unknown
in-app paths show a not-found page. The server serves the SPA only for these routes, so a direct request for any
other URL gets a plain 404. Sign-in and setup are plain centered cards. Overlays (modal, drawer, sheet) animate in
and out; all motion is disabled under `prefers-reduced-motion`.

## Library and trash

- Every screen has a 48 px panel header with the title and that screen's controls. Library: search (`/`), 筛选
  (one popover: 格式 / 可见性 / 上传时间 / 来源 / 标签), sort menu, list / grid toggle, 上传 (hidden when the library
  is empty). Trash: 清空回收站. Tablet: search collapses to an icon. Phones: search icon, 选择 toggle and a ⋯ sheet
  with sort, view and filters; a floating upload button.
- Active filters show as removable chips under the header.
- Selecting images replaces the header controls in place with 已选 N 张, 全选 / 取消全选, 复制链接, 可见性,
  移到回收站 (trash: 恢复, 永久删除) and ×. On phones the bulk bar sits above the tab bar.
- List (default on every width): no column header; 40 px rows with a checkbox column that fades in, thumbnail,
  name (with a lock for private images and tag chips), then dimensions, size and relative time. Columns hide as
  the width narrows. Row actions: copy in the current link format, and a ⋯ menu (复制为 URL / Markdown / HTML /
  BBCode, 打开原图, 下载, 设为私有 / 公开, 移到回收站). They show on hover or focus and always on touch.
  No footer count; paging loads automatically (加载中…, with a retry button on failure).
- Grid: tiles with radius 12, 4:3 cover image, name inside, lock and GIF overlays.
- Image detail: ≥ 1280 px a 380 px column inside the panel; 721–1279 px a right drawer; phones a bottom sheet.
  Top bar ‹ › ×. Order: preview (checkerboard for transparent images, 打开原图), the filename as an editable
  title, meta line, link tabs URL / Markdown / HTML / BBCode with a copy field, then the property list: 公开
  (switch), 标签 (chips and an 添加标签 combobox listing existing tags with counts; new tags up to 32
  characters, leading `#` stripped, max 20), 来源, 上传时间, 删除时间 (trash), ID, SHA-256. Footer 下载 /
  移到回收站 (trash: 恢复 / 永久删除).
- Upload dialog; purge confirm (the only confirmation in the app; trash and visibility changes offer 撤销
  instead).
- Filters and sort live in the URL: `q`, `format` (repeatable), `visibility`, `date`
  (`today | 7d | 30d | older`), `source` (repeatable), `tag` (repeatable), `sort`. Dates are turned into
  `since` / `until` when the list is requested. The list is paged with `cursor`; the next page loads as you scroll.
- Counts in the filter popover come from `GET /api/v1/images/facets`, refreshed after uploads and mutations.
- The table / grid choice (`picberth.view`, default table), the link format (`picberth.linkFormat`) and the theme
  (`picberth.theme`) are remembered in `localStorage`.
- Keyboard: `/` search, `U` file picker, Esc clears the selection, then closes the image detail, ← / → move
  between images when the detail is open (docked or overlay), Enter opens and Space selects on a focused row or
  tile; Ctrl / ⌘ and Shift click select. Touch: long-press or the 选择 toggle starts selection on phones.
  Window-wide drag and drop and paste open the upload dialog.

## Status and settings

- Settings: centred column with section headings. Number fields have a fixed width with the unit inside the
  field. Unsaved changes show a floating bar at the bottom of the screen (放弃 / 保存), above the tab bar on phones.
- Status: a warning alert with an inline 创建快照; five figures in one card (图片, 占用空间, 回收站, 数据库,
  磁盘剩余 with a usage bar); maintenance rows; run information (including 已运行); audit log.
- Tokens: a list with a small header row; the expiry is picked with the product-drawn select.

## Code layout

| Path | Content |
|---|---|
| `web/src/app.css` | tokens (light and dark through `data-theme` on `<html>`), primitives, shell, admin pages |
| `web/src/lib/` | `api.ts` (endpoints), `upload.svelte.ts` (queue), `library.svelte.ts` (list state and actions), `facets`, `health`, `session`, `router`, `toast`, `theme`, `modal` (overlay stack, focus trap), formatting helpers |
| `web/src/components/` | shell, menus, selects and comboboxes (floating layer), overlays, upload dialog, shared controls |
| `web/src/components/library/` | toolbar, filters, table, grid, rows, detail panel |
| `web/src/screens/` | route screens (library and trash, tokens, settings, status, login, setup, not found) |

Inter Variable is self-hosted; CJK text uses system fonts. No external assets are loaded. The CSP in
`internal/httpapi/server.go` applies (no inline scripts or event-handler strings).
