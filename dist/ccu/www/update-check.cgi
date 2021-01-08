#!/usr/bin/env tclsh
set downloadCmd [regexp {\mcmd=download\M} $env(QUERY_STRING)]
if {$downloadCmd} {
  # forward to download page
  puts -nonewline "Content-Type: text/html; charset=utf-8\r\n\r\n"  
  puts "<html><head><meta http-equiv='refresh' content='0; url=https://github.com/mdzio/ccu-jack/releases' /></head></html>"
} else {
  # retrieve version of latest release
  set infoUrl https://api.github.com/repos/mdzio/ccu-jack/releases/latest
  set infoError [catch {
    set info [exec wget -q -O- --no-check-certificate $infoUrl]
    set found [regexp {\"tag_name\"\s*:\s*\"v([^\"]*)\"} $info -> version]
    if {!$found} error
  }]
  puts -nonewline "Content-Type: text/plain; charset=utf-8\r\n\r\n"
  if {$infoError} {
    puts "N/A"
  } else {
    puts $version
  }
}
