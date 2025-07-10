ARG CAMOUFOX_VERSION=develop

FROM ghcr.io/starudream/aichat-proxy-camoufox:${CAMOUFOX_VERSION}

RUN mv supervisor/conf.d/aichat-proxy.conf.bak supervisor/conf.d/aichat-proxy.conf

COPY bin/aichat-proxy bin/aichat-proxy
