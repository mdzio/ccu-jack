// Version component displays the version of the CCU-Jack.
function Version() {
    var version = "?"
    return {
        oninit: function () {
            m.request("/~vendor").then(function (info) {
                version = info.serverVersion
            }).catch(function () {
                version = "-"
            })
        },
        view: function () {
            return m("p.text-right", "V" + version)
        }
    }
}

// Layout component contains the base layout of the web app.
function Layout() {
    // title of watcher probably with badge
    function viewWatcherTitle() {
        if (WatchList.length) {
            return m("span.badge", { "data-badge": WatchList.length }, "Überwachung")
        } else {
            return "Überwachung"
        }
    }

    // mithril component
    return {
        view: function (vnode) {
            return m(".container",
                m(".columns",
                    m(".column.col-2.col-md-12",
                        m("h1.text-center.text-primary", "CCU-Jack"),
                        m("button.btn.btn-block.my-2", {
                            onclick: function () { m.route.set("/navigator:key...", { key: "/" }) }
                        }, "Navigator"),
                        m("button.btn.btn-block.my-2", {
                            onclick: function () { m.route.set("/watcher") }
                        }, viewWatcherTitle()),
                        m("button.btn.btn-block.my-2", {
                            onclick: function () { m.route.set("/virtdev") }
                        }, "Virtuelle Geräte"),
                        m("button.btn.btn-block.my-2", {
                            onclick: function () { m.route.set("/config") }
                        }, "Konfiguration"),
                        m("button.btn.btn-block.my-2", {
                            onclick: function () { m.route.set("/diagnostics") }
                        }, "Diagnose"),
                        m(Version)
                    ),
                    m(".column.col-10.col-md-12",
                        vnode.children
                    )
                )
            )
        }
    }
}

// routes to the pages of the web app
m.route(document.body, "/navigator", {
    "/navigator:addr...": {
        render: function (vnode) {
            var addr = vnode.attrs.addr
            if (!addr.startsWith("/")) {
                addr = "/" + addr
            }
            // use key to force oninit on address change
            return m(Layout, m(Navigator, { key: addr, addr: addr }))
        }
    },
    "/watcher": {
        render: function () {
            return m(Layout, m(Watcher))
        }
    },
    "/virtdev": {
        render: function () {
            return m(Layout, m(VirtualDevices))
        }
    },
    "/config": {
        render: function () {
            return m(Layout, m(Config))
        }
    },
    "/diagnostics": {
        render: function () {
            return m(Layout, m(Diagnostics))
        }
    }
})
