// Watcher data
var WatchList = {
    dataPoints: [],
    get length() {
        return this.dataPoints.length
    },
    add: function (name, addr) {
        this.dataPoints.push({
            name: name, addr: addr
        })
    },
    remove: function (addr) {
        this.dataPoints = this.dataPoints.filter(function (dp) {
            return dp.addr !== addr
        })
    },
    includes: function (addr) {
        return this.dataPoints.findIndex(function (dp) {
            return dp.addr === addr
        }) != -1
    }
}

// Watcher component
function Watcher() {

    // views a data point row
    function viewRow(dp) {
        return m("tr", { key: dp.addr },
            m("td", dp.name),
            m("td", dp.addr),
            m(ProcessValue, { addr: dp.addr }),
            m("td",
                { onclick: function () { WatchList.remove(dp.addr) } },
                m("button.btn.btn-sm", m("i.icon.icon-delete"))
            )
        )
    }

    // mithril component
    return {
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
                )
            ]
        }
    }
}