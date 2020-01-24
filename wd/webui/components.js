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
        onremove: function (vnode) {
            if (timer !== null) {
                clearTimeout(timer)
            }
        },
        view: function (vnode) {
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
