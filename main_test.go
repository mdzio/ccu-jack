package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdzio/ccu-jack/rtcfg"
)

func TestDebmaticCertificateHostname(t *testing.T) {
	config, err := os.ReadFile(filepath.Join("dist", "debmatic", "ccu-jack.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	var initial rtcfg.Config
	if err := json.Unmarshal(config, &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Host.Name != "" {
		t.Fatalf("Debmatic default host name must be empty, got %q", initial.Host.Name)
	}

	dir := t.TempDir()
	store.FileName = filepath.Join(dir, "ccu-jack.cfg")
	if err := os.WriteFile(store.FileName, config, 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Read(); err != nil {
		t.Fatal(err)
	}
	store.Close()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	if store.Config.Host.Name != hostname {
		t.Fatalf("host name = %q, want %q", store.Config.Host.Name, hostname)
	}
	certConfig := &store.Config.Certificates
	certConfig.CACertFile = filepath.Join(dir, "cacert.pem")
	certConfig.CAKeyFile = filepath.Join(dir, "cacert.key")
	certConfig.ServerCertFile = filepath.Join(dir, "svrcert.pem")
	certConfig.ServerKeyFile = filepath.Join(dir, "svrcert.key")
	if err := certificates(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(certConfig.ServerCertFile)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("server certificate is not PEM encoded")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.VerifyHostname(hostname); err != nil {
		t.Fatalf("server certificate does not match hostname %q: %v", hostname, err)
	}
}
