// ChannelKindSelect component
// Attributes:
//     Object channel: Channel
function ChannelKindSelect() {

    // mithril component
    return {
        view: function (vnode) {
            const channel = vnode.attrs.channel
            return m("select.form-select", { onchange: function (e) { channel.Kind = e.target.value } },
                m("option[value=STATIC_KEY]", { selected: channel.Kind === "STATIC_KEY" }, "Statischer Taster"),
                m("option[value=STATIC_SWITCH]", { selected: channel.Kind === "STATIC_SWITCH" }, "Statischer Schaltaktor"),
                m("option[value=STATIC_ANALOG]", { selected: channel.Kind === "STATIC_ANALOG" }, "Statischer Analogwerteingang"),
                m("option[value=STATIC_DOOR_SENSOR]", { selected: channel.Kind === "STATIC_DOOR_SENSOR" }, "Statischer Fenster-/Türkontakt"),
                m("option[value=STATIC_DIMMER]", { selected: channel.Kind === "STATIC_DIMMER" }, "Statischer Dimmer"),
                m("option[value=STATIC_TEMPERATURE]", { selected: channel.Kind === "STATIC_TEMPERATURE" }, "Statischer Temperatursensor"),
                m("option[value=STATIC_POWER_METER]", { selected: channel.Kind === "STATIC_POWER_METER" }, "Statische Energiemessung"),
                m("option[value=STATIC_ENERGY_COUNTER]", { selected: channel.Kind === "STATIC_ENERGY_COUNTER" }, "Statischer Energiezähler"),
                m("option[value=STATIC_GAS_COUNTER]", { selected: channel.Kind === "STATIC_GAS_COUNTER" }, "Statischer Gaszähler"),
                m("option[value=MQTT_KEY_SENDER]", { selected: channel.Kind === "MQTT_KEY_SENDER" }, "MQTT Sendetaster"),
                m("option[value=MQTT_KEY_RECEIVER]", { selected: channel.Kind === "MQTT_KEY_RECEIVER" }, "MQTT Empfangstaster"),
                m("option[value=MQTT_SWITCH]", { selected: channel.Kind === "MQTT_SWITCH" }, "MQTT Schaltaktor"),
                m("option[value=MQTT_SWITCH_FEEDBACK]", { selected: channel.Kind === "MQTT_SWITCH_FEEDBACK" }, "MQTT Schaltaktor mit Rückmeldung"),
                m("option[value=MQTT_ANALOG_RECEIVER]", { selected: channel.Kind === "MQTT_ANALOG_RECEIVER" }, "MQTT Analogwertempfänger"),
                m("option[value=MQTT_DOOR_SENSOR]", { selected: channel.Kind === "MQTT_DOOR_SENSOR" }, "MQTT Fenster-/Türkontakt"),
                m("option[value=MQTT_DIMMER]", { selected: channel.Kind === "MQTT_DIMMER" }, "MQTT Dimmer"),
                m("option[value=MQTT_TEMPERATURE]", { selected: channel.Kind === "MQTT_TEMPERATURE" }, "MQTT Temperatursensor"),
                m("option[value=MQTT_POWER_METER]", { selected: channel.Kind === "MQTT_POWER_METER" }, "MQTT Energiemessung"),
                m("option[value=MQTT_ENERGY_COUNTER]", { selected: channel.Kind === "MQTT_ENERGY_COUNTER" }, "MQTT Energiezähler"),
                m("option[value=MQTT_GAS_COUNTER]", { selected: channel.Kind === "MQTT_GAS_COUNTER" }, "MQTT Gaszähler"),
            )
        }
    }
}

