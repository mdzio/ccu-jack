// Maintenance component
function Maintenance() {
    var errorMessage, successMessage

    // views an error
    function viewError() {
        return m(".toast.toast-error.my-2",
            m("h4", "Fehler"),
            m("p", "Beschreibung: " + errorMessage),
        )
    }

    // views a success
    function viewSuccess() {
        return m(".toast.toast-success.my-2",
            m("p", successMessage),
        )
    }

    // start a refresh
    function refresh() {
        m.request("/~vendor/refresh/~pv", {
            method: "PUT",
            body: { v: true },
        }).then(function () {
            errorMessage = null
            successMessage = "Das Auffrischen wurde gestartet."
        }).catch(function (err) {
            errorMessage = errorToString(err)
            successMessage = null
        })
    }

    // mithril component
    return {
        view: function () {
            return [
                m("h1", "Wartung"),
                errorMessage != null && viewError(),
                successMessage != null && viewSuccess(),
                m("button.btn", { onclick: refresh }, "Auffrischen"),
                m("p", "Die Geräte-/Kanalnamen, Räume, Gewerke, Systemvariablen und " +
                    "Programme werden neu aus der CCU ausgelesen. (Hinweis: Dies erfolgt " +
                    "automatisch beim Start des CCU-Jacks und dann regelmäßig alle 30 Minuten.)"),
            ]
        }
    }
}
