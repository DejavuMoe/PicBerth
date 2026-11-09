# syntax=docker/dockerfile:1

# ---- web: build the Svelte SPA (Vite writes to /src/internal/webui/dist) ----
FROM node:24.21.0-bookworm-slim@sha256:d6aa754f16b3197301076f047b5def2f02ea1dbbc2ca920407d46d7ec7f87b20 AS web
ADD --checksum=sha256:992070e422dae33eac72268ffbde465e14dae81ed250149299d1fee3d842d60e https://registry.npmjs.org/pnpm/-/pnpm-12.10.1.tgz /tmp/pnpm.tgz
RUN npm install -g --ignore-scripts /tmp/pnpm.tgz && rm /tmp/pnpm.tgz
WORKDIR /src/web
# Wildcard so a missing pnpm-workspace.yaml does not break the COPY
COPY web/package.json web/pnpm-*.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# ---- build: static Go binary with the SPA embedded ----
FROM golang:1.27.3-bookworm@sha256:dc0963f32053e4bc451d24d87e3892d42d01021b61f093e98eb95c0352bb84b0 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=web /src/internal/webui/dist ./internal/webui/dist
ARG VERSION=0.1.0
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X picberth/internal/buildinfo.Version=${VERSION} -X picberth/internal/buildinfo.Commit=${COMMIT}" -o /out/picberth ./cmd/picberth && mkdir -p /out/root && mkdir -m 0700 /out/root/data

# ---- final: minimal, non-root runtime ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
# The container listens on all interfaces *inside its network namespace*. Never publish
# the port on all host interfaces: deploy/compose.yaml binds it to 127.0.0.1 only.
COPY --from=build /out/picberth /picberth
# Copy the parent so the data/ entry keeps its 0700 mode (copying a directory itself only copies its
# contents and recreates the target with 0755).
COPY --from=build --chown=65532:65532 /out/root/ /
ENV PICBERTH_LISTEN=0.0.0.0:12433 PICBERTH_DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 12433
USER 65532:65532
STOPSIGNAL SIGTERM
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/picberth", "healthcheck"]
ENTRYPOINT ["/picberth"]
CMD ["serve"]
