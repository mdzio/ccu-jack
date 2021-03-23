# Docker

## Start von CCU Jack mit einem fertigen Image

Am einfachsten geht das Starten des Containers (unter Linux) mit Docker-Compose. Dazu einfach die Datei `docker-compose.yml` herunterladen und folgenden Befehl ausführen:

   ```bash
   docker-compose up -d .
   ```

In der `docker-compose.yml` können Netzwerk-Ports, die in der eigenen Umgebung nicht genutzt werden, auskommentiert werden.

## Eigenes Docker Image bauen und nutzen

Um einen eigenes CCU-Jack-Container-Image zu bauen gibt es zwei verschiedene Möglichkeiten

### Variante 1: Image aus stabilen "Release" von mdzio/ccu-jack bauen

das ist die empfohlene Variante, wenn man auf der sicheren Seite sein möchte.

1. `Dockerfile` und `build-release.sh` herunterladen
2. Docker-Image bauen:

   ```bash
   sh build-release.sh
   ```

### Variante 2: Image aus dem aktuellen Code im Repo von mdzio/ccu-jack bauen

das ist die Variante, wenn man neueste Änderungen aus dem Repo benötigt

1. `Dockerfile` und `build-nightly.sh` herunterladen
2. Docker-Image bauen:

   ```bash
   sh build-nightly.sh
   ```

### Konfiguration und Start des Containers

- nun die Verzeichnisse `conf` und `cert` erstellen, und dort die eigene Konfiguration bzw. die eigenen Zertifikate speichern.
- Starten direkt über Docker:

   ```bash
   docker run -d -v "$PWD"/conf:/app/conf --name ccu-jack ccu-jack:latest
   ```

- Alternativ: im `docker-compose.yml` die `image` Zeile ersetzen durch

  ```yaml
  image: ccu-jack:latest
  ```

   und starten mit

   ```bash
   docker-compose up -d .
   ```
