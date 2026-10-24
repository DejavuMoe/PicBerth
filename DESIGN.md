# PicBerth design rules

Binding rules for every prototype under `designs/` and for the production console in `web/`. Product scope and
content boundaries are in `docs/product/constraints.md`; the approved prototype is the visual truth for anything
these rules leave open.

## 1. Copy

- No middle dot `·` as a separator, in copy or in composed values. Separate values with layout: their own
  element and a gap, a column, or a line break.
- No explanatory or descriptive text that the layout, the control or the label already makes clear. No hints
  under fields unless the user cannot finish the task without them.
- Write for the instance administrator: plain, precise product words, not chatty and not invented. Reuse the
  terms in `docs/ui/capabilities.md` (图片, 上传, 回收站, 令牌, 公开, 私有, 永久删除 …).
- Length follows the job: navigation and buttons are one or two words; states are short phrases; errors say
  what is wrong and, when it is not obvious, what to do.
- One name per thing everywhere (nav, title, empty state, toast).

## 2. Hierarchy

- Build hierarchy with size, weight, colour and spacing, not with extra labels.
- No eyebrows or overlines: no small caption above a heading, and never an uppercase English one. Keep a small
  label only when it adds a category or context the heading does not carry.
- One primary action (lime fill) per view. Everything else is secondary or ghost.
- Order each screen by how often a thing is used: the job first, metadata after, rare and destructive actions
  last or inside a menu.

## 3. Colour

All colours come from [Radix Colors](https://www.radix-ui.com/colors) 3.0.0. No other hex, rgb or named colour
appears in styles; components use semantic tokens that point at Radix steps.

| Scale | Use |
|---|---|
| `gray` | every surface, line and text |
| `lime` | the fill of the primary action only |
| `red` | danger: destructive actions, errors, invalid fields |
| `amber` | warnings |
| `green` | success |
| `blackA` / `whiteA` | scrims, overlays on images, contrast ink |

Steps are used for their Radix purpose:

| Step | Purpose | Tokens |
|---|---|---|
| 1–2 | app and panel backgrounds | `--bg`, `--panel` (dark: 1 / 2; light: 2 / 1) |
| 3 | component fill, hover of ghost controls | `--fill`, `--hover`, `--recessed` |
| 4 | pressed or selected fill | `--active`, `--sel` |
| 5 | selected segment (dark) | `--seg-on` |
| 6 | separators and panel edges | `--hairline` |
| 7 | control borders | `--line` |
| 8 | hovered control borders, focus ring, disabled text | `--line-hover`, `--ring`, `--faint` |
| 9 | solid fills: primary action, danger button, status dots | `--accent`, `--danger-fill`, `--warning` |
| 10 | hover of solid fills | `--accent-hover`, `--danger-fill-hover` |
| 11 | secondary text, status text | `--muted`, `--danger-text`, `--warning-text`, `--success-text` |
| 12 | primary text | `--text` |

- Lime 9 is a bright step: text and icons on it use gray 12 of the light scale (`--accent-ink`) in both themes.
  Red 9 takes white ink (`whiteA 12`).
- Status colours mark state only (a dot, a badge, status text), never decoration. No coloured text links.
- Alpha steps (`gray-a*`) are used for fills and lines so they sit correctly on every surface.

## 4. Type

- Inter Variable (self-hosted) for Latin, system CJK fonts for Chinese, `ui-monospace` for IDs, hashes, URLs.
- Three sizes only: `--fs-s` 12 px (metadata, badges, captions), `--fs-m` 13 px (body, controls),
  `--fs-l` 16 px (page, dialog and detail titles; figures; text inputs on touch screens so iOS does not zoom).
  12 px is the minimum.
- Weights 400, 510 and 590. Differentiate with weight and `--text` / `--muted` before adding a size.
- Numbers that line up use `tabular-nums`.

## 5. Alignment and spacing

- 4 px grid. Paddings 4, 8, 12, 16, 20, 24.
- Align optically, not by box: an icon-only button in a row aligns its glyph, not its hit area, with the text
  edge above or below (pull it out by the button's inner padding); icons beside text are centred on the
  x-height; a leading icon in a button shortens that side's padding by 2 px.
- Text in neighbouring columns shares one baseline. Mixed sizes in a row align on the baseline, not the box.
- Right-align numbers in tables; keep units in the field, not after it.

## 6. Radius

| Radius | Use |
|---|---|
| 4 | badges, tags, checkboxes, kbd, items inside menus, small buttons nested in a container |
| 6 | buttons, inputs, selects, segmented controls, chips |
| 8 | menus, popovers, toasts, image tiles, previews, dropzone |
| 12 | the content panel, dialogs, sheets |
| pill | switches, counters, avatars only |

- Nested corners are concentric: inner radius = outer radius − inset (minimum 2). A 4 px padded menu with
  radius 8 has items with radius 4; a 2 px padded segmented control with radius 6 has segments with radius 4.
- No radius on full-bleed rows, list rows or table cells.

## 7. Lines, elevation and shadows

- Every border, divider and outline-style edge is 0.5 px (`--bw`). The only exception is the 2 px keyboard
  focus ring, which must stay visible.
- Surfaces are separated by step changes and 0.5 px lines, not shadows. A shadow is allowed only on layers
  that float over content (menus, popovers, dialogs, sheets, toasts, the phone bulk bar) and is a single soft
  `blackA` shadow.
- Never a coloured stripe on one edge of a card, row or alert. Selection and the open item are shown with a
  fill (`--sel`), not an edge marker.

## 8. Motion

- Default to no transition. Hover, focus and selection changes are instant.
- Motion is kept only where it explains a change of place: dialogs, drawers and sheets entering and leaving,
  the detail column opening, toasts arriving, a switch knob moving. These use `--dur` 250 ms and
  `--ease` `ease-out`.
- Loading indicators (skeleton pulse, indeterminate progress) are the only looping animations.
- `prefers-reduced-motion: reduce` removes all of it.

## 9. Decisions and interaction

- Ask the user to decide as rarely as possible. Prefer sensible defaults, undo (a toast with 撤销) and
  reversible actions over questions.
- When a decision is unavoidable, put it where the user is already choosing: an inline second step on the same
  button (吊销 → 确认吊销) rather than a separate dialog. A blocking dialog is reserved for 永久删除, which
  cannot be undone and can act on many images at once.
- Each action has one home. Do not repeat the same button in an alert, a header and a row.
- Every control is product-drawn (menus, selects, comboboxes, checkboxes); no native popups.
