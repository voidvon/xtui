package config_test

import (
	"encoding/base64"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/curve25519"
	"xtui/config"
)

func TestLoadConfigFile(t *testing.T) {
	cfgPath := filepath.Join("..", "config.json")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("Failed to load config.json: %v", err)
	}

	if cfg.Server.Dest != "blog.xmgspace.me:443" {
		t.Errorf("Expected dest to be blog.xmgspace.me:443, got %s", cfg.Server.Dest)
	}

	if len(cfg.Server.ServerNames) == 0 || cfg.Server.ServerNames[0] != "blog.xmgspace.me" {
		t.Errorf("Expected server_names[0] to be blog.xmgspace.me, got %v", cfg.Server.ServerNames)
	}
}

func TestVerifyKeyPair(t *testing.T) {
	privB64 := "KCvtEPO5UFkSXYjgvqZgEvePxKcBeCXoa1NVC7NCemQ"
	pubB64 := "zPDAEUGQRg-TVFnwLeEgoXa3N785Zc4UJceq91H5qyM"

	privBytes, err := base64.RawURLEncoding.DecodeString(privB64)
	if err != nil {
		t.Fatalf("priv decode error: %v", err)
	}
	pubBytes, err := curve25519.X25519(privBytes, curve25519.Basepoint)
	if err != nil {
		t.Fatalf("curve25519 scalar mult error: %v", err)
	}
	derivedPub := base64.RawURLEncoding.EncodeToString(pubBytes)
	if pubB64 != derivedPub {
		t.Fatalf("KeyPair mismatch: expected %s, derived %s", pubB64, derivedPub)
	}
}
