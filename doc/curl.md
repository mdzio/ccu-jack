# CURL-Beispiele

Im Folgenden werden einige Beispielaufrufe mit dem [Werkzeug CURL](https://curl.haxx.se) gezeigt. Manche Antworten wurden aus Platzgründen gekürzt.

## Datenpunkt lesen

Das Lesen eines Datenpunktes erfolgt über die HTTP-Methode GET. Der aktuelle Wert kann also auch mit einem Web-Browser gelesen werden. In der Web-UI der CCU kann unter _Einstellungen_ → _Geräte_ die Seriennummer des Kanals ermittelt werden. Der `:` in der Seriennummer muss duch `/` ersetzt werden. Der Parametername ist beispielsweise  _STATE_ bei einem Schaltaktor oder _LEVEL_ bei einem Rollladenaktor.

Aufruf:
```
curl http://localhost:2121/device/HEQ0123456/1/STATE/~pv
```
Antwort:
```json
{
  "ts": 1546297200000,
  "v": false,
  "s": 0
}
```

Die Eigenschaft `v` enthält den aktuellen Wert des Datenpunktes. `ts` enthält den Zeitstempel der letzten Aktualisierung des Wertes (Millisekunden seit 1.1.1970 UTC, hier 1.1.2019 00:00 MEZ). `s` gibt Auskunft über die Qualität des Wertes (hier immer 0 → Gut).

## Datenpunkt setzen

Das Setzen eines Datenpunktes erfolgt über die HTTP-Methode PUT.

Aufruf (Linux):
```
curl -X PUT -d '{"v":true}' http://localhost:2121/device/HEQ0123456/1/STATE/~pv
```
Aufruf (Windows):
```
curl -X PUT -d "{""v"":true}" http://localhost:2121/device/HEQ0123456/1/STATE/~pv
```
Antwort: 
```
HTTP/1.1 200 OK
```

Fehler werden über den HTTP-Status angezeigt.

## Erkundung von / (Wurzelverzeichnis)

Aufruf:
```
curl http://127.0.0.1:2121
```
Antwort:
```json
{
  "description": "Root of the CCU-Jack VEAP server",
  "identifier": "root",
  "title": "Root",
  "~links": [
    {
      "rel": "domain",
      "href": "~vendor",
      "title": "Vendor Information"
    },
    {
      "rel": "domain",
      "href": "device",
      "title": "Devices"
    },
    {
      "rel": "domain",
      "href": "sysvar",
      "title": "System variables"
    }
  ]
}
```

Die Eigenschaft `href` der Objekte im `~links`-Array kann zur weiteren Erkundung des VEAP-Servers genutzt werden.

## Erkundung von /device (Geräte)

```
curl http://127.0.0.1:2121/device
```
Antwort:
```json
{
  "description": "CCU Devices",
  "identifier": "device",
  "title": "Devices",
  "~links": [
    {
      "rel": "device",
      "href": "NEQ1234567",
      "title": "Rollladen Bad"
    },
    {
      "rel": "device",
      "href": "OEQ0123456",
      "title": "Türkontakt Terrasse"
    },
    {
      "rel": "device",
      "href": "GEQ0123456",
      "title": "Bewegungsmelder Flur EG"
    },
    ...
  ]
}
```

## Erkundung von /device/OEQ0123456 (Türkontakt Terrasse)

```
curl http://localhost:2121/device/OEQ0123456
```
Antwort:
```json
{
  "address": "OEQ0123456",
  "aesActive": 0,
  "availableFirmware": "",
  "children": [
    "OEQ0123456:0",
    "OEQ0123456:1"
  ],
  "direction": 0,
  "firmware": "2.4",
  "flags": 1,
  "group": "",
  "identifier": "OEQ0123456",
  "index": 0,
  "interface": "PEQ0123456",
  "linkSourceRoles": "",
  "linkTargetRoles": "",
  "paramsets": [
    "MASTER"
  ],
  "parent": "",
  "parentType": "",
  "rfAddress": 5855058,
  "roaming": 0,
  "rxMode": 12,
  "team": "",
  "teamChannels": null,
  "teamTag": "",
  "title": "Türkontakt Terrasse",
  "type": "HM-Sec-SC-2",
  "version": 16,
  "~links": [
    {
      "rel": "channel",
      "href": "0",
      "title": "Türkontakt Terrasse:0"
    },
    {
      "rel": "channel",
      "href": "1",
      "title": "Türzustand Terrasse"
    },
    {
      "rel": "devices",
      "href": "..",
      "title": "Devices"
    }
  ]
}
```

## Erkundung von /device/OEQ0123456/1 (Kanal 1)

```
curl http://localhost:2121/device/OEQ0123456/1
```
Antwort:
```json
{

    "address": "OEQ0123456:1",
    "aesActive": 1,
    "availableFirmware": "",
    "children": null,
    "direction": 1,
    "firmware": "",
    "flags": 1,
    "group": "",
    "identifier": "1",
    "index": 1,
    "interface": "",
    "linkSourceRoles": "KEYMATIC SWITCH WINDOW_SWITCH_RECEIVER WINMATIC",
    "linkTargetRoles": "",
    "paramsets": [
        "LINK",
        "MASTER",
        "VALUES"
    ],
    "parent": "OEQ0123456",
    "parentType": "HM-Sec-SC-2",
    "rfAddress": 0,
    "roaming": 0,
    "rxMode": 0,
    "team": "",
    "teamChannels": null,
    "teamTag": "",
    "title": "Türzustand Terrasse",
    "type": "SHUTTER_CONTACT",
    "version": 16,
    "~links": [
        {
            "rel": "parameter",
            "href": "INSTALL_TEST",
            "title": "INSTALL_TEST"
        },
        {
            "rel": "parameter",
            "href": "LOWBAT",
            "title": "LOWBAT"
        },
        {
            "rel": "parameter",
            "href": "STATE",
            "title": "STATE"
        },
        {
            "rel": "parameter",
            "href": "ERROR",
            "title": "ERROR"
        },
        {
            "rel": "device",
            "href": "..",
            "title": "Türkontakt Terrasse"
        },
        {
            "rel": "room",
            "href": "/room/1248",
            "title": "Wohnzimmer"
        },
        {
            "rel": "function",
            "href": "/function/1243",
            "title": "Sicherheit"
        }
    ]
}
```

Bei den `~links` sind ebenfalls Verweise auf die Räume (`"rel":"room"`) und Gewerke (`"rel":"function"`) des Kanals zu finden. In den `title`-Eigenschaften sind in der Regel die benutzerspezifischen Namen zu finden. Die Abfrageadresse eines Datenpuntkes ändert sich also nicht, wenn der Kanal in der CCU umbenannt wird. Das gleiche gilt auch für Systemvariablen.

## Erkundung von /device/OEQ0123456/1/STATE (Parameter STATE)

```
curl http://localhost:2121/device/OEQ0123456/1/STATE
```
Antwort:
```json
{
  "control": "DOOR_SENSOR.STATE",
  "default": false,
  "flags": 1,
  "id": "STATE",
  "identifier": "STATE",
  "maximum": true,
  "minimum": false,
  "operations": 5,
  "tabOrder": 0,
  "title": "STATE",
  "type": "BOOL",
  "unit": "",
  "~links": [
    {
      "rel": "channel",
      "href": "..",
      "title": "Türzustand Terrasse"
    },
    {
      "rel": "service",
      "href": "~pv",
      "title": "PV Service"
    }
  ]
}
```

Der Dienst `PV Service` kennzeichnet einen Datenpunkt. Durch Anhängen von `/~pv` an die Adresse des Datenpunktes kann auf den aktuellen Wert zugegriffen werden.

## Erkundung von /room (Räume)

```
curl http://localhost:2121/room
```
Antwort:
```json
{
    "description": "Rooms of the ReGaHss",
    "identifier": "room",
    "title": "Rooms",
    "~links": [
        {
            "rel": "room",
            "href": "1248",
            "title": "Wohnzimmer"
        },
        {
            "rel": "room",
            "href": "2194",
            "title": "Außengelände"
        },
        {
            "rel": "room",
            "href": "6510",
            "title": "Bad"
        },
        ...
    ]
}
```

Das oben genannte gilt analog für die Erkundung der Gewerke (`/function`).

## Erkundung von /room/123 (Raum)

```
curl http://localhost:2121/room/1248
```
Antwort:
```json
{
    "identifier": "1248",
    "title": "Wohnzimmer",
    "~links": [
        {
            "rel": "collection",
            "href": "..",
            "title": "Rooms"
        },
        {
            "rel": "channel",
            "href": "/device/GEQ0123456/1",
            "title": "Bewegung Wohnzimmer"
        },
        {
            "rel": "channel",
            "href": "/device/GEQ0012345/1",
            "title": "Rollladen Esstisch"
        },
        ...
    ]
}
```

In der `~links`-Eigenschaft werden die einem Raum zugewiesenen Kanäle aufgelistet. Über den enthaltenen Verweis kann direkt zu den Datenpunkten gesprungen werden.

Das Gleiche gilt analog für ein Gewerk.

## Lesen einer Geräte- pzw. Kanalkonfiguration

HomeMatic Geräte und die zugehörigen Kanäle können Konfigurationoptionen besitzen. Sie befinden sich im sogenannten Parametersatz MASTER. Über die REST-API des CCU-Jacks kann dieser Parametersatz gelsesen und gesetzt werden. Im Folgenden Beispiel werden die Konfigurationsdaten eines Kanals einer Innensirene gelesen.

Aufruf:
```
curl  http://localhost:2121/device/NEQ0123456/1/$MASTER/~pv
```
Antwort:
```json
{
    "ts": 1583613359343,
    "v": {
        "AES_ACTIVE": false,
        "SOUND_ID": 64,
        "STATUSINFO_MINDELAY": 2,
        "STATUSINFO_RANDOM": 1,
        "TRANSMIT_TRY_MAX": 6
    },
    "s": 0
}
```

## Setzen einer Geräte- pzw. Kanalkonfiguration

Das Setzen einer Geräte- pzw. Kanalkonfiguration erfolgt über die HTTP-Methode PUT. Es brauchen nur die Parameter angegeben zu werden, die auch geändert werden sollen. Im Folgenden Beispiel wird der Signalton einer Innensirene geändert.

Aufruf (Linux):
```
curl -X PUT -d '{"v":{"SOUND_ID":64}}' http://localhost:2121/device/NEQ0123456/1/$MASTER/~pv
```
Aufruf (Windows):
```
curl -X PUT -d "{""v"":{""SOUND_ID"":64}}" http://localhost:2121/device/NEQ0123456/1/$MASTER/~pv
```
Antwort: 
```
HTTP/1.1 200 OK
```

Fehler werden über den HTTP-Status angezeigt.

## Erkundung von /~vendor (Herstellerinformationen)

```
curl http://127.0.0.1:2121
```
Antwort:
```json
{
  "description": "Information about the server and the vendor",
  "identifier": "~vendor",
  "serverDescription": "VEAP-Server for the HomeMatic CCU",
  "serverName": "CCU-Jack",
  "serverVersion": "1.0.0-alpha.2",
  "title": "Vendor Information",
  "veapVersion": "1",
  "vendorName": "info@ccu-historian.de",
  "~links": [
    {
      "rel": "item",
      "href": "statistics",
      "title": "HTTP(S) Handler Statistics"
    },
    {
      "rel": "collection",
      "href": "..",
      "title": "Root"
    }
  ]
}
```
