# Folders, routes, operation log retention and API reference — prototype revision

Status: **approved** on 2026-11-09. The owner approved the prototype in `designs/picberth/` together
with the decisions below. `docs/ui/capabilities.md` still describes production as it is; this file
describes what the prototype proposes. Design rules: `DESIGN.md`. Content boundary:
`docs/product/constraints.md`.

## Review URLs

Serve with `python -m http.server 4321 --directory designs`.

| What | URL |
|---|---|
| Library root, tree pane, detail docked (1440 px) | `http://localhost:4321/picberth/index.html#/library` |
| A folder at depth 3 (create blocked, delete blocked) | `…/index.html#/library/旅行/2026/云南` |
| Empty folder (delete allowed), 旅行 → 2025 | `…/index.html#/library/旅行/2025` |
| Missing folder | `…/index.html#/library/不存在` |
| `/` redirect | `…/index.html#/` |
| Trash with 文件夹 column | `…/index.html#/trash` |
| Tokens with API 文档 link | `…/index.html#/tokens` |
| Settings → 操作记录 保留时间 | `…/index.html#/settings` |
| Status → 操作记录 filter and 加载更多 | `…/index.html#/status` |
| API reference | `…/api-docs.html` |
| Folder states | `…/index.html?folders=none`, `?folders=loading`, `?folders=error`; library data `?data=empty|loading|error` |
| Sign-in with return address | `…/auth.html?next=/library/旅行` |

The prototype keeps hash routing (`#/library/旅行/2026`) because it is served statically. The path after `#` is the
production path. Resize below 1100 px for the tree popover (tablet) and below 721 px for the phone layout.

## Route map

Production today serves exact paths only (`internal/httpapi`: `/`, `/login`, `/setup`, `/library`, `/trash`,
`/tokens`, `/settings`, `/status`; `/library/extra` is a 404). Existing paths are already semantic and stay.

| Path | Screen | Change |
|---|---|---|
| `/` | redirect | signed out → `/login?next=…`; signed in → replace with `/library`; first run (no admin) → `/setup` |
| `/setup` | setup | unchanged; first run only |
| `/login` | sign-in | unchanged; after sign-in goes to `next`, else `/library` |
| `/library` | library, root | unchanged |
| `/library/<folder path>` | library, one folder | **new**: names joined by `/`, each segment URL-encoded, e.g. `/library/旅行/2026/云南` |
| `/trash` | trash | unchanged; flat across folders |
| `/tokens` | tokens | unchanged; adds an `API 文档` link |
| `/settings` | settings | unchanged; adds 操作记录 section |
| `/status` | status | unchanged; operation log gets filter and paging |
| `/docs/api` | API reference | **new**: plain server-rendered page, no session needed, same-origin CSS and JS only |
| `/api/v1/openapi.json` | OpenAPI document | **new** (machine-readable); linked from the reference page |
| anything else | 404 / in-app not-found | unchanged |

Server consequence: the SPA entry must be served for the prefix `/library/` (today `/library/extra` is a 404 on
purpose). Folder paths that do not exist show the in-app 文件夹不存在 state, not a server 404.

Filters and sort stay in the query (`q`, `format`, `date`, `source`, `sort`); the `tag` parameter goes away.

## Folder model

- Depth at most 3 (`/A/B/C`). The fourth level cannot be created: 新建子文件夹 on a depth-3 folder is disabled and
  states 最多 3 层; moving a folder shows targets that would exceed 3 levels (counting its own subtree) disabled with
  the same words.
- An image is in exactly one folder or in the root (图片). Folders are organisation only: moving images or folders
  never changes an image link.
