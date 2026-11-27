# Revision 8 candidate: refined console with 最近

Status: **approved** (2026-11-27): the owner approved `designs/picberth/r8/console.html`, kept 最近 and 系统, and accepted the new `/recent` route. Production is not yet implemented; until it is, production follows revision 7
(`docs/ui/r7-revision.md`) plus zero-day trash retention.

The owner chose direction A (refined console) combined with direction B's 最近 view
(`docs/ui/redesign-exploration.md`). This candidate merges them and covers every state of the surfaces that change.

## Review

Serve with `python -m http.server 4321 --bind 127.0.0.1 --directory designs`, then open
`http://127.0.0.1:4321/picberth/r8/index.html`. The board lists 65 states. Each state is shown at 1440, 390, 1024 and
1280 px, in both themes. Every state can also be opened directly as `r8/console.html?…`; the board links each frame.

Files: `designs/picberth/r8/console.html` (the prototype), `base.css` (r7 tokens and primitives, D7 fix),
`shared.css`, `data.js` (synthetic fixtures), `index.html` (the review board, which is not product UI) and
`content-inventory.json`.

## What changes compared with revision 7

| Area | Revision 7 | Revision 8 candidate |
|---|---|---|
| Landing | `/` goes to `/library` | `/` goes to `/recent` (new route) |
| Sidebar from 1100 px | 图片 + tree, 回收站, 令牌, 设置, 状态, account | 搜索 Ctrl K, **最近**, 图片 + tree, 回收站, then **系统** and account at the bottom |
| Icon rail 721–1099 px | five items | 搜索, 最近, 图片, 回收站, 系统, account. The tree opens from the 文件夹 button next to the title (as in r7) |
| Phone tab bar | 图片 / 回收站 / 令牌 / 设置 / 状态 | **最近 / 图片 / 回收站 / 系统** (D9) |
| Admin pages | three separate pages | one 系统 entry with tabs 状态 / 令牌 / 设置 in the header. Routes `/status`, `/tokens` and `/settings` stay, each one a tab |
| Warning dot | on 状态 for any warning, including a stale snapshot | on 系统 only for real warnings (for example a broken image). A stale snapshot appears only on the 快照 row (D4) |
| Status disk | stacked bar scaled to the whole disk | five figures; 磁盘剩余 carries a thin usage bar. The stacked bar is removed (D3) |
| Grid tile | name and size share one line | name on its own line, truncated in the middle so the extension stays visible. Trash adds a separate 「N 天后清除」 line (D1, D2) |
| List row | checkbox column that fades in; actions on hover | the checkbox appears over the thumbnail on hover, so thumbnails line up with the title. The copy button is always visible, muted until hover (D8) |
| Phone selection | checkboxes always drawn | none until 选择 in the 显示 menu or a long press; a floating bulk bar then appears (D6) |
| System pages at 1440 | left-aligned, about 920 px | centred 760 px column (D5) |
| `.btn-danger` | white on red 9, 3.67:1 | white on red 11, 5.2:1 (D7) |
| Focus outlines | removed on `.input`, `.menu-item` and others | kept |

### 最近 (new)

- Uploads grouped by day (今天, 昨天, then dates), 60 px rows: thumbnail, name, folder and source, then size,
  dimensions and time, then 复制 and a ⋯ menu (the same row menu as the library).
- 复制为 URL / Markdown / HTML / BBCode in the header sets the link format. It is the same remembered format
  (`picberth.linkFormat`) that every copy button, the row menu and the highlighted row in the image detail use.
- 上传 is a split button. Its menu picks the 位置 (the root 图片 or any folder) for uploads started from 最近. A chosen
  folder shows next to the button, and the drop overlay names it.
- An upload batch appears at the top as 正在上传 with progress, 等待中, 已存在，使用原有链接, 超过单文件上限 and
  不支持的格式 (移除), plus 复制全部链接 for the finished items. Inside the library the same batch shows as the
  r7 inline tray in the current folder.
- Each day heading offers 复制全部链接 on hover.
- Clicking a row opens the image detail used by the library (docked from 1440 px, drawer from 721 to 1439 px, sheet
  on phones). ‹ › step through 最近. `j` / `k` move the selection.
- States: empty (还没有图片 / 拖入、粘贴或选择文件上传), loading, error (加载失败 / 重试), 加载更多.

### Unchanged from revision 7

These keep their r7 design and are not redrawn: sign-in and setup, the API reference, the viewer (查看大图), folder
create, rename and move inline in the tree, the folder ⋯ menu and depth limits, drag-and-drop moves, undo toasts
and zero-day retention. The prototype shows them only where the new shell hosts them: palette, account menu, filter
popover, 显示 menu, row menu, purge confirmation, token dialogs and the leave-settings dialog.

## Copy

The prototype uses revision 7 copy wherever it exists (`r8/content-inventory.json` shows the source of every string).
Two new words were approved by the owner on 2026-11-27:

- `最近`: the new navigation item, page title and palette entry.
- `系统`: the merged entry for 状态 / 令牌 / 设置.

The exploration flagged seven more words as drafts. In this candidate they are either approved revision 7 copy
(复制全部链接, 查看大图, 管理 as the token scope, 拖入、粘贴或选择文件上传 in place of 拖放或粘贴, 保留时间 under
回收站 and 操作记录) or no longer used (全部文件夹).

## Not in this candidate

The audit's product gaps stay out of scope until the owner decides on them: account and password, listing and
restoring snapshots, token onboarding after creation, custom link templates and a storage breakdown. 最近
covers the upload history gap using the existing image list API.

## Verification

- Rendered with Chrome (Playwright, local Chrome) at 1440 × 900, 1280 × 800, 1024 × 768 and 390 × 844, in both themes,
  across the board states. Issues found and fixed during review: the trash-only folder chips, the tree popover
  inheriting the rail's styles, the phone dialog overflowing, phone batch rows wrapping, the time showing twice in
  desktop sublines, and trash dates earlier than upload dates.
- `pfui.py check designs/picberth`: review, contract and sources pass. The 7 lint errors are all in revision 7 files
  (`styles.css`, `retention.html`); `pfui.py lint designs/picberth/r8` has 0 errors.
- Content: 69 rendered-DOM captures, 521 strings, every string classified. `content_audit.py check` passes after the owner
  approved 最近 and 系统.
- Not verified: screen readers, full keyboard order, Safari and Firefox, 200 % zoom. Upload, clipboard, drag and drop
  and every request are simulated.
