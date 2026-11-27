# PicBerth web console

The console is the Svelte 5 SPA in `web/`, embedded in the Go binary. Its visual and interaction truth is the
approved prototype in `designs/picberth/` (single console shell `Shell` in `app.jsx`; styles in `styles.css`,
admin pages in `admin.jsx`, API reference in `api-docs.html`; folders revision approved 2026-11-09 as `c3bb4d0`, see `docs/ui/folders-revision.md`). Design rules are in `DESIGN.md`. Approved copy is
the prototype's copy; strings it does not cover come from the production code (`web/src/lib/api.ts` error
messages, sign-in field labels). Behaviour, states and keyboard support are inventoried in
`docs/ui/capabilities.md`; durable constraints are in `docs/product/constraints.md`.
The earlier 「暗房」 design was rejected and no longer exists in the code.

## Visual system

- Linear-like. Every screen sits in an inset content panel (radius 12). All lines are 0.5 px; only floating
  layers have a shadow. Colours are Radix Colors (gray, lime, red, amber, green); three type sizes (12 / 13 / 16).
- Motion only on overlays, the detail column, toasts and switches (250 ms ease-out); hover and selection are
  instant.
- Light and dark themes follow the system by default. The user can override with 跟随系统 / 浅色 / 深色, stored
  in `picberth.theme` (absent = system).
- Fonts: self-hosted Inter Variable (Latin subsets, OFL) with system CJK fonts.
- Lime (Radix lime 9) appears only on the single primary action of a view: 上传, 新建令牌, 保存, 创建, the sign-in
  and setup submit buttons, and the phone header upload button.
- Menus, selects, comboboxes and checkboxes are drawn by the product in a floating layer. There are no native
  select popups or datalists.

## Shell and routes

From 1100 px: sidebar with brand, a 搜索 (Ctrl K) button that opens the command palette, then 最近, the folder tree
under 图片, 回收站 with its count, and at the bottom 系统 and the account menu (theme options and 退出). 系统 carries a
dot only for warnings that need action (a broken image, a failed check); a missing or old snapshot shows on the 快照
row only. 721–1099 px: the sidebar is an icon rail (tooltips, the same dot). ≤ 720 px: a full-bleed panel with a
sticky header and a fixed bottom tab bar 最近 / 图片 / 回收站 / 系统. There is no top bar and no disk meter.
系统 groups 状态, 令牌 and 设置 as tabs in the page header (a row under the header on phones); each tab keeps its
own route.

Routes: `/recent`, `/library` and `/library/<folder path>` (folder names joined by `/`, each segment URL-encoded, e.g.
`/library/旅行/2026/云南`), `/trash`, `/tokens`, `/settings`, `/status`, `/docs/api`, `/login`, `/setup`.
`/` is not a screen of its own: signed in it is replaced with `/recent`; signed out it goes to `/login`; before the
first administrator exists every console path goes to `/setup`. Signed out, any other known path goes to
`/login?next=<path>` and sign-in returns there (only same-origin known paths are accepted, else `/recent`).
`/docs/api` needs no session. Unknown in-app paths show a not-found page; a folder path that does not exist shows
文件夹不存在 inside the library. The server returns the SPA for these routes and for any `/library/...` path, so a
direct request for any other URL gets a plain 404. Sign-in and setup are plain centered cards. Overlays (modal,
drawer, sheet) animate in and out; all motion is disabled under `prefers-reduced-motion`.

## 最近

- Every active image, newest first, grouped by day (今天, 昨天, then the date). 60 px rows: thumbnail, name, folder
  and source, then dimensions, size and relative time, then 复制 (in the current link format) and the row ⋯ menu.
  Each day heading offers 复制全部链接 on hover.
- The header holds 复制为 URL / Markdown / HTML / BBCode, which sets the remembered link format (`picberth.linkFormat`),
  and 上传 as a split button whose menu picks the 位置 for uploads, drops and pastes started on 最近 (the root by
  default; a chosen folder is named next to the button). On phones 复制为 sits in a row under the header.
- The upload queue is listed at the top of 最近 (the floating upload panel is not shown there). Drops and pastes on
  screens other than 最近 and the library go to 最近.
- A row opens the same image detail as the library; `j` / `k` move through 最近, ← / → when the detail is open.
- States: loading rows, 加载失败 with 重试, and 还没有图片 / 拖入、粘贴或选择文件上传. Paging loads as you scroll.

