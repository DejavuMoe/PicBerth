# syntax=docker/dockerfile:1

# ---- web: build the Svelte SPA (Vite writes to /src/internal/webui/dist) ----
FROM node:24.21.0-bookworm-slim AS web
RUN npm install -g pnpm@12.10.1
WORKDIR /src/web
# Wildcard so a missing pnpm-workspace.yaml does not break the COPY
COPY web/package.json web/pnpm-*.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# ---- build: static Go binary with the SPA embedded ----
FROM golang:1.27.3-bookworm AS build
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
FROM gcr.io/distroless/static-debian12:nonroot
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
