# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM node:24-bookworm-slim AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X main.version=$VERSION -X main.defaultUpdateFeed=https://github.com/TrueFix/localert/releases/latest/download/update-manifest.json" -o /out/localert ./cmd/localert

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/* \
    && mkdir /data && chown 1000:1000 /data
COPY --from=build /out/localert /usr/local/bin/localert
ARG VERSION=dev
LABEL org.opencontainers.image.source="https://github.com/TrueFix/localert" \
      org.opencontainers.image.title="Localert" \
      org.opencontainers.image.version=$VERSION
USER 1000:1000
ENV HOME=/data
EXPOSE 8787
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD curl -fsS http://127.0.0.1:8787/api/health || exit 1
ENTRYPOINT ["/usr/local/bin/localert"]
CMD ["--addr", "0.0.0.0:8787", "--data-dir", "/data", "--no-open", "--container"]