## Library and trash

- Every screen has a 48 px panel header with the title and that screen's controls. Library: breadcrumb (from the
  root 图片 to the current folder; on phones only the current folder name), search (`/`), 筛选 (one popover:
  格式 / 上传时间 / 来源), a 显示 menu (视图 列表 / 网格, 排序; 选择 on phones), 上传 (hidden when the library is
  empty). Trash: 清空回收站. Tablet and phones: search collapses to an icon.
- Folders: from 1100 px a 224 px tree pane sits left of the list; below 1100 px the tree opens as a popover from
  the 文件夹 button next to the title. 新建文件夹, and per node a ⋯ menu with 新建子文件夹, 重命名, 移动到… and
  删除. Create and rename are inline in the tree row (Enter saves, Esc cancels). At most 3 levels: 新建子文件夹 on
  a depth-3 folder and move targets that would exceed 3 levels are disabled with 最多 3 层. 删除 is disabled with
  不为空 while the folder holds images or subfolders; an empty folder is deleted at once with 撤销. Folders are not
  draggable.
- Browsing shows the images directly in the current folder. Searching or filtering covers the current folder and
  everything below it (at the root: the whole library) and adds a 文件夹 column.
- Active filters show as removable chips under the header.
- Selecting images replaces the header controls in place with 已选 N 张, 全选 / 取消全选, 复制链接, 移动,
  移到回收站 (trash: 恢复, 永久删除) and ×. On phones the bulk bar floats above the tab bar.
- Moving images: bulk 移动, 移动到… in the row menu, the 文件夹 row in the image detail, and drag and drop onto a
  tree node or a breadcrumb ancestor (dragging a selected image drags the whole selection; a collapsed node
  expands after 0.7 s of hovering). Every move toasts with 撤销.
- List: rows with the thumbnail on the gutter (in line with the title); its checkbox appears over the thumbnail on
  hover or focus, and stays while selecting. Then name, dimensions, size and relative time; the trash and filtered
  results add 文件夹. Columns hide as the width narrows. Row actions are always visible, muted until the row is
  hovered: copy in the current link format, and a ⋯ menu (复制为 URL / Markdown / HTML / BBCode, 移动到…, 下载原图,
  移到回收站). On touch there are no checkboxes until a selection starts (选择 in the 显示 menu, or a long press). No footer count; paging loads
  automatically (加载中…, with a retry button on failure).
- Grid: justified rows of true-proportion images. The name has a line of its own under the image and is shortened in
  the middle so its extension stays visible; the trash adds 「N 天后清除」 and filtered results the folder on a
  second line.
- Image detail: ≥ 1440 px a docked column inside the panel; 721–1439 px a right drawer; phones a bottom sheet.
  Top bar ‹ › ×. Order: preview (checkerboard for transparent images, 打开原图), the filename as an editable
  title, meta line, link tabs URL / Markdown / HTML / BBCode with a copy field, then the property list: 文件夹,
  来源, 上传时间, 删除时间 (trash), ID, SHA-256. Footer 下载 / 移到回收站 (trash: 恢复 / 永久删除).
- Uploads go to the current folder. The upload queue is listed in place above the images (the floating panel is
  used only on other screens) and names the folder for uploads outside the root; the
  window-wide drop overlay names the destination folder. Purge confirm is the only confirmation dialog besides
  leaving unsaved settings; trash, move and folder delete offer 撤销 instead.
- Filters and sort live in the URL: `q`, `format` (repeatable), `date` (`today | 7d | 30d | older`), `source`
  (repeatable), `sort`. Dates are turned into `since` / `until` when the list is requested. The list is paged with
  `cursor`; the next page loads as you scroll.
- Counts in the filter popover come from `GET /api/v1/images/facets`, refreshed after uploads and mutations.
- The list / grid choice (`picberth.view`, default list), the link format (`picberth.linkFormat`) and the theme
  (`picberth.theme`) are remembered in `localStorage`.
- Keyboard: `/` search, `U` file picker, Esc clears the selection, then closes the image detail, ← / → move
  between images when the detail is open (docked or overlay), Enter opens and Space selects on a focused row or
  tile; Ctrl / ⌘ and Shift click select. Touch: long-press or 选择 starts selection on phones.
  Window-wide drag and drop and paste upload into the current folder.

