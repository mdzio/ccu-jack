#!/usr/bin/env tclsh
puts -nonewline "Content-Type: text/html; charset=utf-8\r\n\r\n"
puts {<!doctype html>
<html><head>
    <script>
        document.write('<meta http-equiv="refresh" content="0; url=')
        document.write('https://' + window.location.hostname + ':2122/ui')
        document.write('">')
    </script>
</head></html>
}
