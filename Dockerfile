# syntax=docker/dockerfile:1
# Multi-arch: buildx (CI) sets BUILDPLATFORM/TARGETPLATFORM/TARGETARCH.
# Legacy `docker build` works too — all three default to the host.
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/localsend-nas ./cmd/localsend-nas

FROM alpine:3.21
RUN adduser -D -H -h /data localsend && mkdir -p /data && chown localsend:localsend /data
COPY --from=build /out/localsend-nas /usr/local/bin/localsend-nas
USER localsend
# Web UI on :8080 (unprivileged); LocalSend protocol on 53317.
# Override any of these with flags or LOCALSEND_NAS_* env vars.
ENV LOCALSEND_NAS_LISTEN=:8080 \
    LOCALSEND_NAS_DATA_DIR=/data
EXPOSE 8080 53317 53317/udp
VOLUME /data
ENTRYPOINT ["localsend-nas"]
