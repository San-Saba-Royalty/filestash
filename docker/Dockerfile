# Stage 1: Build backend from local source
FROM golang:1.26-trixie AS builder

WORKDIR /home/filestash/

# Copy source (respects .dockerignore — node_modules, dist, .git excluded)
COPY . .

RUN apt-get update -qq && \
    apt-get install -y --no-install-recommends \
        curl make \
        libjpeg-dev libtiff-dev libpng-dev libwebp-dev \
        libraw-dev libheif-dev libgif-dev libvips-dev \
    > /dev/null 2>&1 && \
    make init && \
    make build

# Stage 2: Minimal runtime image
FROM debian:stable-slim

WORKDIR /app/

COPY --from=builder /home/filestash/dist/ .

RUN apt-get update -qq && \
    apt-get install -y --no-install-recommends \
        apt-utils curl ffmpeg \
        libjpeg-dev libtiff-dev libpng-dev libwebp-dev \
        libraw-dev libheif-dev libgif-dev \
    && useradd --no-create-home filestash \
    && chown -R filestash:filestash /app/ \
    && find /app/data/ -type d -exec chmod 770 {} \; \
    && find /app/data/ -type f -exec chmod 760 {} \; \
    && chmod 730 /app/filestash \
    && rm -rf /var/lib/apt/lists/* /tmp/*

USER filestash

EXPOSE 8334

CMD ["/app/filestash"]
