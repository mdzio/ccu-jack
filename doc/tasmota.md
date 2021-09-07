# Anbindung einer WLAN-Steckdose mit Tasmota-Firmware

## Voraussetzungen

Folgende Schritte sind einmalig auszuführen:

* Installation CCU-Jack als Add-On auf der CCU3 oder RaspberryMatic
* Freischaltung der Ports 2121 und 1883 in der CCU-Firewall

Hilfreiche Werkzeuge:
* [MQTT Explorer](https://mqtt-explorer.com/)

## Konfiguration WLAN-Steckdose

* Die WLAN-Steckdose, wie in der Tasmota-Dokumentation beschrieben, in das eigene Netzwerk einbinden.
* Die IP-Adresse der CCU als MQTT-Server in der Konfiguration eintragen. (Später können noch Zugangsdaten eingerichtet werden.)
* Akivierung von virtuellen Geräten in der Konfigurationsoberfläche des CCU-Jacks (Web-UI CCU → Einstellungen → Systemsteuerung → CCU-Jack → Konfiguration → CCU-Anbindung → Virtuelle Geräte aktivieren)

Beispiel:

![Konfiguration Tasmota](tasmota-config.png)

## MQTT Explorer

Im _MQTT Explorer_ können die gesendeten MQTT-Nachrichten der WLAN-Steckdose betrachtet werden. Dazu muss eine Verbindung zum MQTT-Server auf der CCU aufgebaut werden:

![Konfiguration MQTT Explorer](tasmota-mqtt-explorer.png)

Der Topic-Baum der WLAN-Steckdose ist aus folgendem Bild ersichtlich:

![Topic-Baum](tasmota-topics.png)

## Virtuelles Gerät im CCU-Jack anlegen

In der Web-UI des CCU-Jacks unter _Virtuelle Geräte_ ein neues Gerät mit folgenden Kanälen erstellen:

![Virtuelle Kanäle](tasmota-channels.png)

Mit dem _MQTT Analogwertempfänger_ soll die Betriebsspannung ausgelesen werden. Es können bis zu 32 weitere Kanäle angelegt werden, um auch komplexe Geräte abzubilden.

## Geräteposteingang der CCU

Um zum Geräteposteingang der CCU zu gelangen oben rechts in der Web-UI der CCU auf _Geräte anlernen_ klicken, danach im erscheinenden Dialog auf _Posteingang (1)_. Bei dem Gerät _Fertig_ anklicken. In der Geräteliste der CCU sollte nun das neue virtuelle Gerät mit aufgelistet werden.

## Virtuelles Gerät konfigurieren

Die zwei angelegten Kanäle des virtuellen Gerätes müssen noch über die Web-UI der CCU konfiguriert werden. Die Einstellungen des virtuellen Gerätes in der CCU Web-UI aufrufen: _Einstellungen_ → _Geräte_ → Geräte auswählen → _Einstellen_.

### Kanal für Schaltaktor (1. Kanal)

Folgende Parameterkonfiguration ist zu setzen:

Parameter      | Wert
---------------|------------
COMMAND_TOPIC  | tasmota/cmnd/switch1/POWER
FEEDBACK_TOPIC | tasmota/stat/switch1/POWER

Die restliche Konfiguration ist aus folgendem Bild ersichtlich:

![Topic-Baum](tasmota-parameters.png)

## Kanal für Messwert (2. Kanal)

Folgende Parameterkonfiguration ist zu setzen:

Parameter      | Wert
---------------|------------
TOPIC          | tasmota/tele/switch1/STATE

Die restliche Konfiguration ist aus folgendem Bild ersichtlich:

![Topic-Baum](tasmota-parameters-2.png)

_Hinweis: Wenn die Parameterwerte erneut betrachtet werden, so werden die " (doppelten Hochkommas) auf Grund eines Fehlers in Web-UI der CCU durch HTML-Sonderzeichen ersetzt (s.a. [Hinweis in der Beschreibung der virtuellen Geräte](virtual-devices.md#virtuelle-geräte-im-ccu-jack))._

## Abschluss

Damit ist die WLAN-Steckdose über die Web-UI bedienbar und kann wie ein HM-Gerät auch innerhalb von CCU-Programmen verwendet werden. Weitere Informationen zu den virtuellen Geräten sind in der CCU-Jack Dokumentation zu finden (Kapitel ["Virtuelle Geräte"](https://github.com/mdzio/ccu-jack#virtuelle-geräte) und Kapitel ["Virtuelle Geräte im CCU-Jack"](virtual-devices.md))

