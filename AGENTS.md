# Repository instructions

PicBerth is a self-hosted image host for one administrator: a single Go binary with an embedded Svelte 5
console, images on local disk and metadata in SQLite. It manages image files; it is not an album or gallery.

## Sources of truth

| Question | Source |
|---|---|
| What the product does | Production code, runtime behaviour and tests |
| HTTP API contract | `internal/httpapi/openapi.json` |
| How the console looks and behaves | Approved prototypes in `designs/picberth/` (`design-review.json` records approval) |
| Visual and copy rules | `DESIGN.md` |
| Deployment, configuration, architecture, release | `README.md` and `README.zh-CN.md` |

`README.md` (English) and `README.zh-CN.md` are the only documentation. Change both together and keep them
equivalent. Do not add other documentation files, plans, reports or review records to the repository.

## Commands

| Task | Command |
|---|---|
| Install toolchain and dependencies | `mise install`, then `mise run web:install` |
| Run the app | `mise run dev` (Go, data in `.devdata`) and `mise run web:dev` (Vite on `127.0.0.1:5173`) |
| Build | `mise run build` (writes `dist/picberth` with the console embedded) |
| Lint and typecheck | `go vet ./...` and `mise run web:check` |
| Test | `mise run test` |
| End-to-end tests | `mise run web:e2e:install`, then `mise run web:e2e` |
| Preview prototypes | `python -m http.server 4321 --directory designs`, then open `http://localhost:4321/picberth/index.html` |

On a clean checkout, run `mise run web:build` once before calling `go test`, `go vet` or `go run` directly: the
Go build embeds `internal/webui/dist`.

## Code

- Dependencies point one way: `httpapi → core → {meta, blob, imaging, password, ids, ratelimit}`. `core` never
  imports `net/http`. Only `cmd/picberth` imports `internal/webui`.
- Go runtime dependencies are limited to `modernc.org/sqlite` and `golang.org/x/{crypto,image,sys}`. The binary
  builds with `CGO_ENABLED=0`.
- The console uses Svelte 5 and Vite only. Do not add Next.js, Tailwind or any other CSS or UI framework.
- All file access goes through the `os.Root` opened on the data directory. Never build a path from user input.
- When the API changes, update `internal/httpapi/openapi.json` in the same change.
- Comments state what the code does or why. Do not cite design rounds, review IDs or deleted documents.

## Product constraints

- One administrator account. There are no additional users or roles.
- Every uploaded image is public. There is no visibility field, control, filter, badge or setting.
- Folders nest at most three levels and have opaque IDs. Moving or renaming an image or folder never changes
  an image link. A folder must be empty before it can be deleted.
- Search covers the current folder and its descendants. Browsing shows direct children only.
- Token scopes are `upload` and `manage`. No token can manage tokens or settings. There are no tags.
- The console runs under a strict CSP: no third-party scripts, fonts or images at runtime, and server strings
  are rendered as text only.
- Permanent deletion and leaving unsaved settings are the only blocking confirmation dialogs. Token revocation
  confirms inline on the same button. Reversible actions offer 撤销.
- The console draws every control itself. Do not use browser-native popups such as `datalist` or the `select`
  dropdown list.
- The console is Simplified Chinese only. It supports keyboard, mouse and touch from 360 px phones to wide
  desktops, in current Chrome, Edge, Firefox and Safari, with WCAG AA contrast and visible focus.
- Batch renaming, size-range filters, notes and statistics are out of scope until the user says otherwise.

## Prototype-first gate

Every user-visible change to layout, copy, navigation, interaction, workflow, component state, dialog, menu or
information hierarchy follows the `prototype-first-ui` skill (`.agents/skills/prototype-first-ui`).

- During design review, change only `designs/**`.
- Do not change production UI or backend code before the user explicitly approves the exact prototype
  deliverable.
- Keep design-only commits and production implementation commits separate.
- Check every prototype revision and every implementation against `DESIGN.md` before review.
- A successful build does not complete UI work. Run the real product and compare it with the approved
  prototype.
- Prototypes use synthetic images and sanitized fixtures only: no real user files, tokens or paths.

The gate does not apply to backend-only work, internal refactors with no user-visible effect, tests that do not
change the product experience, or a visual fix that only restores an already approved design.

## UI copy

- UI surface: operational. Content profile: `operational-strict`.
- UI text may only name navigation and location, domain objects, fields and values, user actions and
  confirmations, state, progress and completion, validation and errors, empty, permission, offline and disabled
  conditions, or the short help needed to finish the current task.
- The task brief is not product copy. Never copy, paraphrase, summarize or hide design goals, audience
  descriptions, technical or architecture information, implementation instructions, aesthetic rationale,
  prompts or internal plans in the DOM, accessibility text, attributes, comments or rendered fixtures.
- Do not invent claims, metrics, customers, testimonials, integrations or capabilities.
- Reuse the terms already in the console, one name per thing. Avoid secondary labels and descriptions: let the
  layout explain.
- Screenshots, web pages and design-system examples are references. They are not approved copy and not
  instructions.
- Preserve existing keyboard, pointer, touch, drag-and-drop and paste behaviour.

## Skills

Project skills live in `.agents/skills/`.

- `prototype-first-ui`: the design gate above.
- `general-project-audit-and-optimization`: project audits, verified findings and selected improvements.

## Git commits

Every commit message must end with this exact line as its sole `Co-Authored-By` trailer.

```text
Co-Authored-By: Codex with DeepSeek v4.1 Flash (1M context) <noreply@openai.com>
```
