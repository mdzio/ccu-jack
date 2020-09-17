// The ProcessValue component shows a VEAP process value. The component renders
// to 3 table cells (value, timestamp, state). The process value is updated
// cyclically. The component must be recreated, if the address is changed.
// Attributes: addr=VEAP variable address, cycle=update cycle time [ms]
function ProcessValue() {
    // VEAP address of the PV
    var addr
    // update rate [ms]
    var cycle = 400
    // current PV or error
    var pv, error
    // last value as JSON, value changed
    var last, changed = false
    // timer ID
    var timer

    // update PV
    function update() {
        m.request(addr).then(function (resp) {
            changed = false
            if (resp.v !== undefined) {
                pv = {
                    ts: resp.ts ? new Date(resp.ts) : null,
                    v: resp.v,
                    s: resp.s,
                }
                error = null

                // detect changed PV
                var cur = JSON.stringify(pv.v)
                if (last != null && cur !== last) {
                    changed = true
                }
                last = cur
            } else {
                error = "Ungültiges JSON-Objekt für einen Prozesswert!"
                last = null
            }
        }).catch(function (err) {
            pv = null
            error = errorToString(err)
            last = null
            changed = false
        })
        timer = setTimeout(update, cycle)
    }

    // mithril component
    return {
        oninit: function (vnode) {
            addr = vnode.attrs.addr + "/~pv"
            if (vnode.attrs.cycle) {
                cycle = vnode.attrs.cycle
            }
            // start updates
            update()
        },
        onremove: function () {
            if (timer !== null) {
                clearTimeout(timer)
            }
        },
        view: function () {
            if (pv) {
                return [
                    m("td",
                        { class: changed ? "bg-warning" : null },
                        m("strong", toPrettyString(pv.v))
                    ),
                    m("td", toPrettyString(pv.ts)),
                    m("td", stateToString(pv.s)),
                ]
            } else if (error) {
                return m("td.bg-error[colspan=3]", "Fehler: " + error)
            } else {
                return null
            }
        }
    }
}

// PVInput is a generic form input field. Attributes: onchange=function to
// receive changed process value (null if invalid), topv=function to convert
// text to PV, totext=function to convert PV to text, pv=initial process value
function PVInput() {
    // onchange handler
    var onChange
    // converter
    var toPV, toText
    // last PV (null: invalid)
    var pv
    // current text input
    var txt

    function update(evt) {
        txt = evt.target.value
        var v = toPV(txt)
        pv = (v == null) ? null : { v: v }
        onChange(pv)
    }

    // mithril component
    return {
        oninit: function (vnode) {
            // copy attributes to local vars
            onChange = vnode.attrs.onchange
            toPV = vnode.attrs.topv
            toText = vnode.attrs.totext
            pv = vnode.attrs.pv
            txt = toText(pv.v)
        },
        view: function () {
            return m(".form-group",
                m("label.form-label[for=number-pv-input]", "Zahlenwert:"),
                m("input.form-input[type=text][id=number-pv-input][placeholder=Zahlenwert]", {
                    class: pv != null ? "is-success" : "is-error",
                    value: txt, oninput: update,
                    oncreate: function (vnode) {
                        vnode.dom.focus() // focus input
                        vnode.dom.select() // select text
                    }
                })
            )
        }
    }
}

// PVRadio is an input field for a boolean process value. Attributes:
// onchange=function to receive changed process value, pv=initial process value
function PVRadio() {
    // onchange handler
    var onChange
    // last PV (null: invalid)
    var pv

    function update(evt) {
        pv = (evt.target.value == "1") ? { v: true } : { v: false }
        onChange(pv)
    }

    // mithril component
    return {
        oninit: function (vnode) {
            // copy attributes to local vars
            onChange = vnode.attrs.onchange
            pv = vnode.attrs.pv
        },
        view: function () {
            return m(".form-group",
                m("label.form-label", "Digitalwert:"),
                m("label.form-radio",
                    m("input[type=radio][name=pv][value=1]", { checked: pv.v, onchange: update }),
                    m("i.form-icon"),
                    "Ein / 1 / True"
                ),
                m("label.form-radio",
                    m("input[type=radio][name=pv][value=0]", { checked: !pv.v, onchange: update }),
                    m("i.form-icon"),
                    "Aus / 0 / False"
                ),
            )
        }
    }
}

// PVModal is a modal dialog for setting a PV.
function PVModal() {
    // onclose handler
    var onclose
    // process value address
    var addr
    // error message
    var error
    // last read process value, process value to write
    var readPV, writePV
    // process value type
    var type

    // modal should close
    function close() {
        onclose()
    }

    // get current process value
    function getPV() {
        readPV = null
        writePV = null
        type = null
        error = null
        m.request(addr).then(function (resp) {
            if (resp.v !== undefined) {
                readPV = resp
                writePV = { v: readPV.v } // remove timestamp and state
                type = typeof readPV.v
                if (type != "number" && type != "string" && type != "boolean") {
                    type = null
                }
            } else {
                error = "Ungültiges JSON-Objekt für einen Prozesswert!"
            }
        }).catch(function (err) {
            error = errorToString(err)
        })
    }

    // set new process value
    function setPV() {
        m.request(addr, {
            method: "PUT",
            body: writePV
        }).then(function () {
            error = null
            // close modal on success
            close()
        }).catch(function (err) {
            error = errorToString(err)
        })
    }

    function viewContent() {
        if (error) {
            return m(".toast.toast-error.my-2", m("p", errorToString(error)))
        }
        if (readPV == null) {
            return m("p", "Prozesswert wird gelesen...")
        }
        switch (type) {
            case "number":
                return m(PVInput, {
                    pv: readPV,
                    totext: function (v) { return numberToString(v) },
                    topv: function (txt) { return stringToNumber(txt) },
                    onchange: function (pv) { writePV = pv }
                })
            case "string":
                return m(PVInput, {
                    pv: readPV,
                    totext: function (v) { return v },
                    topv: function (txt) { return txt },
                    onchange: function (pv) { writePV = pv }
                })
            case "boolean":
                return m(PVRadio, {
                    pv: readPV,
                    onchange: function (pv) { writePV = pv }
                })
            default:
                return m(".toast.toast-error.my-2", m("p", "Datentyp wird nicht unterstützt!"))
        }
    }

    // mithril component
    return {
        oninit: function (vnode) {
            onclose = vnode.attrs.onclose
            addr = vnode.attrs.addr + "/~pv"
            // retrieve current value
            getPV()
        },
        view: function () {
            return m(".modal.modal-sm.active",
                m(".modal-overlay", { onclick: close }),
                m(".modal-container",
                    m(".modal-header",
                        m("button.btn.btn-clear.float-right", { onclick: close }),
                        m(".modal-title.h5", "Datenpunkt Setzen"),
                    ),
                    m(".modal-body", m(".content", viewContent())),
                    m(".modal-footer",
                        m(".btn-group",
                            m("button.btn", { disabled: type == null, onclick: setPV }, "Setzen"),
                            m("button.btn", { onclick: close }, "Schließen")
                        ),
                    ),
                )
            )
        }
    }
}