## Status, settings, tokens and API reference

- Settings: centred column with sections 上传 (单文件上限, 格式, 重复检测) / 回收站 (保留时间) / 操作记录
  (保留时间, 7–3650 days) / 地址 (read-only). Number fields have a fixed width with the unit inside the field.
  Unsaved changes show a floating bar at the bottom of the screen (放弃 / 保存), above the tab bar on phones.
- 系统 pages are one centred column (760 px).
- Status: warning alerts (a broken image is listed as 图片已损坏; a missing or old snapshot is shown on the 快照 row
  as 近期没有快照 / 上次快照在 N 天前 instead); five figures in one card (图片, 占用空间, 回收站, 数据库, 磁盘剩余
  with a thin usage bar; no whole-disk breakdown); maintenance rows; run information (including 已运行); 操作记录
  with 保留 N 天, a filter (全部操作, 图片, 文件夹, 令牌, 设置, 系统, 登录; server-side `kind`) and pages of 10 with
  加载更多.
- Tokens: a list with a small header row; scopes 上传 and 管理; 吊销 turns into 确认吊销 for 4 s and revokes on
  the second press; the expiry is picked with the product-drawn select. The header links API 文档 (new tab; on phones only
  新建令牌, as an icon button, and API 文档 stays in the command palette).
- API reference (`/docs/api`): a page of its own with a contents column (a contents row on phones), copy buttons
  on code blocks and a link to `/api/v1/openapi.json`.
## Code layout

| Path | Content |
|---|---|
| `web/src/app.css` | tokens (light and dark through `data-theme` on `<html>`), primitives, shell, admin pages |
| `web/src/lib/` | `api.ts` (endpoints), `upload.svelte.ts` (queue), `library.svelte.ts` (list state and actions), `folders` (tree helpers, name validation), `facets`, `health`, `session`, `router`, `toast`, `theme`, `modal` (overlay stack, focus trap), formatting helpers |
| `web/src/components/` | shell, menus, selects and comboboxes (floating layer), overlays, upload dialog, shared controls |
| `web/src/components/library/` | header, breadcrumb, folder tree and menus, filters, table, grid, rows, detail panel, upload tray |
| `web/src/screens/` | route screens (library and trash, tokens, settings, status, API reference, login, setup, not found) |

Inter Variable is self-hosted; CJK text uses system fonts. No external assets are loaded. The CSP in
`internal/httpapi/server.go` applies (no inline scripts or event-handler strings).

## Automated E2E regression tests

The Playwright Chromium suite (`web/e2e/`, run via `mise run web:e2e`) runs against the compiled Go binary (`dist/picberth`) with an isolated temporary database and storage directory (`mkdtemp`), testing critical user interactions across desktop (1440×900) and mobile (390×844) viewports:

- **Upload tray cancellation and retry** (`web/e2e/upload.spec.ts`): Tests cancelling an in-flight upload from the inline upload tray (the XHR must abort and nothing is stored), and retrying after a transient 503 error with real backend persistence verification.
- **Batch trash and toast undo** (`web/e2e/trash-undo.spec.ts`): Tests multi-image selection (desktop hover checkboxes with the header bulk bar; mobile 选择 from the header ⋯ menu with the floating bulk toolbar), moving to trash, undoing via toast "撤销", and verifying restoration persistence across page reload.
- **Folders** (`web/e2e/folders.spec.ts`): inline create / rename / validation, the missing-folder state, the depth limit and delete rules with undo, folder moves, moving images by row menu and by drag and drop, and uploading into a folder then finding the image from the root with the 文件夹 column.
- **Console** (`web/e2e/console.spec.ts`): no tag or visibility controls, the 上传 / 管理 token scopes and the API 文档 link, `/docs/api`, the five editable settings, and the operation-log category filter with paging.
- **Settings form navigation guards** (`web/e2e/settings-guard.spec.ts`): Tests unsaved changes detection, the "离开设置？" guard, preserving drafts ("留下"), discarding in the save bar ("放弃") or in the guard ("放弃更改"), and saving with backend and reload verification.

`mise run web:check` checks both Svelte code and the E2E TypeScript files. Run browser tests on Linux/WSL;
the harness waits for the Go process to exit before deleting temporary data. Failure screenshots and traces
are written to the system temporary directory (`picberth-e2e-artifacts`, overridable with `E2E_OUTPUT_DIR`).
