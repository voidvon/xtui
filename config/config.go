package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tailscale/hujson"

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
	Dest        string   `json:"dest"`         // Reality destination, e.g. "www.kernel.org:443"
	ServerNames []string `json:"server_names"` // SNI list, e.g. ["www.kernel.org"]
	PrivateKey  string   `json:"private_key"`  // Reality private key (base64)
	PublicKey   string   `json:"public_key"`   // Reality public key (base64)
	ShortIds    []string `json:"short_ids"`    // Hex short IDs
	SpiderX     string   `json:"spider_x"`     // Reality spiderX, default "/"
	Fingerprint string   `json:"fingerprint"`  // uTLS fingerprint, default "chrome"
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
			Dest:        "www.kernel.org:443",
			ServerNames: []string{"www.kernel.org"},
			PrivateKey:  priv,
			PublicKey:   pub,
			ShortIds:    []string{shortId},
			SpiderX:     "/",
			Fingerprint: "chrome",
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

// LoadConfig loads configuration from path (supports JSON with comments), or creates a default one.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Migration check: if config.json not found, but xtui.json exists, migrate it
		if path == "config.json" {
			if oldData, errOld := os.ReadFile("xtui.json"); errOld == nil {
				_ = os.WriteFile("config.json", oldData, 0600)
				_ = os.Remove("xtui.json")
				data = oldData
				err = nil
			}
		}
		if data == nil {
			cfg, err := DefaultConfig()
			if err != nil {
				return nil, err
			}
			if err := SaveConfig(path, cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		}
	} else if err != nil {
		return nil, err
	}

	// Standardize JSON with comments (HuJSON / JSONC) and trailing commas
	stdJSON, err := hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("配置文件 JSON 语法解析失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(stdJSON, &cfg); err != nil {
		return nil, fmt.Errorf("配置文件结构解析失败: %w", err)
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
	if cfg.Server.Dest == "" {
		cfg.Server.Dest = "www.kernel.org:443"
	}
	if len(cfg.Server.ServerNames) == 0 {
		cfg.Server.ServerNames = []string{"www.kernel.org"}
	}
	if cfg.Server.SpiderX == "" {
		cfg.Server.SpiderX = "/"
	}
	if cfg.Server.Fingerprint == "" {
		cfg.Server.Fingerprint = "chrome"
	}
	return &cfg, nil
}

// MarshalJSONC formats Config into human-friendly JSON with helpful comments.
func MarshalJSONC(cfg *Config) []byte {
	var buf bytes.Buffer
	buf.WriteString("{\n")
	buf.WriteString("  // ==========================================\n")
	buf.WriteString("  //  XTUI 服务端与 Reality 核心网络配置\n")
	buf.WriteString("  // ==========================================\n")
	buf.WriteString("  \"server\": {\n")
	buf.WriteString(fmt.Sprintf("    \"port\": %d, // 服务监听端口 (推荐 443 或 4433 等)\n", cfg.Server.Port))
	buf.WriteString(fmt.Sprintf("    \"listen\": %q, // 监听绑定的网络地址\n", cfg.Server.Listen))
	buf.WriteString(fmt.Sprintf("    \"public_host\": %q, // 节点公网 IP 或域名 (生成客户端分享链接时使用)\n", cfg.Server.PublicHost))
	buf.WriteString(fmt.Sprintf("    \"dest\": %q, // Reality 伪装目标网站 (必须支持 TLS 1.3 / H2)\n", cfg.Server.Dest))

	snisJSON, _ := json.Marshal(cfg.Server.ServerNames)
	buf.WriteString(fmt.Sprintf("    \"server_names\": %s, // Reality 伪装域名 (SNI 列表)\n", string(snisJSON)))

	buf.WriteString(fmt.Sprintf("    \"private_key\": %q, // Reality 私钥 (服务端严格保密)\n", cfg.Server.PrivateKey))
	buf.WriteString(fmt.Sprintf("    \"public_key\": %q, // Reality 公钥 (用于客户端连接校验)\n", cfg.Server.PublicKey))

	sidsJSON, _ := json.Marshal(cfg.Server.ShortIds)
	buf.WriteString(fmt.Sprintf("    \"short_ids\": %s, // 8字节短 ID 列表\n", string(sidsJSON)))

	buf.WriteString(fmt.Sprintf("    \"spider_x\": %q, // 爬虫伪装路径 (默认 \"/\")\n", cfg.Server.SpiderX))
	buf.WriteString(fmt.Sprintf("    \"fingerprint\": %q // 客户端 uTLS 模拟指纹 (chrome/firefox/safari/ios)\n", cfg.Server.Fingerprint))
	buf.WriteString("  },\n\n")

	buf.WriteString("  // ==========================================\n")
	buf.WriteString("  //  客户端用户列表与独立流量记账\n")
	buf.WriteString("  // ==========================================\n")
	buf.WriteString("  \"users\": [\n")
	for i, u := range cfg.Users {
		buf.WriteString("    {\n")
		buf.WriteString(fmt.Sprintf("      \"id\": %q, // 用户唯一凭证 UUID (VLESS 密码)\n", u.ID))
		buf.WriteString(fmt.Sprintf("      \"email\": %q, // 用户标识名 / 流量统计账户\n", u.Email))
		buf.WriteString(fmt.Sprintf("      \"flow\": %q, // 流控模式 (推荐 xtls-rprx-vision)\n", u.Flow))
		buf.WriteString(fmt.Sprintf("      \"up_bytes\": %d, // 累计已用上传流量 (字节 Byte)\n", u.UpBytes))
		buf.WriteString(fmt.Sprintf("      \"down_bytes\": %d, // 累计已用下载流量 (字节 Byte)\n", u.DownBytes))
		buf.WriteString(fmt.Sprintf("      \"limit_bytes\": %d, // 流量上限配额 (字节 Byte, 0 为不限制)\n", u.LimitBytes))
		buf.WriteString(fmt.Sprintf("      \"expiry_time\": %d, // 到期时间戳 (Unix时间戳, 0 为永久有效)\n", u.ExpiryTime))
		buf.WriteString(fmt.Sprintf("      \"enabled\": %t, // 是否启用该用户 (true/false)\n", u.Enabled))
		buf.WriteString(fmt.Sprintf("      \"created_at\": %d // 创建时间戳\n", u.CreatedAt))
		if i == len(cfg.Users)-1 {
			buf.WriteString("    }\n")
		} else {
			buf.WriteString("    },\n")
		}
	}
	buf.WriteString("  ]\n")
	buf.WriteString("}\n")
	return buf.Bytes()
}

// SaveConfig writes the configuration safely as human-readable JSON with comments.
func SaveConfig(path string, cfg *Config) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data := MarshalJSONC(cfg)

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, path)
}
