# Virtuelle Geräte im CCU-Jack

Im Folgenden sind die unterstützten virtuellen Geräte beschrieben. Spezifische Einstellungen der Geräte können in der CCU vorgenommen werden (_Einstellungen_ → _Geräte_ → Gerät auswählen → _Einstellen_).

_Hinweis:_

_Durch einen Fehler in der Web-UI der CCU können zwar die Zeichen `'` und `"` (einfaches und doppeltes Hochkomma) in Werten von Text-Parametern (z.B. SHORT_PAYLOAD) angegeben werden, beim Anzeigen dieser Zeichen werden sie aber fälschlicherweise als HTML-Entitäten (`&#39;` und `&#34;`) kodiert. Bei einem erneuten Setzen werden die HTML-Entitäten durch den CCU-Jack automatisch zurückgewandelt. Zudem dürfen in Text-Parametern zurzeit nur ASCII-Zeichen verwendet werden. Beispielsweise sind die Zeichen `üöäÜÖÄß` nicht zulässig._

## Statische Geräte (Keine Logik)

Statische Geräte besitzen keine interne Logik und keine Einstellmöglichkeiten. Sie dienen dazu, zusätzliche Datenpunkte zu erschaffen, die über die MQTT- und REST-API des CCU-Jacks angesprochen werden können. Gleichzeitig können sie nahtlos in CCU-Programmen verwendet werden.

Kanaltyp            | Ab Version | Funktion
--------------------|------------|-----------------------------------------------------
Taster              | 2.0.11     | Taster (wie die virtuellen Taster in der CCU)
Schaltaktor         | 2.0.11     | Schaltaktor (wie HM-LC-Sw1-Pl)
Analogeingang       | 2.0.11     | Analogeingang (wie HmIP-MIO16-PCB Kanal 1, aber der Eingang kann zusätzlich von der CCU oder von extern _gesetzt_ werden)
Tür-/Fensterkontakt | 2.0.47     | Tür-/Fensterkontakt (wie HM-Sec-SC-2)

## MQTT-Geräte (Senden und Empfangen von MQTT-Nachrichten)

MQTT-Geräte senden bei Zustandsänderungen (z.B. Tastendruck, Schalten eines Aktors) frei konfigurierbare Nachrichten (MQTT-Payload) auf frei konfigurierbaren MQTT-Topics. Zudem können MQTT-Geräte Topics abonnieren und bei eingehenden Nachrichten ihren eigenen Zustand anpassen (z.B. Rückmeldungen von Schaltaktoren, Messwerte). 

