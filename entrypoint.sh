#!/bin/bash
set -e

cleanup() {
    kill $FIREFOX_PID $X11VNC_PID $NOVNC_PID $XVFB_PID $SOCAT_PID 2>/dev/null
    wait $FIREFOX_PID $X11VNC_PID $NOVNC_PID $XVFB_PID $SOCAT_PID 2>/dev/null
}
trap cleanup SIGTERM SIGINT

RESOLUTION=${RESOLUTION:-1920x1080}
PROFILE_DIR=/config/firefox-data

mkdir -p "$PROFILE_DIR"

rm -f /tmp/.X99-lock /tmp/.X11-unix/X99

Xvfb :99 -screen 0 ${RESOLUTION}x24 -ac -nolisten tcp &
XVFB_PID=$!

for i in $(seq 10); do
    if xdpyinfo -display :99 >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

x11vnc -display :99 -forever -nopw -quiet &
X11VNC_PID=$!

/usr/share/novnc/utils/novnc_proxy --vnc 127.0.0.1:5900 --listen 3000 --web /usr/share/novnc &
NOVNC_PID=$!

cat > "$PROFILE_DIR/user.js" <<EOF
user_pref("browser.startup.homepage", "about:blank");
user_pref("startup.homepage_welcome_url", "about:blank");
user_pref("browser.shell.checkDefaultBrowser", false);
user_pref("browser.sessionstore.resume_from_crash", false);
user_pref("datareporting.policy.dataSubmissionEnabled", false);
user_pref("app.update.auto", false);
EOF

FIREFOX_ARGS=(
    --new-instance
    -no-remote
    -foreground
    --profile "$PROFILE_DIR"
    --remote-debugging-port 9222
    --remote-allow-origins '*'
    --remote-allow-hosts '*'
)

if [ -n "$PROXY" ]; then
    echo "proxy: $PROXY"
    PROXY_HOST=$(echo "$PROXY" | sed -e 's#^.*://##' -e 's/:.*$//')
    PROXY_PORT=$(echo "$PROXY" | sed -e 's#^.*://##' -e 's/^[^:]*://' -e 's#/.*$##')
    cat >> "$PROFILE_DIR/user.js" <<EOF
user_pref("network.proxy.type", 1);
user_pref("network.proxy.http", "$PROXY_HOST");
user_pref("network.proxy.http_port", $PROXY_PORT);
user_pref("network.proxy.ssl", "$PROXY_HOST");
user_pref("network.proxy.ssl_port", $PROXY_PORT);
user_pref("network.proxy.no_proxies_on", "localhost, 127.0.0.1, [::1], .ru");
EOF
else
    echo "proxy: none"
fi

if [ -n "$FIREFOX_ARGS_EXTRA" ]; then
    FIREFOX_ARGS+=($FIREFOX_ARGS_EXTRA)
fi

/opt/firefox/firefox "${FIREFOX_ARGS[@]}" &
FIREFOX_PID=$!

echo "waiting for Firefox Remote Agent on port 9222..."
for i in $(seq 30); do
    if curl -sf http://127.0.0.1:9222/ >/dev/null 2>&1; then
        echo "Firefox Remote Agent is ready"
        break
    fi
    sleep 1
done

# Bridge the Docker-published port 9222 to Firefox's loopback listener on
# the same port. Firefox RemoteAgent binds 127.0.0.1 only (httpd.js can't
# bind an IPv4 address, Bug 1783938), while Docker's published port targets
# the container eth0. Bind socat to the eth0 IP specifically so it does not
# collide with Firefox's 127.0.0.1:9222. The same port is kept on both sides
# so the Host header Firefox sees matches its server identity.
CONTAINER_IP=$(hostname -I | awk '{print $1}')
socat TCP-LISTEN:9222,fork,reuseaddr,bind="$CONTAINER_IP" TCP:127.0.0.1:9222 &
SOCAT_PID=$!

wait $FIREFOX_PID