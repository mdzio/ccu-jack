/*
cleanPath returns the shortest path name equivalent to path by purely lexical
processing:
1. Replace multiple slashes with a single slash.
2. Eliminate each . path name element (the current directory).
3. Eliminate each inner .. path name element (the parent directory) along with
   the non-.. element that precedes it.
4. Eliminate .. elements that begin a rooted path: that is, replace "/.." by "/"
   at the beginning of a path.
(Inspired by the function path.Clean of Go.)
*/
function cleanPath(path) {
    var abs = false
    if (path.startsWith("/")) {
        path = path.substring(1)
        abs = true
    }
    var stack = [], parts = path.split("/")
    for (var i = 0; i < parts.length; i++) {
        var p = parts[i]
        // remove adjacent slashes and "."
        if (p == "" || p == ".") {
            continue
            // go always up on absolute paths, on relative paths only if the last
            // element is not ".."
        } else if (p == ".." && (abs || (stack.length > 0 && stack[stack.length - 1] != ".."))) {
            stack.pop()
            // otherwise append element
        } else {
            stack.push(p)
        }
    }
    path = stack.join("/")
    if (abs) {
        // rebuild absolute path
        return "/" + path
    } else {
        // return empty path as current element
        return path ? path : "."
    }
}

// resolvePath either resolves a relative path against a base path or returns
// the path unchanged if it is absolute.
function resolvePath(base, path) {
    if (!path.startsWith("/")) {
        // resolve relative path
        return cleanPath(base + "/" + path)
    }
    return path
}

// toPrettyString converts a value to a human readable string (german locale).
function toPrettyString(v) {
    switch (true) {
        // special values
        case v === null:
        case v === undefined:
            return ""
        // primitive types
        case typeof v == "boolean":
            return v ? "1" : "0"
        case typeof v == "number":
            // use scientific notation for very huge or small numbers
            var av = Math.abs(v)
            if (av > 10e10 || (av > 0 && av < 10e-10)) {
                return v.toExponential().replace(".", ",")
            } else {
                return v.toLocaleString("de-DE")
            }
        case typeof v == "string":
            return v
        // specific objects
        case Array.isArray(v):
            var ps = v.map(function (vp) {
                return toPrettyString(vp)
            })
            return "(" + ps.join(", ") + ")"
        case v instanceof Date:
            return v.toLocaleString("de-DE")
        // all other objects
        case typeof v == "object":
            var ps = Object.keys(v).map(function (k) {
                return k + "=" + toPrettyString(v[k])
            })
            return "(" + ps.join(", ") + ")"
        default:
            throw new Error("Unsupported type: " + v.constructor.name)
    }
}

// stateToString converts a VEAP state to a human readable string.
function stateToString(state) {
    switch (true) {
        case state === null || state === undefined:
            return ""
        case state >= 0 && state <= 99:
            return "GOOD (" + state + ")"
        case state >= 100 && state <= 199:
            return "UNCERTAIN (" + state + ")"
        case state >= 200 && state <= 299:
            return "BAD (" + state + ")"
        default:
            return "? (" + state + ")"
    }
}

// veapStatusToString converts a VEAP status to a human readable string.
function veapStatusToString(status) {
    switch (true) {
        case status === null || status === undefined:
            return ""
        case status == 200:
            return "OK (200)"
        case status == 201:
            return "OK, Objekt angelegt (201)"
        case status == 400:
            return "Ungültige Anfrage, HTTP-Protokollfehler (400)"
        case status == 401:
            return "Authentifizierung nötig (401)"
        case status == 403:
            return "Zugriffsrechte fehlen (403)"
        case status == 404:
            return "Objekt/Dienst nicht vorhanden (404)"
        case status == 405:
            return "Methode nicht erlaubt (405)"
        case status == 422:
            return "Anfrage entspricht nicht dem VEAP-Protokoll (422)"
        case status == 500:
            return "Unerwarteter Fehler im Server (500)"
        default:
            return "? (" + status + ")"
    }
}

// errorToString converts an error to a pretty string.
function errorToString(err) {
    // HTTP response?
    if (err.code !== undefined) {
        let detail = ""
        if (err.response && err.response.message) {
            detail = " (" + err.response.message + ")"
        }
        if (err.code === 0) {
            return "Anfrage an den VEAP-Server ist fehlgeschlagen." + detail
        } else {
            return "VEAP-Status: " + veapStatusToString(err.code) + detail
        }
    }
    // javascript error?
    if (err.message != null && err.message !== "null") {
        return err.message
    }
    if (err.name && err.name != "Error") {
        return err.name
    }
    // string?
    if (typeof err == "string") {
        return err
    }
    return "Unbekannter Fehler"
}

// parses a string as number (german locale)
function stringToNumber(str) {
    // test format
    if (!/^-?((\d{4,})|(\d{1,3}(\.\d{3})*))(,\d+)?$/.test(str)) {
        return null;
    }
    // remove group separators, replace decimal separator and parse
    var val = Number.parseFloat(str.replace(/\./g, '').replace(',', '.'));
    if (Number.isNaN(val)) {
        return null
    } else {
        return val
    }
}

// converts a number to string (german local)
function numberToString(val, options) {
    if (val == null) {
        return '';
    } else {
        return val.toLocaleString('de-DE', options);
    }
}

