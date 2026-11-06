# syntax=docker/dockerfile:1

# The interface and the Go build run on the build machine's own architecture;
# Go then cross-compiles, which is far faster than emulating arm64.
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS server
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed_prod.go web/embed_dev.go web/
COPY --from=web /src/web/dist web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -tags embedweb -trimpath -ldflags="-s -w" -o /out/lucy ./cmd/lucy

FROM alpine:3.22
# ca-certificates verifies monitored sites; tzdata makes day boundaries follow TZ.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 lucy \
    && mkdir /data && chown lucy:lucy /data
COPY --from=server /out/lucy /usr/local/bin/lucy
USER lucy
ENV LUCY_DB=/data/lucy.db \
    LUCY_ADDR=0.0.0.0:8080
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/api/auth/status || exit 1
ENTRYPOINT ["lucy"]
