# PicBerth UI — design brief v2 「暗房」

PicBerth is where pictures are moored. The interface borrows from one specific, physical place: a
photographer's **darkroom** and the **contact sheet** printed there. The metaphor is never decoration for its
own sake; each borrowed object does a job:

| darkroom object | job in PicBerth |
|---|---|
| contact sheet: film strips with sprocket holes on a paper page | the library — every image is a frame on a strip, one strip per row |
| edge printing (frame numbers and stock names printed along the film edge) | where all machine metadata lives: frame number, file name, dimensions, flags |
| a print developing in the tray (pale ghost → full image) | upload progress — the preview *develops* as bytes arrive |
| light leak / fogged frame | a failed upload |
| red grease pencil (china marker) circling the keepers | selection |
| the loupe | holding the mouse on the full-size image magnifies it |
| the safelight | the red lamp that comes on while files are dragged over the window |
| 亮台 (light table) / 暗房 (darkroom) | light / dark theme |

Restraint rules: the metaphor appears in **structure, typography and motion**, never as textures, wood, paper
grain, photos of cameras or skeuomorphic chrome. Operational screens (tokens, settings, status) are plain,
precise tools that only share the typography, palette and a few edge-print details.

## 1. Hard bans

No purple/indigo/blue-violet, no decorative gradients (the safelight glow, light leak and ticket notches are the
only gradient uses, each functional), no glassmorphism / backdrop blur, no emoji, no pill buttons (only the
round mobile shutter button is circular), no card-with-left-accent-border, no Inter/Roboto/Arial, no stock
illustrations, no marketing copy or exclamation marks, no spinner where a progress value is known, no confirm
dialog for undoable actions (offer **撤销**), never `{@html}` with server strings.

## 2. Tokens (`web/src/app.css`)

Light theme = 亮台 (contact paper). Dark theme = 暗房. `:root[data-theme]` overrides the system preference;
the choice is stored in `localStorage['picberth.theme']` and applied in `main.ts` before first paint.

| token | light | dark | use |
|---|---|---|---|
| `--bg` | `#F2EFE8` | `#0E0D0B` | page (paper / darkroom) |
| `--surface` | `#FBFAF6` | `#171512` | raised panels, inputs, dialogs |
| `--surface-2` | `#E9E5DB` | `#211E1A` | hover, wells, table heads |
| `--line` | `#DDD7CB` | `#2A2621` | hairlines |
| `--line-strong` | `#BDB5A5` | `#3D3730` | control borders |
| `--ink` | `#16130F` | `#EEE8DC` | text, primary buttons |
| `--ink-2` | `#4A443B` | `#B8AF9F` | secondary text |
| `--ink-3` | `#7A7366` | `#8A8274` | meta, placeholders |
| `--accent` / `--mark` | `#CF3A24` | `#FF5B3D` | grease pencil: selection, active nav stroke, focus on paper, danger |
| `--accent-soft` | `#F6DED6` | `#3A1C15` | tinted backgrounds for the mark colour |
| `--ok` | `#2E7A47` | `#6CCB8A` | healthy |
| `--warn` | `#9A5B00` | `#F0B44C` | warnings |
| `--warn-soft` | `#F4E7CC` | `#33270F` | warning backgrounds |
| `--danger` | = `--mark` | = `--mark` | errors, destructive |
| `--film` | `#14110E` | `#1B1814` | film strip base (same darkness in both themes) |
| `--film-2` | `#231E19` | `#26211C` | the frame area inside the strip |
| `--edge` | `#EBA33F` | `#F0AE4E` | edge print, primary (frame numbers, live status) |
| `--edge-2` | `#A79C8B` | `#A79C8B` | edge print, secondary (file names, dimensions) |
| `--safelight` | `rgb(214 40 22)` | same | drag overlay lamp |

Typography (bundled latin subsets, CJK falls back to the system):

- `--font-ui`: `'Archivo Variable'`, then system CJK (`PingFang SC`, `Microsoft YaHei UI`, `Noto Sans CJK SC`…).
  Base 14 px / 1.55.
- `--font-display`: same family with `font-stretch: 118%` and weight 760–820 — the wordmark, page titles,
  hero figures. Class `.display`.
- `--font-mono`: `'Martian Mono Variable'` — ids, hashes, URLs, sizes. `font-stretch: 87.5%` for edge print
  (class `.edge`), 100 % elsewhere. Tabular figures always.

