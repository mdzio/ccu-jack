// Diagnostics component
function Diagnostics() {
    var indicators
    var errorMessage

    function fetch() {
        m.request("/~vendor/diagnostics/~pv").then(function (resp) {
            if (resp.v !== undefined) {
                indicators = resp.v
                errorMessage = null
            } else {
                errorMessage = "Ungültiges JSON-Objekt als Diagnose!"
            }
        }).catch(function (e) {
            errorMessage = errorToString(e)
        })
    }

    // views a log message
    function viewMsg(msg) {
        var cls
        switch (msg[1]) {
            case "ERROR": cls = "bg-error"; break
            case "WARNING": cls = "bg-warning"; break
            case "INFO": break
            case "DEBUG": cls = "text-gray"; break
            case "TRACE": cls = "text-gray"; break
        }
        return m("tr", { class: cls },
            m("td[style=padding:1px]", msg[0]),
            m("td[style=padding:1px]", msg[1]),
            m("td[style=padding:1px]", msg[2]),
            m("td[style=padding:1px]", msg[3])
        )
    }

    // views the log message table
    function viewMsgTable() {
        return [
            m("h2", "Log-Meldungen"),
            m("table.table",
                m("thead",
                    m("tr",
                        m("th[style=padding:1px]", "Zeitstempel"),
                        m("th[style=padding:1px]", "Dringlichkeit"),
                        m("th[style=padding:1px]", "Modul"),
                        m("th[style=padding:1px]", "Meldung")
                    )
                ),
                m("tbody",
                    indicators.Log.map(viewMsg)
                )
            )
        ]
    }

    // views the indicators
    function viewIndicators() {
        return [
            viewMsgTable()
        ]
    }

    // views a possible error
    function viewErr() {
        return m(".toast.toast-error.my-2",
            m("h4", "Fehler"),
            m("p", "Beschreibung: " + errorMessage),
        )
    }

    // mithril component
    return {
        oninit: function (vnode) {
            fetch()
        },
        view: function (vnode) {
            return [
                m("h1", "Diagnose"),
                m(".float-right", m("button.btn.my-2", { onclick: fetch }, "Aktualisieren")),
                indicators != null && viewIndicators(),
                errorMessage != null && viewErr()
            ]
        }
    }
}