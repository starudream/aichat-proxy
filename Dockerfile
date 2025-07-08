ARG CAMOUFOX_VERSION=develop

FROM golang:1.24.4-bookworm AS builder

WORKDIR /build

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates tzdata bash make \
    ; \
    rm -rf /var/lib/apt/lists/*

COPY . .

RUN make swag bin version-bin

FROM starudream/aichat-proxy-camoufox:${CAMOUFOX_VERSION}

RUN mv supervisor/conf.d/aichat-proxy.conf.bak supervisor/conf.d/aichat-proxy.conf

COPY --from=builder /build/bin/aichat-proxy bin/aichat-proxy