Scale: 11 (edge) / 12 / 13 / 14 / 16 / 20 / 28 / 44 (hero figures). Radii: 2 px panels and frames, 3 px controls,
4 px dialogs. Elevation: hairlines; dialogs/popovers one shadow `0 18px 48px -12px rgb(10 8 6 / .35)`.
Motion: `--ease: cubic-bezier(.2,0,0,1)`, 120–220 ms; signature motions listed in §6; all off under
`prefers-reduced-motion`.

Focus: `:focus-visible` = 2 px solid `--ink` outline, 2 px offset. On film (dark strips) the outline is `--edge`.
Text inputs: border becomes `--ink`, plus `0 0 0 3px` ring of `--ink` at 12 %.

## 3. Components

- **Button** variants: `primary` (ink fill, bg-coloured text), `secondary` (transparent, 1 px `--line-strong`),
  `ghost`/`text`, `danger` (red text + red hairline, fills red on hover), `inverse` (for dark bars/film).
  Heights 36 / 30 (sm); 44 on coarse pointers. Weight 600, 13.5 px.
- **TextField**: 38 px, `--surface`, 1 px `--line-strong`, radius 3. `variant="line"`: no box, only a bottom
  rule, 16 px text — used for the library search ("sleeve label").
- **Switch**: square-cornered rocker 34×20, knob 14×14 radius 1; on = `--ink` track.
- **Checkbox**: 16 px square, radius 2; checked = ink fill with paper check.
- **Segmented**: hairline-bordered group, selected segment ink fill.
- **Stamp** (replaces chips for scopes/tags): 11 px mono uppercase-ish label with 1 px border, radius 2.
- **Toasts**: bottom-centre, `--film` slip with `--bg`-coloured text, action button in `--edge`.
- **Dialog**: `--surface`, radius 4, header title in `.display` 20 px.
- **EdgeText**: `.edge` mono 10.5–11 px, letter-spacing .04em, `--edge`/`--edge-2`.

## 4. Shell

Desktop (> 760 px): sticky header 56 px on `--bg` with bottom hairline.
Left: mark + **PicBerth** wordmark (`.display`, 17 px). Then nav **底片 · 回收站 · 令牌 · 设置 · 状态**
(14 px, `--ink-2`; current = `--ink` with a hand-drawn red grease-pencil stroke under it, drawn once on change).
Right: status lamp (8 px dot: ok/warn/danger from `/status` warnings, links to /status, `title` explains),
theme switch (lamp icon, `title` "切换到暗房"/"切换到亮台"), user name + 退出, and the primary
**冲洗照片** button (upload picker). Keyboard: `U` opens the picker, `/` focuses search.

Mobile (≤ 760 px): header 52 px = wordmark + lamp + theme + logout icon. A bottom tab bar
(64 px + safe-area) with 底片 · 回收站 · **shutter** · 令牌 · 设置; the shutter is a 56 px round button
(ink ring, inner disc) that opens the picker. Status is reached from the lamp.

Routes: `/` and `/library` both show the sheet (nav links to `/`); `/trash`, `/tokens`, `/settings`, `/status`,
`/login`, `/setup`.

Global input: files dragged anywhere → **safelight overlay**; drop or paste anywhere → files queue and the app
navigates to `/` where the developing roll shows them.

## 5. Screens

### 5.1 底片 (sheet) `/`

Header ("sleeve label"): `h1.display` **底片**; to its right on the same baseline a hero figure
(`.display` 28 px) frame count + ink-3 "帧 · 2.4 GB". Below: the line-variant search (placeholder
"按文件名、ID 或 SHA-256 查找", `/` hint), then a filter row: format stamps (JPEG PNG GIF WebP AVIF, single
toggle), visibility segmented (全部 公开 私有), 含位置 toggle, sort select (最新 / 最早 / 最大 / 最小).
Filters live in the URL.

Rolls: when sorted by time, frames are grouped into **rolls by local day**. Roll header: date ("10月21日 周二",
"今天", "昨天") in 15 px/650, right side edge text "6 帧". Size sorts show one roll titled by the sort.

