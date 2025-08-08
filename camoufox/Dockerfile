FROM python:3.12.11-slim-bookworm

WORKDIR /app

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates tzdata bash net-tools wget curl unzip supervisor \
      libgtk-3-0 libx11-xcb1 libasound2 \
      fluxbox tigervnc-standalone-server novnc \
    ; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    pip install -U camoufox[geoip]==0.4.11 playwright==1.52.0; \
    python -m camoufox fetch; \
    mkdir -p /root/.cache/ms-playwright-go/; \
    ln -s /usr/local/lib/python3.12/site-packages/playwright/driver /root/.cache/ms-playwright-go/1.52.0

ENV DISPLAY=:0.0 \
    DISPLAY_WIDTH=1920 \
    DISPLAY_HEIGHT=1080
COPY supervisor supervisor
CMD ["supervisord", "-c", "/app/supervisor/supervisord.conf"]