// HMTypeSelect component
// Attributes:
//     Object device: Device
function HMTypeSelect() {
    const hmTypes = [
        ["HmIP-MIO16-PCB", "Multi IO Modulplatine - 4x4"],
        ["HM-LC-Dim1T-FM", "Dimmaktor 1-fach, Phasenab., UP"],
        ["HM-LC-Dim1T-DR", "Dimmaktor 1-fach, Phasenab., Huts."],
        ["HM-LC-Dim1TPBU-FM", "Dimmaktor 1-fach für Markens."],
        ["HM-LC-Sw1-FM", "Schaltaktor 1-fach, UP"],
        ["HM-LC-Sw2-FM", "Schaltaktor 2-fach, UP"],
        ["HM-LC-Sw1-DR", "Schaltaktor 1-fach, Huts."],
        ["HM-LC-Sw4-DR", "Schaltaktor 4-fach, Huts."],
        ["HM-LC-Sw1PBU-FM", "Schaltaktor 1-fach für Markens."],
        ["HM-ES-PMSw1-Pl", "Funk-Schaltaktor mit Leistungsmessung"],
        ["HM-RC-4", "Handsender 4 Tasten"],
        ["HM-RC-8", "Handsender 8 Tasten"],
        ["HM-RC-12", "Handsender 12 Tasten"],
        ["HM-RC-19", "Handsender 19 Tasten"],
        ["HM-Sec-SC-2", "Tür-/ Fensterkontakt"],
        ["HM-SCI-3-FM", "Schließerkontaktschnittstelle 3-fach, UP"],
        ["HmIP-STHO", "Temp.- und Luftf.-sensor außen"],
        ["HmIP-STHD", "Temp.- und Luftf.-sensor innen mit Display"],
        ["HM-ES-TX-WM", "Zähler-Sensor"],
    ]

    // mithril component
    return {
        view: function (vnode) {
            const device = vnode.attrs.device
            return m("select.form-select", { onchange: function (e) { device.HMType = e.target.value } },
                hmTypes.map(hmType =>
                    m("option", { value: hmType[0], selected: device.HMType === hmType[0] }, hmType[0] + " " + hmType[1])
                )
            )
        }
    }
}

// VirtualDeviceModal is a modal dialog for creating a new virtual device.
// Attributes:
//     Object config: Current configuration
//     Function onclose(boolean configModified): Callback for closing modal
function VirtualDeviceModal() {
    // virtual devices configuration, set by attribute
    var vdConfig
    // callback when closing, set by attribute
    var onClose
    // currently configured device
    var device
    // generated next serial no.
    var nextSerialNo

    function cancel() {
        // config not modified
        onClose(false)
    }

    function create() {
        // add device to configuration
        vdConfig.Devices[device.Address] = device
        // update next serial no.
        vdConfig.NextSerialNo = nextSerialNo
        // config modified
        onClose(true)
    }

    function addChannel() {
        // limit number of channels
        if (device.Channels.length >= 64) {
            return
        }
        // add a new channel to the edited device
        const channel = {
            "Kind": "STATIC_KEY",
            "MasterParamset": {}
        }
        device.Channels.push(channel)
    }

    function deleteChannel(index) {
        device.Channels.splice(index, 1)
    }

    // mithril component
    return {
        oninit: function (vnode) {
            onClose = vnode.attrs.onclose
            vdConfig = vnode.attrs.config.VirtualDevices

            // generate device address
            const addr = "JACK" + String(vdConfig.NextSerialNo).padStart(6, "0")
            nextSerialNo = vdConfig.NextSerialNo + 1

            // create the device for editing
            device = {
                Address: addr,
                HMType: "HmIP-MIO16-PCB",
                Specific: 0,
                Channels: []
            }
        },
        view: function (vnode) {
            // validation
            const ok = device.Channels.length > 0

            return m(".modal.active",
                m(".modal-overlay"),
                m(".modal-container",
                    m(".modal-header",
                        m("button.btn.btn-clear.float-right", { onclick: cancel }),
                        m(".modal-title.h5", "Virtuelles Gerät erstellen"),
                    ),
                    m(".modal-body",
                        m(".content",
                            m(".input-group",
                                m("span.input-group-addon", "Gerätesymbol"),
                                m(HMTypeSelect, { device: device }),
                            ),
                            m("table.table",
                                m("thead",
                                    m("tr",
                                        m("th", "Kanalnr."),
                                        m("th", "Kanaltyp"),
                                        m("th", ""),
                                    ),
                                ),
                                m("tbody",
                                    device.Channels.map(function (channel, idx) {
                                        return m("tr",
                                            m("td", ":" + (idx + 1)),
                                            m("td", m(ChannelKindSelect, { channel: channel })),
                                            m("td",
                                                m("button.btn.btn-sm",
                                                    { onclick: function () { deleteChannel(idx) } },
                                                    m("i.icon.icon-delete")
                                                ),
                                            ),
                                        )
                                    }),
                                ),
                            ),
                            device.Channels.length != 0 ||
                            m("p", "Keine Kanäle angelegt."),
                        ),
                    ),
                    m(".modal-footer",
                        m("button.btn.input-group-btn.float-left",
                            { class: device.Channels.length < 64 ? "" : "disabled", onclick: addChannel },
                            "Kanal hinzufügen"
                        ),
                        m(".btn-group",
                            m("button.btn", { class: ok ? "" : "disabled", onclick: create }, "Erstellen"),
                            m("button.btn", { onclick: cancel }, "Abbrechen"),
                        ),
                    ),
                )
            )
        }
    }
}