- Folder names: 1 to 64 characters, no `/` or `\`, unique within the same parent. Errors appear inline in the tree
  row: 请输入名称, 名称最多 64 个字符, 名称不能包含 / 或 \, 已有同名文件夹.

## Decisions taken where the brief left a choice

1. **What a folder shows.** Browsing shows the images directly in the current folder. Searching or filtering covers
   the current folder and everything below it (at the root: the whole library) and adds a 文件夹 column, so results
   can be located. Header count: `n` when browsing, `n / total` when filtering. Root count is the root's own images.
2. **Delete a non-empty folder: blocked.** 删除 in the folder menu is disabled and says 不为空 (images or subfolders
   inside). An empty folder is deleted at once with 撤销; no dialog. Trashed images that were in a deleted folder
   return to the root when restored.
3. **Moving images:** bulk 移动 button (header bar; phone float bar, icon only), 移动到… in the row menu, the
   文件夹 row in the image detail, and drag and drop onto a tree node or a breadcrumb ancestor. Dragging a selected
   image drags the whole selection (ghost shows 3 张图片). Every move toasts with 撤销.
4. **Moving folders:** 移动到… in the folder menu (list of valid targets). A same-name conflict in the target is
   refused with a toast. Folders are not draggable (keeps the tree stable; revisit if wanted).
5. **Navigation:** from 1100 px a 224 px tree pane sits inside the panel, left of the list, with the breadcrumb in
   the header. Below 1100 px the pane becomes a popover from a folder button next to the title; on phones the
   breadcrumb ancestors collapse to the current folder name. Create, rename, move and delete work the same in the
   popover. Hovering a collapsed node during a drag expands it after 0.7 s.
6. **Create and rename are inline** in the tree row (Enter saves, Esc cancels); no dialogs.
7. **Upload goes to the current folder.** The tray shows the folder for uploads that are not in the root. The
   window-wide drop overlay says 松开以上传到 <path> inside a folder.
8. **Docked image detail now starts at 1440 px** (was 1280 px) so the tree pane, list and 380 px column fit.
   721 to 1439 px uses the existing right drawer.
9. **Tag feature removed completely:** chips, filter section, editor, column, detail row, search by tag, copy.
   The detail's first property is 文件夹.
10. **Operation log is read-only.** No per-entry delete. 设置 → 操作记录 → 保留时间 (days, 7 to 3650, default 180).
    状态 → 操作记录 shows 保留 N 天, a filter (全部操作, 图片, 文件夹, 令牌, 设置, 系统, 登录) and pages of 10 with
    加载更多.
11. **API reference** is its own page (`/docs/api`), opened in a new tab from the 令牌 header. Fixtures use
    `https://pics.example.com` and `pb_…`. Folder endpoints identify a folder by its path (`旅行/2026`). Token
    scopes are 上传 and 管理 only; the upload rate limit is a fixed 60 per minute.

## Decisions (approved)

1. Browsing a folder shows only its direct images; search and filter include all subfolders.
2. API upload with a non-existent `folder` returns 404; folders are never auto-created.
3. The docked image detail starts at 1440 px.
4. Folders are moved only through the folder menu (no folder drag and drop); images can be dragged onto tree nodes.
5. The server returns the SPA for any `/library/...` path.
6. The backend visibility (public/private) concept is removed entirely.
7. Settings are reduced to: max upload size, allowed formats, dedupe, trash retention, and operation-log retention.
   Upload rate limit (60/min), session idle (7 days) and absolute (30 days) lifetimes, and max pixels (50 MP) become
   fixed constants. The 会话 section and the 像素上限 and 频率上限 rows leave the settings prototype; the 地址 display stays.
8. Token scopes are reduced to `upload` and `manage` (the `read` / 只读 scope is removed).
9. Formats stay JPEG, PNG, GIF, WebP and AVIF.

## Surfaces and states covered

