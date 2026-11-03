package config

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xtui/pkg/keys"
)

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Flow       string `json:"flow"`        // default "xtls-rprx-vision"
	UpBytes    int64  `json:"up_bytes"`    // Total uploaded bytes
	DownBytes  int64  `json:"down_bytes"`  // Total downloaded bytes
	LimitBytes int64  `json:"limit_bytes"` // 0 = unlimited
	ExpiryTime int64  `json:"expiry_time"` // 0 = permanent, Unix timestamp
	Enabled    bool   `json:"enabled"`
	CreatedAt  int64  `json:"created_at"`
}

type ServerConfig struct {
	Port        int      `json:"port"`         // Listening port (default 4433)
	Listen      string   `json:"listen"`       // Bind IP (default "0.0.0.0")
	PublicHost  string   `json:"public_host"`  // Public IP or Domain for client config
	Dest        string   `json:"dest"`         // Reality destination, e.g. "gateway.icloud.com:443"
	ServerNames []string `json:"server_names"` // SNI list, e.g. ["gateway.icloud.com"]
	PrivateKey  string   `json:"private_key"`  // Reality private key (base64)
	PublicKey   string   `json:"public_key"`   // Reality public key (base64)
	ShortIds    []string `json:"short_ids"`    // Hex short IDs
}

type Config struct {
	Server ServerConfig `json:"server"`
	Users  []*User      `json:"users"`
	mu     sync.RWMutex `json:"-"`
}

// FetchPublicIP queries public IP services with a short timeout.
func FetchPublicIP() string {
	client := &http.Client{Timeout: 2 * time.Second}
	endpoints := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, ep := range endpoints {
		resp, err := client.Get(ep)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				ip := strings.TrimSpace(string(body))
				if ip != "" {
					return ip
				}
			}
		}
	}
	return "127.0.0.1"
}

// DefaultConfig generates a ready-to-run configuration with realistic Reality settings.
func DefaultConfig() (*Config, error) {
	priv, pub, err := keys.GenerateX25519()
	if err != nil {
		return nil, err
	}
	shortId := keys.GenerateShortId()
	publicIP := FetchPublicIP()

	cfg := &Config{
		Server: ServerConfig{
			Port:        4433,
			Listen:      "0.0.0.0",
			PublicHost:  publicIP,
			Dest:        "gateway.icloud.com:443",
			ServerNames: []string{"gateway.icloud.com"},
			PrivateKey:  priv,
			PublicKey:   pub,
			ShortIds:    []string{shortId},
		},
		Users: []*User{
			{
				ID:         keys.GenerateUUID(),
				Email:      "user_1",
				Flow:       "xtls-rprx-vision",
				UpBytes:    0,
				DownBytes:  0,
				LimitBytes: 0, // unlimited
				ExpiryTime: 0, // permanent
				Enabled:    true,
				CreatedAt:  time.Now().Unix(),
			},
		},
	}
	return cfg, nil
}

// LoadConfig loads configuration from path, or creates a default one if not exists.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg, err := DefaultConfig()
		if err != nil {
			return nil, err
		}
		if err := SaveConfig(path, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	} else if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Fallback checks
	if cfg.Server.Port <= 0 {
		cfg.Server.Port = 4433
	}
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = "0.0.0.0"
	}
	if cfg.Server.PublicHost == "" {
		cfg.Server.PublicHost = "127.0.0.1"
	}
	return &cfg, nil
}

// SaveConfig writes the configuration safely as indented JSON.
func SaveConfig(path string, cfg *Config) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, path)
}
