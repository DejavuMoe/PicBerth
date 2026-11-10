# Revision 7 — one sidebar, one list, viewer, command palette

Status: **approved** on 2026-11-10 by the owner, including the grid as the default view. `docs/ui/capabilities.md` still describes
production as it is; this file describes what the prototype in `designs/picberth/` proposes. The previous
approved prototype is commit `c3bb4d0`.

Revision 6 (same day, never committed) changed the neutral colour, spread lime across selection and progress,
added type sizes, radii, cards and motion. The owner reviewed it and asked for the opposite: order, optical
alignment, concentric radii, as little motion and decoration as possible. Revision 7 keeps the structural ideas of
revision 6 and puts colour, type, radius, line, shadow and motion back under `DESIGN.md`.

## Review URLs

Serve with `python -m http.server 4321 --directory designs`.

| What | URL |
|---|---|
| Library root: sidebar tree, folders first, grid | `http://localhost:4321/picberth/index.html#/library` |
| Folder with a subfolder | `…/index.html#/library/旅行/2026` |
| Empty folder | `…/index.html#/library/旅行/2025` |
| Trash | `…/index.html#/trash` |
| Tokens, settings, status | `…/index.html#/tokens`, `#/settings`, `#/status` |
| Page that does not exist | `…/index.html#/missing` |
| Sign-in, setup | `…/auth.html`, `…/auth.html?screen=setup` |
| API reference | `…/api-docs.html` |
| States | `?data=empty|loading|error`, `?folders=none|loading|error`, `?theme=light|dark` |

In the library: click an image for the detail, click its preview (or double-click an image) for the viewer,
`Ctrl/⌘ K` for the command palette. The 原型 control (bottom left of the panel) starts sample uploads, shows the
drop overlay and opens the palette.

## What changes against the approved prototype

| Area | Approved | Proposed |
|---|---|---|
| Navigation | sidebar plus a separate folder pane inside the panel | one sidebar: the folder tree is the top of the navigation, 回收站 and the three admin pages below it. Rail and phone layouts keep the tree popover |
| Folders in the content | not shown | in the grid view the subfolders of the current folder are a row of plain folder buttons above the images; they accept dropped images. The list view shows images only. No previews, no extra requests |
| Grid | fixed 4:3 tiles, cropped | rows of equal height that fill the width; every image keeps its proportions |
| Default view | list | grid (the choice is still remembered) |
| View and sort | one 显示 menu | a two-button view switch and a sort menu (phone keeps one menu) |
| Viewer (new) | — | the image at window size: previous / next, 复制链接, fit / actual size, Esc. The preview grows into the viewer where the browser supports view transitions, otherwise it fades |
| Command palette (new) | — | `Ctrl/⌘ K`: finds images (name, ID, SHA-256 prefix) and folders anywhere, runs 上传图片 / 新建文件夹 / 新建令牌, goes to any page, switches the theme. Enter opens, `Ctrl/⌘ Enter` copies the link |
| Uploads | tray at the top of the list, pushes rows down | panel floating at the bottom right (above the tab bar on phones), collapsible; the list does not move. Same rows, states and actions |
| Drop overlay | opaque box over the panel | the window dims, a dashed frame marks it, a label names the target folder |
| Status | figures, maintenance, log, runtime | same order with 损坏 added to the figures and disk usage as one bar (图片 / 数据库 / 其他 / 可用) with its values beside it |
| Settings | addresses as text | the two addresses copy on click |
| Setup | one field for the code | the same field shown as three groups of four characters |
| API reference | own centred layout with a top bar | the console shell: the table of contents is the sidebar, the title and its two links are the panel header, content starts on the same gutter. The current section is marked. Below 1100 px the contents become a row of tabs under the header |

## Alignment

One gutter (20 px, 16 px on phones) carries every title, section heading, row, image edge and the checkbox column.
What sits on it is the glyph or the text, not the box around it:

- header: the title text; on the right a filled button ends on the gutter with its box, a ghost or icon button
  with its label or glyph (pulled out by its inner padding);
- list: the checkbox (16 px in a 32 px hit area) and the grid's image edge share the gutter; hover and selection
  in the grid are a fill 4 px outside the image so the image itself does not move;
- sidebar: the brand mark, every navigation icon, the root of the tree and the avatar's centre share one column;
  folders are indented one step under 图片;
- detail: the first and last header icons, the preview, the text and the footer button labels share the gutter.

## Radius

Four values, as in `DESIGN.md`: 4, 6, 8, 12. Nested corners are the outer radius minus the inset: menu 8 → item 4,
segmented control 6 → segment 4, toast 8 → action 4, palette 12 → result 8, tile fill 12 → image 8 → controls on
the image 4, code block 8 → copy button 4. Rows that touch both edges of their container have no radius.

## Colour, type, motion, shadow

- Radix `gray` for every surface, line and text; lime 9 only as the fill of the primary action (and the brand
  mark); selection, focus, progress and switches are neutral. Status colours mark state only.
- Three type sizes (12 / 13 / 16).
- Motion only where something changes place: dialogs, the drawer and sheet, the detail column, toasts, the
  palette, the uploads panel, the drop overlay, the switch knob, the preview growing into the viewer. All 250 ms
  ease-out, all off under reduced motion. No hover transitions.
- Shadows only on floating layers. No gradients, no blur, no cards, no icons in tinted boxes.

## Decisions removed or moved

- New folder: on the root row of the tree (hover or focus), in each folder's menu, in the palette, and as a
  button in the empty tree. No dialog.
- Copy from the grid: one button on the image copies in the last format used; the other formats stay in the
  detail and the row menu.
- The viewer has no primary action and no settings; clicking the image switches between fit and actual size.
- The palette shows one hint only where it is not obvious (`Ctrl/⌘ Enter` copies the link of the selected image).

## Backend and data notes

- Palette image search: `GET /api/v1/images?folder=root&recursive=true&q=…&limit=6`; folders and commands are
  matched in the client. No new endpoint.
- The disk bar uses fields the status endpoint already returns; 其他 is used − images − database. 损坏 is
  `images.broken`, returned today but not shown.
- The viewer shows the original image URL.

## Fixed from revision 6

- `#/missing`: the title used the class name of the API reference's code block and was dark on dark in the light
  theme. The class is renamed (`nf-title`, `codeblock`); the page was checked in the light theme.

## Additions made to `DESIGN.md` with this approval

- §5: name the gutter and the "glyph on the gutter" rule above.
- §7: floating layers also include the command palette, the uploads panel and the drop label.
- §8: motion also covers the palette, the uploads panel, the drop overlay and the preview-to-viewer transition.
- `docs/product/constraints.md` and `docs/ui/capabilities.md`: default view, if grid is chosen.

## Checks run

- Rendered at 1440 × 900, 900 × 800 and 390 × 844, dark and light, in ego lite (Chromium): 41 states exercised
  with no console errors.
- `content_audit.py check`: passed, 663 classified strings (the warnings are `data-id` attributes, as before).
- `validate_workflow.py structure | contract --phase draft | sources`: passed.
- Not checked: Firefox and Safari, real touch devices, screen readers.
