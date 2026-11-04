package config_test

import (
	"path/filepath"
	"testing"

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
