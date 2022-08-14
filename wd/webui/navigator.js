// VEAP navigator component. The component must be recreated, if the address is
// changed. Attributes: addr=VEAP object address
function Navigator() {
    // address set by route parameter, address in input field
    var routeAddr, editAddr
    // current VEAP object or error
    var vobj, error
    // VEAP object has a process value?
    var pv = false
    // show modal for PV?
    var showPVModal = false

    // queries the VEAP object properties
    function init(addr) {
        routeAddr = addr
        editAddr = addr

        // read VEAP properties
        m.request(routeAddr).then(function (resp) {
            vobj = resp

            // has the VEAP object a process value?
            if (vobj["~links"]) {
                var lnk = vobj["~links"].find(function (l) {
                    return l.href.endsWith("~pv")
                })
                // PV found?
                if (lnk) {
                    pv = true
                }
            }
        }).catch(function (err) {
            error = err
        })
    }

    // switch route to new address
    function routeTo(addr) {
        m.route.set("/navigator:addr...", { addr: resolvePath(routeAddr, addr) })
    }

    // views the editable address bar
    function viewAddrBar() {
        return [
            m("form.input-group",
                { onsubmit: function (e) { e.preventDefault(); routeTo(editAddr) } },
                m("span.input-group-addon", "Adresse"),
                m("input.form-input[type=text]", {
                    value: editAddr,
                    oninput: function (e) {
                        editAddr = e.target.value
                        // path must start with /
                        if (!editAddr.startsWith("/")) editAddr = "/" + editAddr
                    }
                }),
                m("button.btn.input-group-btn", "Übernehmen")
            ),
        ]
    }

    // views a possible error
    function viewErr(err) {
        return m(".toast.toast-error.my-2",
            m("h4", "Fehler"),
            m("p", "Beschreibung: " + errorToString(err)),
        )
    }

    // views the properties of the VEAP object as table
    function viewProps() {
        // filter and sort additional properties
        var props = Object.keys(vobj).filter(function (key) {
            // title and description are displayed separately. ~link is a
            // reserved VEAP property. identifier is not needed.
            return key != "identifier" && key != "title" && key != "description" && key != "~links"
        }).sort(function (a, b) {
            return a.toLowerCase().localeCompare(b.toLowerCase())
        })

        return [
            m("h2.pt-2", "Eigenschaften"),
            m("table.table",
                m("thead",
                    m("tr",
                        m("th", "Name"),
                        m("th", "Wert"),
                    ),
                ),
                m("tbody",
                    m("tr",
                        m("td", m("strong", "Titel")),
                        m("td", toPrettyString(vobj.title)),
                    ),
                    m("tr",
                        m("td", m("strong", "Beschreibung")),
                        m("td", toPrettyString(vobj.description)),
                    ),
                    props.map(function (key) {
                        return m("tr",
                            m("td", key),
                            m("td", toPrettyString(vobj[key])),
                        )
                    }),
                ),
            ),
        ]
    }

    // views the links of the VEAP object as table
    function viewLinks() {
        // get links
        var links = vobj["~links"] ? vobj["~links"] : []

        // filter links
        links = links.filter(function (l) {
            // remove service links
            return !l.href.endsWith("~pv") && !l.href.endsWith("~hist")
                && !l.href.endsWith("~exgdata") && !l.href.endsWith("~query")
        })

        // sort links
        var links = links.sort(function (a, b) {
            // parent link first
            if (a.href === "..") return -1
            if (b.href === "..") return 1
            // sort by relation type
            if (a.rel != null && b.rel != null) {
                var r = a.rel.localeCompare(b.rel)
                if (r != 0) return r
            }
            // sort by title
            if (a.title != null && b.title != null) {
                return a.title.toLowerCase().localeCompare(b.title.toLowerCase())
            }
            if (a.title) return -1
            if (b.title) return 1
            // sort by href
            return a.href.toLowerCase().localeCompare(b.href.toLowerCase())
        })

        return [
            m("h2.pt-2", "Verweise"),
            m("table.table",
                m("thead",
                    m("tr",
                        m("th", "Titel"),
                        m("th", "Typ"),
                        m("th", "Adresse"),
                        m("th"),
                    ),
                ),
                m("tbody",
                    links.map(function (l) {
                        return m("tr",
                            { onclick: function () { routeTo(l.href) } },
                            m("td", toPrettyString(l.title)),
                            m("td", toPrettyString(l.rel)),
                            m("td", toPrettyString(l.href)),
                            m("td", m("button.btn.btn-sm",
                                l.href == ".." ? m("i.icon.icon-upward") : m("i.icon.icon-forward")
                            )),
                        )
                    })
                ),
            ),
        ]
    }

    // views the PV of a VEAP variable
    function viewPV() {
        if (pv) {
            var watched = WatchList.includes(routeAddr)
            return [
                m("h2.pt-2", "Aktueller Wert"),
                m("table.table",
                    m("thead",
                        m("tr",
                            m("th", "Wert"),
                            m("th", "Zeitstempel"),
                            m("th", "Qualität"),
                            m("th"),
                        ),
                    ),
                    m("tbody",
                        m("tr",
                            m(ProcessValue, { addr: routeAddr }),
                            m("td",
                                m("button.btn.btn-sm",
                                    {
                                        onclick: function () {
                                            if (watched) {
                                                WatchList.remove(routeAddr)
                                            } else {
                                                WatchList.add(vobj.title, routeAddr)
                                            }
                                        }
                                    },
                                    watched ? m("i.icon.icon-minus") : m("i.icon.icon-plus"),
                                ),
                                m("button.btn.btn-sm.ml-2", { onclick: function () { showPVModal = true } },
                                    m("i.icon.icon-edit")
                                ),
                            ),
                        ),
                    ),
                ),
                showPVModal &&
                m(PVModal, { addr: routeAddr, onclose: function () { showPVModal = false } })
            ]
        }
        return null
    }

    // mithril component
    return {
        oninit: function (vnode) {
            init(vnode.attrs.addr)
        },
        view: function () {
            return [
                m("h1", "Navigator"),
                viewAddrBar(),
                m("h2.mt-2", (vobj && vobj.title) || routeAddr),
                vobj ? m(".columns.",
                    m(".column.col-6.col-xl-12",
                        viewProps(),
                    ),
                    m(".column.col-6.col-xl-12",
                        viewPV(),
                        viewLinks(),
                    )
                ) : null,
                error ? viewErr(error) : null,
            ]
        }
    }
}
