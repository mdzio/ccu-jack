# Please don't use this filke directly but
# build the image with Shellscript "./build-release.sh"

FROM alpine

ARG BUILD_DATE
ARG BUILD_VERSION

LABEL org.opencontainers.image.created=$BUILD_DATE \
      org.opencontainers.image.version=$BUILD_VERSION \
      org.opencontainers.image.title="CCU-Jack" \
      org.opencontainers.image.description="REST/MQTT-Server for the HomeMatic CCU" \
      org.opencontainers.image.vendor="CCU-Jack OpenSource Project" \
      org.opencontainers.image.authors="mdzio <info@ccu-historian.de>" \
      org.opencontainers.image.licenses="GPL-3.0 License" \
      org.opencontainers.image.url="https://github.com/mdzio/ccu-jack" \
      org.opencontainers.image.documentation="https://github.com/mdzio/ccu-jack/blob/master/README.md"

# Set work directory
WORKDIR /app

# Get the latest relase from github and extract it locally
RUN wget -q -O - "https://github.com/mdzio/ccu-jack/releases/download/v${BUILD_VERSION}/ccu-jack-linux-${BUILD_VERSION}.tar.gz" | tar -xvzC . && \
    mkdir -p /app/conf /app/cert /data && \
    adduser -h /app -D -H ccu-jack -u 1000 -s /sbin/nologin && \
    chown -R ccu-jack:ccu-jack /data && chmod -R g+rwX /data && \
    chown -R ccu-jack:ccu-jack /app && chmod -R g+rwX /app

USER ccu-jack

# MQTT, MQTT TLS, CCU-Jack VEAM/UI, CCU-Jack VEAM/UI TLS, CUxD
EXPOSE 1883 8883 2121 2122 2123

# Add a healthcheck (default every 30 secs)
HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
    CMD wget --spider -S -q http://localhost:2121/ui/ 2>&1 | head -1 || exit 1

# workaround to save certificates and make config readable
WORKDIR /app
# Start it up with full path

ENTRYPOINT [ "/app/ccu-jack","-config","/app/conf/ccu-jack.cfg" ]