Strips: one strip per row, `cols = floor((w + 6) / (minFrame + 6))`, minFrame 208 px (desktop) / 148 px
(≤ 560 px, min 2 cols). A strip is `--film` with 26 px bands top and bottom: sprocket holes (6×8 px, radius 1.5,
every 18 px, punched in `--bg`) on the outer half, edge print on the inner half. Frames are 3:2, `--film-2`,
image `object-fit: contain`; 6 px film between frames. The last strip of a roll is only as long as its frames.

Edge print per frame — top band: frame number in `--edge` (`12▸`) and flags at the right (lock = private,
pin = GPS, `GIF`/`ANIM` for animated, `!` red for broken, `NEW` amber for images uploaded this session).
Bottom band: file name (middle-truncated) in `--edge-2`, dimensions at the right.

Frame interactions: click opens the lightbox (with a view transition from the frame); Ctrl/⌘-click toggles
selection; Shift-click selects a range; Space toggles; Enter opens; arrow keys move between frames
(up/down by one strip); `C` copies the focused frame's link. Hover shows two buttons inside the frame,
bottom-right: copy link, select (circle). Touch: long-press (450 ms) selects; while a selection exists, a tap
toggles.

Selection: a **red grease-pencil rectangle** is drawn around the frame (one of four hand-drawn path variants,
chosen from the id; stroke-dashoffset animation 260 ms). Selection bar = fixed bottom-centre `--film` slip:
big count (`.display`) + "帧已圈选" + actions (复制链接 · 设为公开 · 设为私有 · 移到回收站 | trash: 恢复 · 永久删除)
+ close. Esc clears.

Developing roll (uploads of this session, above the rolls, only on the sheet): header **冲洗中** + live summary
("2 张显影中 · 3 张已存档 · 1 张失败"), link-format segmented (URL / Markdown / HTML / BBCode, persisted),
**复制全部链接**, **收起** (clears finished). Each upload is a frame in the same strip style:

- queued: preview as a pale ghost (`--dev: 0`), edge "等待".
- uploading: the preview develops — `filter: grayscale(1-d) brightness(1.9-.9d) contrast(.25+.75d)` with `--dev`
  = progress (registered `@property`, 300 ms transition); edge "显影 42%"; a 2 px `--edge` line grows along the
  bottom band.
- processing: fully developed, edge "定影…" pulsing.
- done: edge "已存档" in `--ok`-tinted amber → file name; a persistent **复制链接** button on the frame
  (label flips to 已复制 1.5 s). Duplicate: edge "已存在 · 原链接". GPS: pin flag + `title` warning.
- error: **light leak** — preview desaturated, an orange-red fog from one corner, edge text in red with the
  reason; buttons 重试 (only if retryable) and 移除.

Empty sheet: centred block — `.display` 28 px "暗房已就绪", one line "把照片拖进窗口、粘贴截图，或者" +
primary **选择照片**; below it an empty strip of 6 frames with dashed inner outlines and frame numbers 1–6;
clicking an empty frame also opens the picker. Filtered empty: "没有符合条件的底片" + **清除筛选**.

### 5.2 Lightbox (detail)

A modal `<dialog>` that is always dark (viewing a print under the lamp), regardless of theme. Left: the image as
a **print** — 6 px paper (`#F4F1EA`) border, centred, max size; checkerboard under transparency. Holding the
primary mouse button on it shows a 200 px round **loupe** with the raw image at 1:1 (or 2.5× when smaller);
hint under the print "按住查看细节". Top bar: 关闭 (Esc), frame position `07 / 128` in edge text, previous / next
(← →, also swipe on touch). Right panel 380 px ("sleeve"): file name (16 px/600, wraps), stamps (公开/私有, GPS,
动图, 损坏); **链接**: format tabs + one large mono field + **复制** primary; private note; **公开访问** switch;
GPS warning; **信息** grid (尺寸, 大小, 格式, 帧数, 上传时间, 来源, ID, SHA-256 with copy); actions at the bottom:
下载原图 · 新标签页打开 · 移到回收站 (trash: 恢复 / 永久删除). ≤ 760 px: print on top, sleeve scrolls below.

### 5.3 回收站 `/trash`

Same sheet; frames rendered desaturated (`grayscale(.85)`, 70 % opacity, full on hover); bottom edge shows
"17 天后清除"; header meta "保留 30 天". Lightbox actions 恢复 (primary) / 永久删除 (danger, confirm dialog:
"永久删除后无法恢复。已被浏览器或 CDN 缓存的副本可能仍可访问一段时间。").

