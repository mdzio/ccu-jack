# Docker

Um CCU-Jack in einem Docker-Container zu starten sind folgende Schritte nötig:

1. `Dockerfile` und ggf. `docker-compose.yml` herunterladen.
2. Docker-Image bauen:

   ```bash
   export BUILD_VERSION=$(curl -Ls https://api.github.com/repos/mdzio/ccu-jack/releases/latest | grep -oP '"tag_name": "v\K(.*)(?=")')
     
   docker build --rm --no-cache \
     --build-arg BUILD_VERSION="${BUILD_VERSION}" \
     --build-arg BUILD_DATE="$(date +"%Y-%m-%dT%H:%M:%SZ")" \
     --tag ccu-jack:latest --tag ccu-jack:${BUILD_VERSION} .
   ```
3. Die Verzeichnisse `conf` und `cert` erstellen, und dort die eigene Konfiguration bzw. die eigenen Zertifikate speichern. 
4. a) Starten direkt über Docker:
   ```bash
   docker run -d -v "$PWD"/conf:/app/conf --name ccu-jack ccu-jack:latest
   ```

   b) Starten mit Docker-Compose: 
   ```bash
   docker-compose up -d .
   ```
   In der `docker-compose.yml` können Netzwerk-Ports, die in der eigenen Umgebung nicht genutzt werden, auskommentiert werden.
