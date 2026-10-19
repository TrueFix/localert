# syntax=docker/dockerfile:1
FROM node:22-bookworm-slim AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o /out/localert ./cmd/localert

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* \
    && mkdir /data && chown 1000:1000 /data
COPY --from=build /out/localert /usr/local/bin/localert
USER 1000:1000
ENV HOME=/data
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/localert"]
CMD ["--addr", "0.0.0.0:8787", "--data-dir", "/data", "--no-open"]
