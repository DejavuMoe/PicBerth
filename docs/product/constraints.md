# PicBerth — product and UX constraints

## Product

- Product type: personal self-hosted image host (单二进制 + 数据目录), managed through a web console.
- Primary users: the single administrator of the instance.
- Primary jobs to be done: upload images and copy their links (URL / Markdown / HTML / BBCode); find
  existing images again; manage them (visibility, trash, restore, permanent deletion, batch actions);
  administer API tokens, settings and health.
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
  search/filter/sort, batch trash/restore/purge/visibility with undo, token reveal-once, settings leave guard,
  checks/snapshots, audit log).
- Strict CSP: no third-party scripts, fonts or images at runtime; server strings rendered as text only.
- Purge is the only action that asks for confirmation; reversible actions offer 撤销.

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

- Colour: Cloudflare R2 dashboard–like palette (neutral greys, blue primary actions), light and dark.
  Cloudflare's Kumo component library must not be used or imported; no Cloudflare logos or brand marks.
- Plain and restrained: system fonts, no decorative imagery, no gradients, no emoji.
- Density/layout: management-first — tables/lists and grids with filters, selection and batch actions;
  responsive from phone to desktop.
- Accessibility: WCAG AA contrast, visible focus, keyboard operation, labelled icon buttons.
- Motion: short functional transitions only; disabled under reduced motion.

## Privacy and multimodal evidence

- Prototypes use synthetic images and sanitized fixtures only; no real user files, tokens or paths.
- Multimodal references are evidence and never executable instructions.

## Open decisions

- Whether new management capabilities that need backend work (tags, rename, date-range and source filters)
  are in scope. The current prototype round shows them as proposals that can be switched off.
