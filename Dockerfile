# 1. build the image with
#   export BUILD_VERSION=$(curl -Ls https://api.github.com/repos/mdzio/ccu-jack/releases/latest | grep -oP '"tag_name": "v\K(.*)(?=")')
#   docker build --rm --no-cache \
#    --build-arg BUILD_VERSION="${BUILD_VERSION}" \
#    --build-arg BUILD_DATE="$(date +"%Y-%m-%dT%H:%M:%SZ")" \
#    --tag ccu-jack:latest --tag ccu-jack:${BUILD_VERSION} .
#
# 2. you have to mount your config DIRECTORY into container and run the image, i.e.
#   docker run -d -v $PWD/conf:/app/conf --name ccu-jack ccu-jack:latest

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
RUN apk add --no-cache curl && \
    curl -SL "https://github.com/mdzio/ccu-jack/releases/download/v${BUILD_VERSION}/ccu-jack-linux-${BUILD_VERSION}.tar.gz" | tar -xvzC . && \
    mkdir -p /app/conf /data && \
    adduser -h /app -D -H ccu-jack -u 1000 -s /sbin/nologin && \
    chown -R ccu-jack:root /data && chmod -R g+rwX /data && \
    chown -R ccu-jack:root /app && chmod -R g+rwX /app

USER ccu-jack

# MQTT, MQTT TLS, CCU-Jack VEAM/UI, CCU-Jack VEAM/UI TLS
EXPOSE 1883 8883 2121 2122

# Add a healthcheck (default every 30 secs)
#HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
#    CMD curl -Isf -o /dev/null -w "%{scheme}/%{http_version} %{http_code}\n" http://localhost:2121/ui/ || exit 1

# workaround to save certificates and make config readable
WORKDIR /app/conf
# Start it up with full path
CMD ["/app/ccu-jack"]