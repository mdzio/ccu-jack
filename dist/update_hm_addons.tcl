#!/usr/bin/env tclsh
# Copyright 2018-2019 MDZ info@ccu-historian.de
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# this script adds or removes an addon configuration

# parse arguments
foreach {n v} $argv {
    switch -- $n {
        -a { set id $v; set add true }
        -d { set id $v; set add false }
        -name { set name $v }
        -url { set url $v }
        -en { set descr(en) "<li>$v</li>" }
        -de { set descr(de) "<li>$v</li>" }
        default {
            puts stderr "error: invalid argument: $n"
            exit 1
        }
    }
}

# check arguments
set argError false
# id is always needed
if {![info exists id] || $id==""} {
    set argError true
} elseif {$add} {
    # for -a all remaining parameters are needed
    foreach n {name url descr(de) descr(en)} {
        if {![info exists $n]} {
            set argError true
            break
        }
    }
}
if {$argError} {
    puts stderr "usage: [file tail [info script]] \[OPTIONS]"
    puts stderr "options:"
    puts stderr "  add (or update) add-on:"
    puts stderr "    -a ID -url URL -name NAME -de DESCR(DE) -en DESCR(EN)"
    puts stderr "  delete add-on:"
    puts stderr "    -d ID"
    exit 1
}

# read config file
set cfgFileName /usr/local/etc/config/hm_addons.cfg
if {[file exists $cfgFileName]} {
  set file [open $cfgFileName r]
  array set config [read -nonewline $file] 
  close $file
} else {
  array set config {} 
}

# remove addon config, if present
if {[info exists config($id)]} { 
  unset config($id)
}

# add addon config, if requested
if {$add} {
  # build structure
  set cfg(ID) $id
  set cfg(CONFIG_NAME) $name
  set cfg(CONFIG_URL) $url
  set cfg(CONFIG_DESCRIPTION) [array get descr]
  # add structure
  set config($id) [array get cfg]
}

# write config file
set file [open $cfgFileName w]
puts -nonewline $file [array get config]
close $file
