# PicBerth

English | [简体中文](README.md)

A personal self-hosted image hosting service designed for a single administrator: **a single binary + a single local data directory**. It statically embeds a Svelte 5 web management console, stores image files on local disk, and keeps metadata in modernc SQLite. The compiled binary requires no Node.js runtime and has no dependencies on external databases, Redis caches, or object storage services.

> **Note**: PicBerth's embedded web management console is currently available in Simplified Chinese only. Bilingual documentation covers server deployment, operations, and API integrations; it does not indicate multilingual web UI support.

- **Public access and stable links**: All uploaded images are public. Images use random IDs, and moving or renaming them preserves their links. Random IDs do not provide access control.
- **Folders up to 3 levels deep**: Organize images into hierarchical folders. Moving images or renaming/moving folders **never changes** image public access links. Deleting a folder requires that it contains no active images or subfolders.
- **Ready-to-copy link formats**: Instantly copy Standard URL, Markdown, HTML, and BBCode formats after upload.
- **Scoped API tokens**: Two permission scopes: `upload` and `manage`. `upload` tokens can only upload (optionally specifying an existing folder) and query their own uploads; `manage` tokens can query and manage all images, folders, moves, and deletions. No token can manage other tokens or alter server settings.
- **Security defaults**: Listens on loopback `127.0.0.1:12433` by default; identifies image formats from content, ignores filename extensions, and rejects SVG. Web setup requires the one-time code in the server logs; an offline CLI can also create the administrator. CSP, rate limits, and Argon2id password hashing are built in.
- **Backups and integrity checks**: Consistent snapshots and integrity checks flag images with missing or damaged files. Snapshots can move data between supported platforms.

Design constraints: [`docs/product/constraints.md`](docs/product/constraints.md); architecture and API specifications: [`docs/architecture.md`](docs/architecture.md), [`docs/api.md`](docs/api.md).

> **Release status**: `v0.0.2` is being prepared and has not been published. GitHub Releases currently provides `v0.0.1`; the source and Compose build examples in this checkout target the v0.0.2 candidate.

---

## Quick Start (Native Binary)

