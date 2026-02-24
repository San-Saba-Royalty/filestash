# ── Stage 1: Build ─────────────────────────────────────────────────────────────
FROM golang:1.26-trixie AS builder

WORKDIR /home/filestash/

# Install C build tools and image processing libraries needed by CGO plugins
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        gcc git make pkg-config \
        libjpeg-dev libtiff-dev libpng-dev libwebp-dev \
        libraw-dev libheif-dev libgif-dev libvips-dev \
    && rm -rf /var/lib/apt/lists/*

# Download Go modules in a separate layer for better cache reuse
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source (respects .dockerignore)
COPY . .

# Git SHA injected by CI — BUILD_REF must be ≥ 7 chars or ServeIndex panics.
# .git is excluded from the build context (.dockerignore), so we overwrite
# constants_generated.go AFTER go generate runs (which writes an empty ref).
ARG GIT_SHA=00000000000000000000000000000000000000000

RUN go generate ./server/... && \
    printf 'package common\n\nfunc init() {\n\tBUILD_REF = "%s"\n\tBUILD_DATE = "%s"\n}\n' \
        "${GIT_SHA}" "$(date +%Y%m%d)" \
        > server/common/constants_generated.go && \
    CGO_ENABLED=1 go build --tags "fts5" -o dist/filestash cmd/main.go

# ── Stage 2: Minimal runtime image ─────────────────────────────────────────────
FROM debian:trixie-slim

WORKDIR /app/

COPY --from=builder /home/filestash/dist/ .

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl ffmpeg \
        libjpeg62-turbo libtiff6 libpng16-16t64 libwebp7 \
        libheif1 libgif7 libvips42t64 \
    && useradd --no-create-home filestash \
    && chown -R filestash:filestash /app/ \
    && chmod 730 /app/filestash \
    && rm -rf /var/lib/apt/lists/* /tmp/*

USER filestash

EXPOSE 8334

CMD ["/app/filestash"]
