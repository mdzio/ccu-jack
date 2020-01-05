#!/usr/bin/env tclsh
set infoUrl https://api.github.com/repos/mdzio/ccu-jack/releases/latest
set infoError [catch {
  set info [exec wget -q -O- --no-check-certificate $infoUrl]
  set found [regexp {\"tag_name\"\s*:\s*\"([^\"]*)\"} $info -> version]
  if {!$found} error
  set found [regexp {\"browser_download_url\"\s*:\s*\"([^\"]*/ccu-jack-[^\"]+\.tar\.gz)\"} $info -> downloadUrl]
  if {!$found} error
}]
set downloadCmd [regexp {\mcmd=download\M} $env(QUERY_STRING)]
if {$downloadCmd} {
  puts -nonewline "Content-Type: text/html; charset=utf-8\r\n\r\n"  
  if {$infoError} {
    puts "<html><body>Error determining download link!</body></html>"
  } else {
    puts "<html><head><meta http-equiv='refresh' content='0; url=$downloadUrl' /></head></html>"
  }
} else {
  puts -nonewline "Content-Type: text/plain; charset=utf-8\r\n\r\n"
  if {$infoError} {
    puts "N/A"
  } else {
    puts $version
  }
}
