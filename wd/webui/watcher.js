// Data point watcher
var WatchList = {
    dataPoints: [],
    get length() {
        return this.dataPoints.length
    },
    add: function (name, addr) {
        this.dataPoints.push({
            name: name, addr: addr, pv: null, last: null, changed: false, error: null
        })
    },
    remove: function (addr) {
        this.dataPoints = this.dataPoints.filter(dp => dp.addr !== addr)
    },
    includes: function (addr) {
        return this.dataPoints.findIndex(dp => dp.addr === addr) != -1
    },
    update: function () {
        // dataPoints may be modified between request and response, therefore
        // clone it
        const reqDataPoints = [...this.dataPoints]
        return m.request({
            method: 'PUT',
            url: "/~exgdata",
            body: {
                readPaths: reqDataPoints.map(dp => dp.addr)
            }
        }).then(function (resp) {
            // check response
            if (resp.readResults == undefined || !Array.isArray(resp.readResults)
                || resp.readResults.length !== reqDataPoints.length) {
                return Promise.reject({ message: "Ungültige Antwort für ExgData-Dienst!" })
            }
            // check result of each data point read
            reqDataPoints.forEach((dp, idx) => {
                const res = resp.readResults[idx]
                dp.changed = false
                if (res.error != undefined) {
                    // got an error
                    dp.pv = null
                    dp.last = null
                    // show VEAP error
                    dp.error = errorToString({
                        code: res.error.code,
                        response: { message: res.error.message }
                    })
                } else if (res.pv != undefined) {
                    // got a PV
                    dp.pv = {
                        ts: res.pv.ts ? new Date(res.pv.ts) : null,
                        v: res.pv.v,
                        s: res.pv.s,
                    }
                    dp.error = null
                    // detect changed PV
                    var cur = JSON.stringify(dp.pv.v)
                    if (dp.last != null && cur !== dp.last) {
                        dp.changed = true
                    }
                    dp.last = cur
                } else {
                    // no PV and no error
                    dp.pv = null
                    dp.last = null
                    dp.error = "Ungültige Teilantwort für ExgData-Dienst!"
                }
            })
        }).catch(function (err) {
            // service failed
            const error = errorToString(err)
            // propagate error to all data points
            reqDataPoints.forEach(dp => {
                dp.pv = null
                dp.last = null
                dp.changed = false
                dp.error = error
            });
        })
    }
}

// Watcher component
function Watcher() {
    // update rate [ms]
    var cycle = 400
    // set process value address
    var setPVAddr
    // timer ID
    var timer

    function update() {
        WatchList.update().then(() => {
            timer = setTimeout(update, cycle)
        })
    }

    // views a data point row
    function viewRow(dp) {
        return m("tr", { key: dp.addr },
            m("td", dp.name),
            m("td", dp.addr),
            m(StaticProcessValue, { pv: dp.pv, changed: dp.changed, error: dp.error }),
            m("td",
                m("button.btn.btn-sm",
                    { onclick: function () { WatchList.remove(dp.addr) } },
                    m("i.icon.icon-delete")
                ),
                m("button.btn.btn-sm.ml-2",
                    { onclick: function () { setPVAddr = dp.addr } },
                    m("i.icon.icon-edit")
                )
            )
        )
    }

    // mithril component
    return {
        oninit: function (vnode) {
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
        view: function (vnode) {
            return [
                m("h1", "Überwachung"),
                m("p", "Datenpunkte können über den Navigator zur Überwachung hinzugefügt werden."),
                m("table.table",
                    m("thead",
                        m("tr",
                            m("th", "Name"),
                            m("th", "Adresse"),
                            m("th", "Wert"),
                            m("th", "Zeitstempel"),
                            m("th", "Qualität"),
                            m("th")
                        )
                    ),
                    m("tbody",
                        WatchList.dataPoints.map(function (dp) { return viewRow(dp) })
                    )
                ),
                setPVAddr != null &&
                m(PVModal, { addr: setPVAddr, onclose: function () { setPVAddr = null } })
            ]
        }
    }
}