Durch die weite Verbreitung des MQTT-Protokolls können eine Vielzahl an Geräten einfach an die CCU angebunden und in die CCU-Automatisierung integriert werden. Im Folgenden sind einige Beispiele aufgelistet:
* [DeLock WLAN-Steckdosen](https://www.delock.de/produkte/G_1744_Geraete.html)
* [Shelly](https://shelly.cloud/)
* Geräte-Firmwares mit MQTT-Unterstützung
  * [Tasmota-Firmware](https://tasmota.github.io/docs/)
  * [Espurna-Firmware](https://github.com/xoseperez/espurna)
  * [ESPEasy-Firmware](https://github.com/letscontrolit/ESPEasy)
  * [ESPHome-Firmware](https://esphome.io)

In der MQTT-Konfiguration der Geräte muss die CCU als MQTT-Server (bzw. Broker) eingetragen werden. Der MQTT-Port 1883 bzw. 8883 muss in der CCU-Firewall freigegeben sein. Die MQTT-Payload wird als Text in UTF-8-Kodierung behandelt. Auf der Diagnose-Seite des CCU-Jacks werden eventuelle Konfigurationsfehler angezeigt.

Kanaltyp                         | Ab Version | Funktion
---------------------------------|------------|-----------------------------------------------------
MQTT Sendetaster                 | 2.0.31     | Taster zum Senden von beliebigen MQTT-Nachrichten
MQTT Empfangstaster              | 2.0.31     | Taster zum Empfangen von beliebigen MQTT-Nachrichten
MQTT Schaltaktor                 | 2.0.31     | Schaltaktor zum Senden von MQTT-Nachrichten beim Ein- und Aussschalten
MQTT Schaltaktor mit Rückmeldung | 2.0.31     | Zusätzlich wird der Status des Schaltaktors durch empfangene MQTT-Nachrichten aktualisiert.
MQTT Analogwertempfänger         | 2.0.31     | Ein Zahlenwert wird aus der MQTT-Nachricht extrahiert und als Analogwert zur Verfügung gestellt.
MQTT Fenster-/Türkontakt         | 2.0.47     | Fenster-/Türkontakt zum Empfangen von MQTT-Nachrichten

### MQTT Sendetaster

Der MQTT-Sendetaster sendet konfigurierbare MQTT-Nachrichten.

Liste der Einstellungsparameter:

Name              | Bedeutung
------------------|-------------------------------------------------------------------------------
SHORT_TOPIC       | MQTT-Topic für einen kurzen Tastendruck
SHORT_PAYLOAD     | MQTT-Payload für einen kurzen Tastendruck
SHORT_RETAIN      | Der MQTT-Server soll die zuletzt gesendete Nachricht speichern.
LONG_TOPIC        | s.o., für langen Tastendruck
LONG_PAYLOAD      | s.o., für langen Tastendruck
LONG_RETAIN       | s.o., für langen Tastendruck

### MQTT Empfangstaster

Der MQTT-Empfangstaster löst einen Tastendruck beim Empfang einer MQTT-Nachricht aus.

Liste der Einstellungsparameter:

Name              | Bedeutung
------------------|-------------------------------------------------------------------------------
SHORT_TOPIC       | MQTT-Topic für einen kurzen Tastendruck. Die Platzhalter + und # werden unterstützt.
SHORT_PATTERN     | Prüfmuster für die MQTT-Payload (abhängig von SHORT_MATCHER)
SHORT_MATCHER     | Vergleichsfunktion für die Überprüfung der Payload mit dem Prüfmuster (EXACT: Die Payload muss dem Prüfmuster entsprechen; CONTAINS: In der Payload muss das Prüfmuster enthalten sein; REGEXP: Das Prüfmuster ist ein regulärer Ausdruck, der zutreffen muss.)
LONG_TOPIC        | s.o., für langen Tastendruck
LONG_PATTERN      | s.o., für langen Tastendruck
LONG_MATCHER      | s.o., für langen Tastendruck

Für reguläre Ausdrücke werden die üblichen Operatoren und Zeichenklassen unterstützt. Weitere Informationen sind in der [Spezifikation](https://github.com/google/re2/wiki/Syntax) zu finden.

### MQTT Schaltaktor                

MQTT-Schaltaktor sendet beim Ein- oder Ausschalten jeweils eine MQTT-Nachricht.

Liste der Einstellungsparameter:

Name              | Bedeutung
------------------|-------------------------------------------------------------------------------
TOPIC             | MQTT-Topic für das Ein- oder Ausschalten
RETAIN            | Der MQTT-Server soll die zuletzt gesendete Nachricht speichern.
ON_PAYLOAD        | MQTT-Payload für das Einschalten
OFF_PAYLOAD       | MQTT-Payload für das Ausschalten

### MQTT Schaltaktor mit Rückmeldung

MQTT-Schaltaktor sendet ebenfalls beim Ein- oder Ausschalten jeweils eine MQTT-Nachricht. Der Zustand wird aber erst aktualisiert, wenn eine Rückmeldung vom MQTT-Gerät eingeht.

Liste der Einstellungsparameter:

Name              | Bedeutung
------------------|-------------------------------------------------------------------------------
COMMAND_TOPIC     | MQTT-Topic für das Ein- oder Ausschalten
RETAIN            | Der MQTT-Server soll die zuletzt gesendete Nachricht speichern.
ON_PAYLOAD        | MQTT-Payload für das Einschalten
OFF_PAYLOAD       | MQTT-Payload für das Ausschalten
FEEDBACK_TOPIC    | MQTT-Topic für die Rückmeldung. Die Platzhalter + und # werden unterstützt.
MATCHER           | Vergleichsfunktion für die Überprüfung der Payload mit dem Prüfmuster (EXACT: Die Payload muss dem Prüfmuster entsprechen; CONTAINS: In der Payload muss das Prüfmuster enthalten sein; REGEXP: Das Prüfmuster ist ein regulärer Ausdruck, der zutreffen muss.)
ON_PATTERN        | Prüfmuster für die MQTT-Payload für den eingeschalteten Zustand (abhängig von MATCHER)
OFF_PATTERN       | Prüfmuster für die MQTT-Payload für den ausgeschalteten Zustand (abhängig von MATCHER)

Für reguläre Ausdrücke werden die üblichen Operatoren und Zeichenklassen unterstützt. Weitere Informationen sind in der [Spezifikation](https://github.com/google/re2/wiki/Syntax) zu finden.

### MQTT Analogwertempfänger        

Der MQTT-Analogwertempfänger extrahiert aus der MQTT-Payload eine als Text übertragene Zahl und stellt sie als Analogwert der CCU zur Verfügung. Diese kann optional ein . (Punkt) als Dezimaltrennzeichen enthalten. Falls die Zahl nicht extrahiert werden kann, so wird der Status vom Analogwert auf _Überlauf_ gesetzt.

Liste der Einstellungsparameter:

Name              | Bedeutung
------------------|-------------------------------------------------------------------------------
TOPIC             | MQTT-Topic für die zu empfangenden Nachrichten. Die Platzhalter + und # werden unterstützt.
PATTERN           | Suchmuster für den Zahlenwert in der MQTT-Payload (abhängig von EXTRACTOR)
EXTRACTOR         | (AFTER: Der nächstliegende Zahlenwert hinter dem Suchmuster wird verwendet; BEFORE: Der nächstliegende Zahlenwert vor dem Suchmuster wird verwendet; REGEXP: Das Suchmuster ist ein regulärer Ausdruck. Der Zahlenwert befindet sich in der Gruppe mit der Nummer REGEXP_GROUP.)
REGEXP_GROUP      | Nummer der zu verwendenden Gruppe des regulären Ausdrucks, wenn EXTRACTOR auf REGEXP gesetzt ist. 

Beispiele:

EXTRACTOR | PATTERN           | REGEXP_GROUP | MQTT-Payload                   | Extrahierter Zahlenwert | Bemerkungen
----------|-------------------|--------------|--------------------------------|-------------------------|--------------------
BEFORE    | cm                | 0            | 100 l 52 cm                    | 52,0                    | REGEXP_GROUP ist egal.
AFTER     | Vcc               | 0            | { "Vcc": 3.3, "Version": 2.2 } | 3,3                     | REGEXP_GROUP ist egal. 
REGEXP    | (\S+) (\S+) (\S+) | 1            | 123 543.31 21.3                | 123,0                   | 1. Zahl wird extrahiert.
REGEXP    | (\S+) (\S+) (\S+) | 2            | 123 543.31 21.3                | 543,31                  | 2. Zahl wird extrahiert.

Für reguläre Ausdrücke werden die üblichen Operatoren und Zeichenklassen unterstützt. Weitere Informationen sind in der [Spezifikation](https://github.com/google/re2/wiki/Syntax) zu finden.

### MQTT Fenster-/Türkontakt (ab v2.0.47)

Der Zustand des virtuellen Fenster-/Türkontakts wird aktualisiert, wenn eine Nachricht vom MQTT-Gerät empfangen wird.

Liste der Einstellungsparameter:

Name              | Bedeutung
------------------|-------------------------------------------------------------------------------
TOPIC             | MQTT-Topic für die Statusmeldungen. Die Platzhalter + und # werden unterstützt.
MATCHER           | Vergleichsfunktion für die Überprüfung der Payload mit dem Prüfmuster (EXACT: Die Payload muss dem Prüfmuster entsprechen; CONTAINS: In der Payload muss das Prüfmuster enthalten sein; REGEXP: Das Prüfmuster ist ein regulärer Ausdruck, der zutreffen muss.)
OPEN_PATTERN      | Prüfmuster für die MQTT-Payload für den geöffneten Zustand (abhängig von MATCHER)
CLOSED_PATTERN    | Prüfmuster für die MQTT-Payload für den geschossenen Zustand (abhängig von MATCHER)