Download the archive for your platform from [GitHub Releases](https://github.com/dejavumoe/picberth/releases) (available for Linux AMD64 / ARM64, Windows AMD64, macOS ARM64). Extract and run directly without Node.js or additional runtimes:

```sh
picberth serve --data-dir /var/lib/picberth --public-url https://pics.example.com
```

### Initial Setup

1. **Web setup**: On first startup with an empty database, the server log prints a one-time setup code at `WARN` level (formatted like `ABCD-EFGH-JKMN`). Open the public URL in your browser, navigate to the `/setup` page, enter the code, and set the administrator credentials.
2. **Offline CLI initialization**: You can also create the initial administrator via CLI before launching the service:

```sh
echo 'your-secure-password' | picberth admin create --username admin --password-stdin --data-dir /var/lib/picberth
```

Then configure your reverse proxy (Caddy or Nginx) to forward public traffic to `127.0.0.1:12433`. Configuration examples are in [`deploy/`](deploy/).

---

## Docker Deployment

Candidate images built from the current source use `gcr.io/distroless/static-debian13:nonroot`, run as UID/GID `65532:65532`, and contain no shell. The Compose example enables a read-only root filesystem and builds from the current checkout:

```sh
docker compose -f deploy/compose.yaml up -d --build
```

### Deployment Considerations

- **Do not publish ports directly to public interfaces**: The example configuration binds ports exclusively to host `127.0.0.1:12433:12433`. **Never** configure `12433:12433`—Docker writes its own iptables rules and bypasses host ufw / firewalld firewalls, exposing the service directly to the public Internet without reverse proxy protection.
- **Volume recommendations**: On Docker Desktop (macOS / Windows), always use Docker named volumes (e.g. `picberth-data:/data`) to avoid filesystem performance bottlenecks and ownership quirks associated with host directory bind mounts.
- **Docker version requirement**: Docker Engine ≥ 28.3.3 is recommended (resolves CVE-2025-54388, where published ports became reachable from the LAN after a firewalld reload, and fixes loopback publishing isolation boundaries).
- **Embedded Caddy example**: If the host lacks an existing reverse proxy, use [`deploy/compose.caddy.yaml`](deploy/compose.caddy.yaml), where an in-network Caddy container automatically provisions TLS certificates and provides HTTPS access.

---

## Reverse Proxy and TLS

PicBerth is designed to operate behind a reverse proxy that terminates TLS and provides public boundary defense.

- **Request body limits**:
  - **Nginx**: The repository example sets `client_max_body_size 110m;`.
  - **Caddy**: The example sets `max_size 110MB` inside a `request_body` block. The proxy limit should exceed PicBerth's hard upload limit and allow for multipart overhead.
- **Real client IP**: Set `PICBERTH_TRUSTED_PROXIES` to the reverse proxy IP or CIDR (`127.0.0.1` for local proxies; Docker bridge gateway like `172.30.112.1` for Docker setups). PicBerth resolves `X-Forwarded-For` from right to left, stripping trusted proxies and ignoring client-forged entries. If untrusted proxies are used, rate limiting lumps all clients together and the status page will display an `untrusted-forwarded` warning.
- **Public URLs and image URLs**: Image links are generated from `PICBERTH_PUBLIC_URL` / `PICBERTH_IMAGE_URL`, without using request Host headers. Existing links keep their old address after a domain change; preserve that address or redirect it if those links must continue working.

Reference configurations: [`deploy/Caddyfile`](deploy/Caddyfile) and [`deploy/nginx.conf`](deploy/nginx.conf).

---

## Configuration

Settings are evaluated in order of precedence: **CLI flags > Environment variables > Config file (`--config`, `KEY=value` lines) > Defaults**.

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `PICBERTH_LISTEN` | `--listen` | `127.0.0.1:12433` | Listen address and port; logs a warning if bound to a non-loopback address |
| `PICBERTH_DATA_DIR` | `--data-dir` | OS application data directory | Storage directory; must be on a local filesystem, NFS / SMB mounts unsupported |
| `PICBERTH_PUBLIC_URL` | `--public-url` | `http://<listen>` | Public base URL for the web console and default base URL for image links |
| `PICBERTH_IMAGE_URL` | `--image-url` | Same as `PUBLIC_URL` | Base URL for public image links; can be pointed to a dedicated CDN or image domain |
| `PICBERTH_TRUSTED_PROXIES` | `--trusted-proxies` | Empty | Comma-separated list of trusted proxy IPs or CIDRs; use `127.0.0.1` for local proxies |
| `PICBERTH_HARD_MAX_UPLOAD` | `--hard-max-upload` | `104857600` (100 MiB) | Hard maximum upload size per file (in bytes) |
| `PICBERTH_SNAPSHOT_DIR` | `--snapshot-dir` | `<data-dir>/snapshots` | Default output directory for snapshots |
| `PICBERTH_LOG_LEVEL` | `--log-level` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `PICBERTH_LOG_FORMAT` | `--log-format` | `text` | Logging output format (`text` or `json`) |

> **Override rule**: Explicitly setting an environment variable or flag to an empty value overrides the corresponding setting from the configuration file. For example, passing `--trusted-proxies ''` clears any trusted proxy list set in the config file.

Per-upload size limits, allowed image formats (JPEG, PNG, GIF, WebP, AVIF), same-folder deduplication, trash retention days, and audit log retention days can be adjusted in the web console under 「设置」 (Settings).

---

## Upload Tool Integrations

Create API tokens in the web console under 「令牌」 (Tokens). The creation confirmation dialog provides ready-to-copy configurations for curl, PicGo / PicList, and ShareX.

### HTTP API Upload

```sh
curl -H "Authorization: Bearer pb_..." \
  -F file=@screenshot.png \
  -F folder=3n7vq2k9xwbd4f6h \
  https://pics.example.com/api/v1/images
```

- **Parameters**: File payload is sent in the `file` field; the optional `folder` field accepts a 16-character folder ID (omit or leave empty to upload to root). PicBerth never creates folders implicitly; supplying an unknown folder ID returns 404.

### Third-Party Client Configurations

- **PicGo / PicList** (using built-in custom web-uploader plugin):
  - API URL: `https://pics.example.com/api/v1/images`
  - POST parameter name: `file`
  - JSON path: `url`
  - Custom headers: `{"Authorization": "Bearer pb_..."}`
- **ShareX**: Download the `.sxcu` configuration file from the token dialog and double-click to import it directly into ShareX.

### Token Permission Scopes

Token permissions are strictly divided into two scopes:
- `upload`: Allows uploading files (optionally into an existing folder) and viewing only images uploaded by that specific token.
- `manage`: Allows viewing and managing all images, managing folder structures, moving/renaming/deleting/restoring images.

No API token can manage other tokens or modify server configuration. Interactive API documentation is available directly in the web console at 「令牌 → API 文档」 (`/docs/api`), and machine-readable OpenAPI definitions are served at `/api/v1/openapi.json`.

---

## Backup, Restore, and Maintenance

### Creating Snapshots and Restoring

```sh
# Create a snapshot via CLI when service is stopped (use the web console Status page when running)
picberth snapshot --out /backup/picberth

# Use the timestamped subdirectory printed by snapshot, which contains MANIFEST.json
picberth restore --from /backup/picberth/20261117T120000Z --data-dir /var/lib/picberth-new

# Verify file hashes and filesystem structural integrity
picberth check --full --data-dir /var/lib/picberth-new
```

- **Never copy a running database directly**: Copying `picberth.db` while the server is active can result in a corrupted WAL state. Always generate an atomic consistency snapshot using the CLI or the web console status page, and then back up the snapshot directory with tools like restic, borg, or rsync.
- **Snapshots contain full credentials**: Snapshots contain the database and all referenced images, restoring administrator credentials, sessions, and API tokens. Restore only from trusted backups protected against tampering.
- **File permissions and directory security**: Other operating-system accounts must not be able to write the data or backup directories. Use restrictive Unix permissions or Windows ACLs. PicBerth opens databases through absolute SQLite paths, not an isolated filesystem sandbox.
- **Sensitive data deletion**: Historical snapshots and schema migration backups have independent retention lifecycles and are not automatically modified when online sessions expire, tokens are revoked, or audit logs age out. When purging historical sensitive data, manage backup archives accordingly.
- **Automated migrations and rollback**: Database migrations run automatically on startup. Before migrating, the system creates an automated schema backup at `<data-dir>/backups/picberth-schema<N>-<UTC>.db` (retaining the 5 newest copies). If the database schema is newer than the binary, startup is refused; roll back safely by restoring `picberth.db` from this pre-migration copy with the previous binary version.
- **Integrity checking mechanisms**:
  - **Quick check** (`quick`): Verifies file presence and sizes for active and trashed records, completes interrupted moves, and cleans expired temporary upload chunks.
  - **Full check** (`full`): In addition to quick checks, recalculates and compares SHA-256 hashes for all files and quarantines untracked orphan files into `<data-dir>/quarantine/` (orphans are never automatically deleted). Only a successful full check clears image corruption flags (`broken_at`).

---

## Development

The project uses [mise](https://mise.jdx.dev) to manage and pin development toolchain versions:

```sh
mise install              # Install pinned versions of Go, Node, and pnpm from mise.toml
mise run web:install      # Install frontend dependencies using frozen-lockfile
mise run dev              # Start Go backend server (data directory at .devdata)
mise run web:dev          # In a separate terminal: start Vite dev server (http://127.0.0.1:5173, proxies /api and /i)
mise run test             # Run full test suite (go vet + go test + svelte-check + E2E type checks)
mise run web:e2e:install  # Install Playwright Chromium and OS dependencies
mise run web:e2e          # Run Playwright E2E integration tests against built binary
mise run build            # Build production single binary (outputs dist/picberth with embedded SPA)
```

> **Note**: The `test`, `dev`, and `build` tasks automatically build frontend embedded assets first. On a clean checkout, if running `go test ./...`, `go vet ./...`, or `go run ./cmd/picberth` directly without mise, run `mise run web:build` once beforehand.
>
> On Windows, running browser dependencies and Playwright E2E tests within WSL is recommended for Linux parity.

```powershell
linux-task.ps1 -Mode build -Project <project-path> -Command 'export PATH="$HOME/.local/bin:$PATH"; mise run web:e2e:install && mise run web:e2e'
```

---

## Release and Verification

Release rehearsal and official release workflows are documented in [`docs/releasing.md`](docs/releasing.md).

Each release archive contains the binary and MIT license. `SHA256SUMS` is a separate release asset; archives also have GitHub build attestations. Released container images include an SBOM and build provenance.

Licensed under the [MIT License](LICENSE).
