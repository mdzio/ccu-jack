package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mdzio/go-lib/logging"
	"github.com/mdzio/go-lib/util/releng"
)

// build configuration
const (
	appName    = "ccu-jack"
	appVersion = "0.5.0"
	appPkg     = "github.com/mdzio/ccu-jack"
	ldFlags    = "-s -w -X main.appVersion=" + appVersion
	buildDir   = ".."
)

var (
	targetSystems = []string{
		"win",
		"ccu2",
		"ccu3-rp0+1",
		"ccu3-rp2+3",
		"linux",
		"darwin",
	}

	goSpecs = map[string]releng.GoSpec{
		"ccu2":       {OS: "linux", Arch: "arm", Arm: "5", LDFlags: ldFlags},
		"ccu3-rp0+1": {OS: "linux", Arch: "arm", Arm: "6", LDFlags: ldFlags},
		"ccu3-rp2+3": {OS: "linux", Arch: "arm", Arm: "7", LDFlags: ldFlags},
		"win":        {OS: "windows", Arch: "amd64", LDFlags: ldFlags},
		"linux":      {OS: "linux", Arch: "amd64", LDFlags: ldFlags},
		"darwin":     {OS: "darwin", Arch: "amd64", LDFlags: ldFlags},
	}

	files = []releng.CopySpec{
		{Inc: "README.md"},
		{Inc: "LICENSE.txt"},
		{Inc: "build/tmp/VERSION"},
		{Inc: "wd/webui/*", DstDir: "webui"},
		{Inc: "wd/webui/ext/*", DstDir: "webui/ext"},
	}

	ccuFiles = []releng.CopySpec{
		{Inc: "dist/ccu/update_script", Exe: true},
		{Inc: "README.md", DstDir: "addon"},
		{Inc: "LICENSE.txt", DstDir: "addon"},
		{Inc: "build/tmp/VERSION", DstDir: "addon"},
		{Inc: "dist/ccu/addon/update_hm_addons.tcl", DstDir: "addon", Exe: true},
		{Inc: "wd/webui/*", DstDir: "addon/webui"},
		{Inc: "wd/webui/ext/*", DstDir: "addon/webui/ext"},
		{Inc: "dist/ccu/rc.d/ccu-jack", DstDir: "rc.d", Exe: true},
		{Inc: "dist/ccu/etc/monit-ccu-jack.cfg", DstDir: "etc"},
		{Inc: "dist/ccu/www/config.cgi", DstDir: "www", Exe: true},
		{Inc: "dist/ccu/www/update-check.cgi", DstDir: "www", Exe: true},
	}
)

// build commands
func build() {
	log.Info("Building application: ", appName)
	log.Info("Version: ", appVersion)
	log.Debug("Changing to build dir: ", buildDir)
	releng.Must(os.Chdir(buildDir))
	log.Info("Build dir: ", releng.Getwd())
	releng.RequireFiles([]string{"README.md"})
	releng.Mkdir("build/tmp")
	releng.WriteFile("build/tmp/VERSION", []byte(appVersion))

	for _, id := range targetSystems {
		goSpec, ok := goSpecs[id]
		if !ok {
			releng.Must(fmt.Errorf("Missing Go build specification: %s", id))
		}

		// build binary
		binFile := "build/tmp/" + id + "/" + appName
		if goSpec.OS == "windows" {
			binFile += ".exe"
		}
		releng.BuildGo(appPkg, binFile, goSpec)

		// build archive
		var allFiles []releng.CopySpec
		if strings.HasPrefix(id, "ccu") {
			allFiles = ccuFiles
			allFiles = append(allFiles, releng.CopySpec{Inc: binFile, DstDir: "addon", Exe: true})
		} else {
			allFiles = files
			allFiles = append(allFiles, releng.CopySpec{Inc: binFile, Exe: true})
		}
		var arcExt string
		if goSpec.OS == "windows" {
			arcExt = ".zip"
		} else {
			arcExt = ".tar.gz"
		}
		releng.Archive(appName+"-"+id+"-"+appVersion+arcExt, allFiles)
	}
}

// launcher
const (
	logLevel = logging.DebugLevel
	logFlags = logging.LevelFlag
)

var log = logging.Get("releng")

func main() {
	logging.SetLevel(logLevel)
	logging.SetFlags(logFlags)
	build()
	os.Exit(0)
}
