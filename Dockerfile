FROM debian:bookworm-slim

ARG FIREFOX_URL=https://download.mozilla.org/?product=firefox-nightly-latest-ssl&os=linux64&lang=en-US

RUN apt-get update && apt-get install -y --no-install-recommends \
    firefox-esr \
    xvfb \
    x11vnc \
    novnc \
    x11-utils \
    ca-certificates \
    curl \
    xz-utils \
    dnsutils \
    dumb-init \
    fonts-liberation \
    fonts-noto-color-emoji \
    fontconfig \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/firefox \
    && curl -fsSL "$FIREFOX_URL" -o /tmp/firefox.tar.xz \
    && tar -xJf /tmp/firefox.tar.xz -C /opt/firefox --strip-components=1 \
    && rm -f /tmp/firefox.tar.xz

ENV DISPLAY=:99
ENV RESOLUTION=1920x1080
ENV FIREFOX_ARGS_EXTRA=""
ENV PROXY=""

EXPOSE 9222 5900 3000

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/entrypoint.sh"]