// VirtualDeviceTitle component
// Attributes:
//     addr: Address of device, String
function VirtualDeviceTitle() {
    var title = ""
    // mithril component
    return {
        oninit: function (vnode) {
            m.request("/virtdev/" + vnode.attrs.addr).then(function (resp) {
                title = resp.title
            }).catch(function (e) {
                title = "?"
            })
        },
        view: function (vnode) {
            return title
        }
    }
}

// VirtualDevices component
function VirtualDevices() {
    var config
    var modified = false
    var errorMessage
    var deviceModal = false

    function fetch() {
        m.request("/~vendor/config/~pv").then(function (resp) {
            if (resp.v !== undefined) {
                config = resp.v
                modified = false
                errorMessage = null
            } else {
                errorMessage = "Ungültiges JSON-Objekt als Konfiguration!"
            }
        }).catch(function (e) {
            errorMessage = errorToString(e)
        })
    }

    function save() {
        m.request("/~vendor/config/~pv", {
            method: "PUT",
            body: { "v": config }
        }).then(function () {
            errorMessage = null
            fetch()
        }).catch(function (e) {
            errorMessage = errorToString(e)
        })
    }

    function deleteDevice(device) {
        delete config.VirtualDevices.Devices[device.Address]
        modified = true
    }

    function openDeviceModal() {
        deviceModal = true
    }

    function closeDeviceModal(configModified) {
        deviceModal = false
        if (configModified) {
            modified = true
        }
    }

    function viewError() {
        return m(".toast.toast-error.my-2",
            m("h4", "Fehler"),
            m("p", "Beschreibung: " + errorMessage),
        )
    }

    function viewContent() {
        if (config == null) {
            return m("p", "Lade Konfiguration...")
        } else {
            const deviceAddrs = Object.keys(config.VirtualDevices.Devices).sort()
            return [
                modified &&
                m(".toast.toast-warning",
                    m("p", "Konfigurationsänderungen sind noch nicht gespeichert!")
                ),
                m("table.table",
                    m("thead",
                        m("tr",
                            m("th", "Seriennummer"),
                            m("th", "Name"),
                            m("th", "Anzahl Kanäle"),
                            m("th", ""),
                        ),
                    ),
                    m("tbody",
                        deviceAddrs.map(function (addr) {
                            const device = config.VirtualDevices.Devices[addr]
                            return m("tr",
                                m("td", device.Address),
                                m("td", m(VirtualDeviceTitle, { addr: addr })),
                                m("td", device.Channels.length),
                                m("td",
                                    m("button.btn.btn-sm", { onclick: function () { deleteDevice(device) } }, m("i.icon.icon-delete")),
                                ),
                            )
                        }),
                    ),
                ),
                deviceAddrs.length != 0 ||
                m("p", "Keine virtuellen Geräte vorhanden."),
                m("button.btn.my-2.float-left", { onclick: openDeviceModal }, "Virtuelles Gerät erstellen"),
                modified &&
                m(".btn-group.float-right",
                    m("button.btn.my-2", { onclick: save }, "Konfiguration speichern"),
                    m("button.btn.my-2", { onclick: fetch }, "Verwerfen")
                ),
                deviceModal &&
                m(VirtualDeviceModal, {
                    config: config,
                    onclose: closeDeviceModal
                })
            ]
        }
    }

    // mithril component
    return {
        oninit: function (vnode) {
            fetch()
        },
        view: function (vnode) {
            return [
                m("h1", "Virtuelle Geräte"),
                errorMessage ?
                    viewError() : (
                        config != null && config.VirtualDevices.Enable === true ?
                            viewContent() :
                            m("p", "Virtuelle Geräte sind nicht in der Konfiguration aktiviert!")
                    )
            ]
        }
    }
}
