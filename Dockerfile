# syntax=docker/dockerfile:1

ARG TARGETARCH
# Use the same complete, checksummed pnpm distributions as mise.lock.
FROM scratch AS pnpm-amd64
ADD --checksum=sha256:502a275984a4cd01e5579ac72049a241091cac5118e44becf84951130c3e079e https://github.com/pnpm/pnpm/releases/download/v12.10.1/pnpm-linux-x64.tar.gz /pnpm.tar.gz
FROM scratch AS pnpm-arm64
ADD --checksum=sha256:18dbb118de1dbef0e4516c17bf847d6ee206fc4deffb2bb488ec8e1d2814e078 https://github.com/pnpm/pnpm/releases/download/v12.10.1/pnpm-linux-arm64.tar.gz /pnpm.tar.gz
FROM pnpm-${TARGETARCH} AS pnpm

# ---- web: build the Svelte SPA (Vite writes to /src/internal/webui/dist) ----
FROM node:24.22.2-bookworm-slim@sha256:01d566e8d94d8c619591d78652babfccd3c6992a3dffa9ed5ab39e11cf6c70e9 AS web
COPY --from=pnpm /pnpm.tar.gz /tmp/pnpm.tar.gz
RUN mkdir /opt/pnpm && tar -xzf /tmp/pnpm.tar.gz -C /opt/pnpm && ln -s /opt/pnpm/pnpm /usr/local/bin/pnpm && rm /tmp/pnpm.tar.gz
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
ARG VERSION=0.0.1-dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X picberth/internal/buildinfo.Version=${VERSION} -X picberth/internal/buildinfo.Commit=${COMMIT}" -o /out/picberth ./cmd/picberth && mkdir -p /out/root && mkdir -m 0700 /out/root/data

# ---- final: minimal, non-root runtime ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
# The container listens on all interfaces *inside its network namespace*. Never publish
# the port on all host interfaces: deploy/compose.yaml binds it to 127.0.0.1 only.
COPY --from=build /out/picberth /picberth
COPY LICENSE /LICENSE
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
