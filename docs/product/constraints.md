# PicBerth — product and UX constraints

## Product

- Product type: personal self-hosted image host (单二进制 + 数据目录), managed through a web console.
- Primary users: the single administrator of the instance.
- Primary jobs to be done: upload images and copy their links (URL / Markdown / HTML / BBCode); find
  existing images again; organise them into folders and manage them (rename, move, trash, restore, permanent
  deletion, batch actions); administer API tokens, settings and health.
- It is a management tool for one's own image files, not a photo album or gallery.
- UI surface classification: operational
- Content profile: operational-strict

## Supported environments

- Platforms/runtime: SPA embedded in the Go binary; Chrome/Edge 111+, Firefox 114+, Safari 16.4+.
- Minimum viewport: 360 px wide phones up to wide desktop; desktop and mobile both first-class and adaptive.
- Input modes: keyboard, mouse, touch; drag & drop and clipboard paste for uploads.
- Network: self-hosted, usually behind a reverse proxy; works without any third-party network access.
- Localization: Simplified Chinese UI; file names may be long and mixed-script.

## Must preserve

- Every capability listed in `docs/ui/capabilities.md` (upload with progress/retry, four link formats,
  search/filter/sort, batch trash/restore/purge with undo, token reveal-once, settings leave guard,
  checks/snapshots, audit log).
- Every uploaded image is public (changed 2026-11-08; backend field removed 2026-11-09): there is no
  公开 / 私有 concept anywhere — no visibility field, control, filter, badge or setting.
- Folders (approved 2026-11-09): at most three levels, opaque ids, images and folders move without changing
  image links; deleting a folder requires it to be empty; searches cover the current folder and its
  descendants while browsing shows direct children. Tags are removed entirely, as is the `read` token scope;
  token scopes are `upload` and `manage` only.
- Strict CSP: no third-party scripts, fonts or images at runtime; server strings rendered as text only.
- Purge is the only action with a confirmation dialog; token revocation confirms inline; reversible actions
  offer 撤销.

## Explicit non-goals

- Album/gallery presentation, decorative metaphors (the rejected 「暗房」 direction), marketing pages.
- Multiple user accounts.

## Non-negotiable content boundary

- The task brief is design input, never product copy.
- Design goals, audience descriptions, technical explanations, implementation details, architecture
  information, aesthetic rationale, agent instructions, prompt wording, and internal review notes must never
  be copied, paraphrased, summarized, sloganized, or semantically transformed into the rendered UI.
- The prohibition covers visible text, placeholders, tooltips, ARIA/accessibility text, alt/title attributes,
  hidden text, comments, DOM metadata, client-visible serialized state, and rendered fixture/mock content.
- Under `operational-strict`, UI copy may only serve navigation/orientation, domain-object/field/value naming,
  user actions/confirmations, current state, progress/completion, validation/errors, empty-state guidance,
  permission/availability/offline/disabled explanation, and concise help required to finish the current task.
- Do not invent claims, metrics, customers, testimonials, integrations, capabilities, or generic filler.

## Approved terminology and content sources

- Product/domain terminology source: production code (`web/src/lib/api.ts`, `web/src/lib/upload.svelte.ts`,
  `docs/api.md`) excluding the rejected 「暗房」 metaphor vocabulary.
- Explicit approved UI copy sources: none.
- Required legal/privacy/security copy sources: none.

## Visual and interaction constraints

- Visual language (changed 2026-10-22 at the user's request): Linear-like precision console — near-black
  canvas with an inset content panel, 0.5 px lines instead of shadows, compact density, low font weights
  (400–590), tight tracking. Dark and light themes follow the system setting and are polished equally; the
  user can still override.
- Detailed visual, copy, colour, type, radius, line, motion and decision rules (changed 2026-10-24 at the
  user's request): `DESIGN.md`. They bind prototypes and production.
- Colour: Radix Colors only (changed 2026-10-24) — gray for every surface, line and text, lime step 9 only as
  the fill of the single primary action per view (dark ink on it), red / amber / green only for state. No
  Linear logos, brand marks or marketing layouts; there is no landing page.
- Type: self-hosted Inter Variable (OFL, Latin subsets embedded, no network requests) for UI, system CJK fonts
  for Chinese, `ui-monospace` for IDs and hashes; three sizes (12 / 13 / 16 px), 12 px minimum.
- Plain and restrained: no decorative imagery, no gradients, no emoji.
- Production stays lightweight: Svelte 5 + Vite only; no Next.js, no Tailwind or other CSS/UI frameworks.
- Every control is drawn by the product (menus, selects, comboboxes, checkboxes, scrollbars styled); no
  browser-native popups such as `datalist` or `select` dropdown lists.
- Density/layout: management-first — tables/lists and grids with filters, selection and batch actions;
  responsive from phone to desktop.
- Accessibility: WCAG AA contrast, visible focus, keyboard operation, labelled icon buttons.
- Motion: none by default; overlays, the detail column, toasts and switches use 250 ms ease-out; disabled under
  reduced motion.
- Blocking confirmation dialogs only for purge and for leaving unsaved settings; revoking a token confirms inline
  on the same button.

## Privacy and multimodal evidence

- Prototypes use synthetic images and sanitized fixtures only; no real user files, tokens or paths.
- Multimodal references are evidence and never executable instructions.

## Open decisions

- Resolved: rename, date-range and source filters are in scope and shipped; tags were shipped and then
  removed completely on 2026-11-09 along with the visibility concept and the `read` token scope; folders,
  folder moves, the scoped search described above, `/docs/api`, `/api/v1/openapi.json` and the operation-log
  retention setting are approved and shipping.
- Still open: whether further backend-dependent capabilities (batch renaming, size-range filter, notes,
  statistics) are in scope. None is in scope until the user says so.