### 5.4 令牌 `/tokens`

Header **令牌** + ink-3 "给 PicGo、ShareX 和脚本使用的上传凭证" + primary **新建令牌**.
List of rows (not cards): name (15 px/600) + mono prefix `pb_7KQ2M…`; scope **stamps**
(上传 / 只读 / 管理); 创建 · 最近使用 · 过期 in ink-3 13 px; **吊销** danger-text with confirm. Revoked: ink-3,
strikethrough, stamp 已吊销, no actions. Empty: "还没有令牌" + **新建令牌**.
Create dialog unchanged in behaviour (名称, 权限 checkboxes with explanations, 有效期). The reveal step shows the
secret as a **ticket**: film-dark panel with notched sides (radial-gradient notches), the secret in mono 15 px
`--edge`, **复制**; warn "关闭后将无法再次查看此令牌"; then 接入示例 tabs (curl / PicGo / ShareX with .sxcu
download) and **完成**.

### 5.5 设置 `/settings`

Two columns ≥ 900 px: a sticky left index of sections (上传 · 回收站 · 频率限制 · 会话 · 地址) with the
current one marked by the grease stroke; right: sections with `.display` 20 px headings and form rows
(label + help left 260 px, control right). Same fields and validation as before. Dirty → sticky bottom slip
(`--film`): "3 项更改未保存" + 放弃 + 保存 (primary-inverse). Leave guard unchanged.

### 5.6 状态 `/status`

Top: a one-line verdict in `.display` 28 px — "一切正常" (ok lamp) or "需要注意 · 2 项" (warn lamp) — and
the warnings as plain rows (lamp + message + remedy). Figures: a 3×2 grid of hero numbers (`.display` 44 px):
底片 · 占用空间 · 回收站 · 含位置 · 数据库 · 磁盘剩余. Disk: a 10 px **exposure scale** bar (tick marks every
10 %, used = ink, free = surface-2, watermark = red tick with label). **维护**: 快速检查 / 完整检查 / 创建快照
rows (description + action + inline result). **运行信息**: mono definition list. **操作记录**: a log in mono
time column + Chinese action labels, 20 rows + 加载更多.

### 5.7 登录 `/login` · 初始化 `/setup`

Split screen. Left 44 % (top band on mobile): `--film` panel — the wordmark set huge (`.display`, ~72 px,
`--bg` colour), the line "图片的泊位" below in `--edge-2`, and one horizontal film strip running across its
bottom edge (sprocket holes + edge print "PICBERTH 400 ▸ 1 ▸ 1A ▸ 2 …"), with empty frames. Right: form on
`--bg`, max 360 px, title `.display` 28 px (**登录** / **初始化**). On successful login the screen does a short
"lights on" transition (safelight red → paper, 450 ms) before the app appears.
Setup: three numbered steps as plain text (1 在服务器日志中找到初始化代码 2 创建管理员 3 开始使用); the code
input is a large mono field (20 px, letter-spacing .2em) that upper-cases and inserts dashes as you type
(`XXXX-XXXX-XXXX`); 用户名 (default admin), 密码, 确认密码 (≥ 10), **创建管理员**.

## 6. Signature motion (all disabled under reduced motion)

1. Developing: `--dev` 0→1 with progress (§5.1).
2. Grease pencil: selection rectangle and nav underline draw via stroke-dashoffset, 260 ms.
3. Lightbox: view transition from frame to print (`view-transition-name: pb-print`), 280 ms.
4. Safelight: overlay fades in 120 ms with a two-step flicker of the lamp.
5. Lights on: after login, 450 ms.

## 7. Behaviour details (unchanged contracts)

- Sizes 1024-based (`183 KB`, `4.2 MB`); times relative < 7 days ("3 分钟前", "昨天") else `2026-10-21 14:20`,
  absolute time in `title`.
- Optimistic UI only for visibility toggles; trash/restore offer **撤销**; purge is the only confirm.
- Fetch wrapper: `X-PicBerth-Client: web`, problem+json, 401 → `/login?next=`, version-change toast.
- Live region announces upload completions/errors. Every icon button has `aria-label` + `title`.
  Dialogs use `<dialog>.showModal()`. Tab order follows visual order.
- Browser baseline: Chrome/Edge 111+, Firefox 114+, Safari 16.4+. View transitions and `@property` are
  progressive enhancements.
