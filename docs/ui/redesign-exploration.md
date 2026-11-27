# Redesign exploration — audit and three directions

Status: **decided** (2026-11-27). The owner chose A combined with B's 最近 view. The merged candidate is
`docs/ui/r8-candidate.md` (`designs/picberth/r8/`). The three explorations are marked `changes-requested` and kept
for reference. Nothing here is approved; production is unchanged. Approved truth is still revision 7
(`docs/ui/r7-revision.md`) plus zero-day trash retention.

## Review URLs

Serve with `python -m http.server 4321 --bind 127.0.0.1 --directory designs`.

| What | URL |
|---|---|
| Board: every direction at 1440 / 390 / 1024, both themes, state chips | `http://127.0.0.1:4321/picberth/explore/index.html` |
| A 精修控制台 | `…/explore/a.html` (`?state=uploading|empty|loading`, `?view=system|trash`, `?detail=2`) |
| B 上传优先 | `…/explore/b.html` (`?state=uploading|empty`, `?tab=library|trash`) |
| C 三栏文件管理 | `…/explore/c.html` (`?state=one|multi`, `?folder=shots`; phone `?m=folders`, `?detail=2`) |

All pages accept `?theme=light|dark`. Fixtures are synthetic (`explore/data.js`, `designs/picberth/assets`).

## 1. Audit of the running product

Method: `dist/picberth.exe` (built 2026-11-19, after the last web change) on an empty temporary data directory,
seeded through the public API with 26 generated images, 6 folders (3 levels), 3 trashed images and 2 tokens.
Captured with Chrome at 1440×900 (dark), 1024×768 and 390×844 (light). Screens: library grid and list, folder,
detail, palette, trash, tokens, settings, status, API reference, sign-in.

### Defects (fix regardless of direction)

| # | Where | Finding | Standard missed |
|---|---|---|---|
| D1 | Trash grid, 390 px | `30 天后清除` is drawn over the file name (`30 天后清除photo…`) | overlap / resilience |
| D2 | Library grid, all widths | name and size share one line; narrow tiles show `ph…` / `photo-…` and lose the extension, so similar names cannot be told apart | task clarity |
| D3 | Status | the disk bar is scaled to the whole disk: PicBerth's own data (KBs to MBs) is an invisible sliver next to 其他 | data graphic answers the wrong question |
| D4 | Shell | the amber dot on 状态 is shown on a fresh instance only because no snapshot exists yet; it stays on every screen | alarm fatigue |
| D5 | Tokens / settings / status at 1440 | content stops at ~920 px and is left-aligned; a third of the panel is empty. `docs/ui.md` says settings is a centred column | drift from approved design |
| D6 | Phone, all grids | selection checkboxes are permanently drawn on every tile | noise on touch |
| D7 | `styles.css` and production tokens | `.btn-danger` white on Radix red 9 is 3.67:1 (needs 4.5:1 for 13 px text). `pfui.py lint` also flags removed focus outlines on `.input`, `.menu-item`, `.jt`, `.cmdk-input input` and unnamed controls in `retention.html` | WCAG 2.2 AA |
| D8 | List view at 1440 | four columns end at ~1300 px; row actions are invisible until hover; folder column only appears while searching | density / discoverability |
| D9 | Phone | three of five tab-bar slots go to rarely used admin pages (令牌, 设置, 状态) | thumb reach for the frequent job |

### Product gaps (need backend work or a scope decision)

Already listed as open in `docs/product/constraints.md`: batch rename, size-range filter, notes, statistics.
Found missing in this audit and not yet discussed:

1. **Account**: no way to change the administrator password or name from the console (the API has no endpoint);
   no list of active sessions and no "sign out everywhere".
2. **Snapshots**: snapshots can be created, but not listed, downloaded or restored from the console; restoring is
   documentation only.
3. **Upload history**: once the upload tray is closed, "what did I just upload" exists only as the library sort.
4. **Token onboarding**: the integration snippets appear once at creation; afterwards there is no place that shows
   how to configure PicGo / ShareX / curl for an existing token.
5. **Link formats**: the four formats are fixed; no custom template (for example a Markdown variant with a width).
6. **Storage view**: no breakdown by folder or format beyond the filter facets.

## 2. Directions

All three keep every capability in `docs/ui/capabilities.md`, the Radix palette, three type sizes, 0.5 px lines and
the motion rules. They differ in navigation model, home screen, pane relationships and phone model.

