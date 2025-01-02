package main

import (
	"fmt"
	"os"

	"github.com/mdzio/go-lib/releng"
	"github.com/mdzio/go-logging"
)

// build configuration
const (
	logLevel   = logging.InfoLevel
	appName    = "ccu-jack"
	appVersion = "2.10.0"
	appPkg     = "github.com/mdzio/ccu-jack"
	ldFlags    = "-s -w -X main.appVersion=" + appVersion
	buildDir   = ".."
)

var (
	// target systems to be built
	targetSystems = []string{
		"ccu3-rm-rp2+3",
		"rm-rp0+1",
		"rm-rp4",
		"vccu-x86",
		"vccu-x86_64",
		"win",
		"linux",
		"darwin",
	}

	// target system specifications
	sysSpecs = map[string]struct {
		addon  bool
		goSpec releng.GoSpec
	}{
		"rm-rp0+1":      {true, releng.GoSpec{OS: "linux", Arch: "arm", Arm: "6", LDFlags: ldFlags}},
		"ccu3-rm-rp2+3": {true, releng.GoSpec{OS: "linux", Arch: "arm", Arm: "7", LDFlags: ldFlags}},
		"rm-rp4":        {true, releng.GoSpec{OS: "linux", Arch: "arm64", LDFlags: ldFlags}},
		"vccu-x86":      {true, releng.GoSpec{OS: "linux", Arch: "386", LDFlags: ldFlags}},
		"vccu-x86_64":   {true, releng.GoSpec{OS: "linux", Arch: "amd64", LDFlags: ldFlags}},
		"win":           {false, releng.GoSpec{OS: "windows", Arch: "amd64", LDFlags: ldFlags}},
		"linux":         {false, releng.GoSpec{OS: "linux", Arch: "amd64", LDFlags: ldFlags}},
		"darwin":        {false, releng.GoSpec{OS: "darwin", Arch: "amd64", LDFlags: ldFlags}},
	}

	// files for non ccu target systems
	files = []releng.CopySpec{
		{Inc: "README.md"},
		{Inc: "LICENSE.txt"},
		{Inc: "dist/ccu-jack.cfg"},
		{Inc: "build/tmp/VERSION"},
		{Inc: "third-party-licenses/*", DstDir: "third-party-licenses"},
		{Inc: "wd/webui/*", DstDir: "webui"},
		{Inc: "wd/webui/ext/*", DstDir: "webui/ext"},
	}

	// files for ccu target systems
	addonFiles = []releng.CopySpec{
		{Inc: "dist/ccu/update_script", Exe: true},
		{Inc: "README.md", DstDir: "addon"},
		{Inc: "LICENSE.txt", DstDir: "addon"},
		{Inc: "dist/ccu/addon/ccu-jack-default.cfg", DstDir: "addon"},
		{Inc: "build/tmp/VERSION", DstDir: "addon"},
		{Inc: "dist/update_hm_addons.tcl", DstDir: "addon", Exe: true},
		{Inc: "third-party-licenses/*", DstDir: "addon/third-party-licenses"},
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
	releng.RequireFiles([]string{"README.md", "LICENSE.txt", "main.go"})
	releng.Mkdir("build/tmp")
	releng.WriteFile("build/tmp/VERSION", []byte(appVersion))

	for _, ts := range targetSystems {
		sysSpec, ok := sysSpecs[ts]
		if !ok {
			releng.Must(fmt.Errorf("Missing Go build specification: %s", ts))
		}

		// build binary
		binFile := "build/tmp/" + ts + "/" + appName
		if sysSpec.goSpec.OS == "windows" {
			binFile += ".exe"
		}
		releng.BuildGo(appPkg, binFile, sysSpec.goSpec)

		// build archive
		var allFiles []releng.CopySpec
		if sysSpec.addon {
			allFiles = addonFiles
			allFiles = append(allFiles, releng.CopySpec{Inc: binFile, DstDir: "addon", Exe: true})
		} else {
			allFiles = files
			allFiles = append(allFiles, releng.CopySpec{Inc: binFile, Exe: true})
		}
		var arcExt string
		if sysSpec.goSpec.OS == "windows" {
			arcExt = ".zip"
		} else {
			arcExt = ".tar.gz"
		}
		releng.Archive(appName+"-"+ts+"-"+appVersion+arcExt, allFiles)
	}
}

// launcher
const logFlags = logging.LevelFlag

var log = logging.Get("releng")

func main() {
	logging.SetLevel(logLevel)
	logging.SetFlags(logFlags)
	build()
	os.Exit(0)
}