| Surface | States prototyped |
|---|---|
| library root / folder | ready, empty folder (这里没有图片), no images at all (还没有图片), filtered, filtered empty, loading, error, folder missing, tree loading / error / none |
| folder tree | expanded, current, hover, drag over node, drag over breadcrumb, inline create, inline create with duplicate name, inline rename, menu, menu at depth 3, delete disabled (不为空), delete empty with 撤销, move menu with depth-blocked targets |
| image move | bulk menu, row menu, detail 文件夹 menu, drag and drop, 撤销 |
| trash | 文件夹 column, detail shows the folder |
| tablet / phone | tree popover with nested folder menu, phone breadcrumb, phone float bar with 移动 |
| upload | into a folder, tray shows folder, drop overlay with destination |
| settings | 操作记录 保留时间, validation |
| status | operation log filter, empty filter result, 加载更多 with loading |
| tokens | API 文档 link beside the single primary 新建令牌 |
| api-docs | desktop with contents column, phone with contents row, copy buttons |
| auth | return to `next` after sign-in |

## Backend this revision needs (not in the API today)

Folders table and `folderId` on images; `GET/POST /api/v1/folders`, `PATCH/DELETE /api/v1/folders/{id}`;
`folder` on upload, on `PATCH /api/v1/images/{id}` and `list`; batch action `move`; errors `folder-exists`,
`folder-not-empty`, `folder-depth`; upload with an unknown folder returns `not-found`; removal of tags
(`tags`, `tag`, facets `tags`, tag audit actions); token scopes reduced to `upload` and `manage`; removal of the
visibility concept; fixed constants for upload rate (60/min), session idle (7 days) and absolute (30 days) lifetime,
and max pixels (50 MP); `auditRetentionDays` setting plus a purge job; `kind` filter on `GET /api/v1/audit`;
`/api/v1/openapi.json`; `/docs/api`; SPA served for `/library/*`. The API reference page documents the folder parts
as the approved contract.

## DESIGN.md check

| Rule | Result |
|---|---|
| 1 Copy: no `·`, no explanatory text, one name per thing | No `·` in any prototype file. New copy is labels, states and errors. 文件夹 / 移动 / 移到回收站 are distinct verbs. |
| 2 Hierarchy: one primary action, no eyebrows | Library 上传, tokens 新建令牌, settings 保存 stay the only lime buttons. 文件夹 pane title is a category the page title does not carry. |
| 3 Colour: Radix tokens only | No new colour literals; new rules use `--sel`, `--hover`, `--muted`, `--faint`, `--hairline`, `--ring`, `--danger-fill`. |
| 4 Type: three sizes | Computed sizes on library, tokens, settings, status and docs are only 12, 13 and 16 px; weights 400, 510, 590. |
| 5 Alignment | 4 px grid; tree indent 16 px; counts right aligned with tabular numbers. |
| 6 Radius | Tree rows and crumbs 6; menu items 4; inline input 6; code blocks 8; copy button 4 inside the 8 block. The drag ghost is a pill because it is a counter. |
| 7 Lines | 0.5 px hairlines; the inline-edit field uses the same 1 px focus line as `.input:focus`. No shadows except floating layers. |
| 8 Motion | None added. Drag states and hover are instant. |
| 9 Decisions | Delete and move are undone with 撤销, not confirmed. Depth and non-empty limits are shown as disabled items with the reason. No new dialogs. |

Content firewall: the DOM audit (`content_audit.py check`) passes for 569 strings across 65 captured states;
folders-revision strings carry the note `folders revision, approved 2026-11-09`.

## Verification actually run

Rendered in Ego browser at 1440, 900 and 390 px: routes and redirect, tree navigation, depth limit, duplicate name,
non-empty delete block, empty delete with undo, rename and move keeping the address in step, drag and drop onto a
node, bulk and row move with undo, scoped search with 文件夹 column, trash column, missing folder, folder loading /
error / none, upload into a folder, settings validation, audit filter and paging, tokens link, API reference
layout and horizontal overflow, return to `next`. After the approved decisions: settings shows only the five
editable values, the token scope options are 上传 and 管理, and the API reference shows the two scopes, the fixed
60/min limit and the unknown-folder 404. No script errors were raised.

Not verified: screenshots (the browser could not capture images in this environment, so visual polish, spacing and
the real drag cursor were not inspected by eye), Safari / Firefox, keyboard-only walk through the tree, reduced
motion.

## Open questions

None. The questions left by the prototype are settled by the approved decisions above.