| | A 精修控制台 | B 上传优先 | C 三栏文件管理 |
|---|---|---|---|
| Home | current folder as a justified grid | 最近: uploads grouped by day | folder table with an inspector |
| Navigation | sidebar with tree; 令牌 / 设置 / 状态 merged into one 系统 page | top bar: 最近 / 图片 / 回收站; system behind an icon | toolbar with path bar; tree pane; one 管理 entry |
| Upload → copy | inline tray at the top of the grid; 复制链接 per item, 复制全部链接 | one copy format chosen once (复制为), then one 复制 per row; upload bar always visible with target folder | upload button; copy from the inspector |
| Detail | docked column / drawer / sheet (as today) | reader pane beside the stream | inspector always visible: folder, one image or a multi-selection summary |
| Density | medium, image-led | medium, row-led (60 px) | high, 32 px rows, sortable column headers, status bar |
| Phone | 3 tabs 图片 / 回收站 / 系统 | 最近 / centre 上传 / 图片 | drill-down: folders → images → detail |
| Fixes D1–D9 | D1 D2 D3 D4 D5 D6 D9 | D2 D9 (stream replaces the grid) | D2 D8 D9 |

### Trade-offs

- **A** is the smallest change and the lowest risk; it re-uses most production components. It keeps the grid as
  the home, so finding a recent upload still means scanning thumbnails. Merging the admin pages makes the 系统 page
  long. A missing snapshot is shown on the 系统 page only, not as a dot in the navigation; the disk bar compares
  PicBerth's own images and database, and the free disk space is a figure.
- **B** shortens the most frequent job (upload, then paste a link elsewhere) to one press per image and makes
  "what did I just upload" a first-class screen. Browsing by folder becomes secondary and the folder tree leaves the
  permanent navigation. It replaces the approved sidebar with a top bar.
- **C** is the fastest for organising and batch work (multi-select summary, sortable columns, status bar) and maps
  well to keyboard use. It is the least visual and the phone drill-down needs more taps to reach an image.

A hybrid is possible: A's shell with B's 最近 view as the first item in the sidebar, or C's inspector states inside A.

### What would change in durable rules

| Rule | A | B | C |
|---|---|---|---|
| `docs/ui.md` shell (sidebar, five nav items, five phone tabs) | merge three admin items into 系统; 3 phone tabs | top bar instead of sidebar ("There is no top bar"); 3-slot phone bar | toolbar + path bar; 管理 entry |
| `DESIGN.md` | none | none | none |
| Lime usage | unchanged | phone centre 上传 button is the view's primary action | unchanged |

### Draft wording (not approved)

`系统`, `最近`, `管理`, `全部文件夹`, `复制全部链接`, `拖放或粘贴`, `回收站保留`, `操作记录保留`, `查看大图`.
They are marked `review` in `designs/picberth/explore/content-inventory.json`; the content check fails on them until
the owner accepts or replaces them.

## 3. Critique scores (rendered, 0–2)

| Dimension | Production r7 | A | B | C |
|---|---|---|---|---|
| Task clarity | 2 | 2 | 2 | 1 |
| Hierarchy | 2 | 2 | 2 | 1 |
| Layout and rhythm | 1 (D5) | 2 | 2 | 2 |
| Typography | 2 | 2 | 2 | 2 |
| Color and contrast | 1 (D7) | 1 (inherits D7 token) | 1 | 1 |
| States and feedback | 2 | 1 (subset prototyped) | 1 | 1 |
| Interaction and access | 1 (D7) | 1 | 1 | 1 |
| Content | 2 | 2 (draft words pending) | 2 | 2 |
| Platform fit | 1 (D9) | 2 | 2 | 1 |
| Craft and consistency | 2 | 2 | 2 | 2 |
| Resilience | 1 (D1, D2) | 2 | 2 | 2 |

The directions prototype the main screens and the states listed in `ui-contract.json` (`explore-a|b|c`), not the
full state matrix. Dialogs, menus, filters, palette, viewer, sign-in and token creation are unchanged from r7 and
were not re-drawn.

## 4. Not verified

- Screen readers and full keyboard order (only the visible focus ring and arrow / Esc handling were exercised).
- Real upload, copy-to-clipboard and drag and drop: the prototypes simulate them with a toast.
- Safari and Firefox; 200 % zoom.
- `pfui.py gate design-scope` cannot pass in the current working tree because of unrelated uncommitted deletions
  under `.agents/skills/` that predate this work.
