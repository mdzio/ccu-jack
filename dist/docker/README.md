# Docker

Um ccu-jack in einem Docker Container laufen zu lassen sind folgende Schritte nötig:

1. Dockerfile und ggf. docker-compose.yml von github herunterladen
2. Docker image bauen:

  ```bash
   export BUILD_VERSION=$(curl -Ls https://api.github.com/repos/mdzio/ccu-jack/releases/latest | grep -oP '"tag_name": "v\K(.*)(?=")')
   
   docker build --rm --no-cache \
    --build-arg BUILD_VERSION="${BUILD_VERSION}" \
    --build-arg BUILD_DATE="$(date +"%Y-%m-%dT%H:%M:%SZ")" \
    --tag ccu-jack:latest --tag ccu-jack:${BUILD_VERSION} .
  ```
3. Verzeichnisse "conf" und "cert" erstellen und dort die eigene Konfiguration bzw Zertifikate zu speichern. 
4. a) direkt über docker laufen lassen:
   ```
   docker run -d -v "$PWD"/conf:/app/conf --name ccu-jack ccu-jack:latest
   ```

    b) oder mit docker-compose: 
    ```
    docker-compose up -d .
    ```

In der compose-Datei kann man ports, die in der eigenen Umgebung nicht genutzt werden (z.B. die TLS Ports), auskonfigurieren. 

