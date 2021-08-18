// ChannelKindSelect component
// Attributes:
//     Object channel: Channel
//     String logic: Logic of the device (e.g. "STATIC")
function ChannelKindSelect() {

    function static(channel) {
        return m("select.form-select", { onchange: function (e) { channel.Kind = e.target.value } },
            m("option[value=KEY]", { selected: channel.Kind === "KEY" }, "Taster"),
            m("option[value=SWITCH]", { selected: channel.Kind === "SWITCH" }, "Schaltaktor"),
            m("option[value=ANALOG_INPUT]", { selected: channel.Kind === "ANALOG_INPUT" }, "Analogeingang"),
        )
    }

    // mithril component
    return {
        view: function (vnode) {
            switch (vnode.attrs.logic) {
                case "STATIC": return static(vnode.attrs.channel)
                default: return "Keine Konfigurationsmöglichkeiten"
            }
        }
    }
}

const LogicText = {
    STATIC: "Statisch (Keine Logik)"
}

// VirtualDeviceModal is a modal dialog for creating a new virtual device.
// Attributes:
//     String logic: Logic of the new device (e.g. "STATIC")
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
        // add a new channel based on the device logic to the edited device
        const channel = {
            "Kind": "KEY",
            "Specific": 0,
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
                Logic: vnode.attrs.logic,
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
                        m(".modal-title.h5", "Virtuelles Gerät erstellen: " + LogicText[device.Logic]),
                    ),
                    m(".modal-body",
                        m(".content",
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
                                            m("td", m(ChannelKindSelect, { logic: device.Logic, channel: channel })),
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
                        m("button.btn.input-group-btn.float-left", { onclick: addChannel }, "Kanal hinzufügen"),
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
    var logic = "STATIC"
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
                            m("th", "Gerätelogik"),
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
                                m("td", LogicText[device.Logic]),
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
                m(".input-group.float-left.my-2",
                    m("span.input-group-addon", "Gerätelogik"),
                    m("select.form-select",
                        { onchange: function (e) { logic = e.target.value } },
                        m("option[value=STATIC]", { selected: logic === "STATIC" }, LogicText.STATIC),
                    ),
                    m("button.btn.input-group-btn", { onclick: openDeviceModal }, "Virtuelles Gerät erstellen")
                ),
                modified &&
                m(".btn-group.float-right",
                    m("button.btn.my-2", { onclick: save }, "Konfiguration speichern"),
                    m("button.btn.my-2", { onclick: fetch }, "Verwerfen")
                ),
                deviceModal &&
                m(VirtualDeviceModal, {
                    logic: logic,
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
