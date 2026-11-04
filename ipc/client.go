package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"

	"xtui/config"
	"xtui/core"
)

type Client struct {
	sockPath   string
	httpClient *http.Client

	mu         sync.RWMutex
	lastStatus *StatusResponse
	stopChan   chan struct{}
	onUpdate   func()
}

func IsDaemonRunning(sockPath string) bool {
	conn, err := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func NewClient(sockPath string) (*Client, error) {
	c := &Client{
		sockPath: sockPath,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", sockPath)
				},
			},
			Timeout: 3 * time.Second,
		},
		stopChan: make(chan struct{}),
	}

	// Initial fetch
	if err := c.refresh(); err != nil {
		return nil, fmt.Errorf("failed to connect to daemon at %s: %w", sockPath, err)
	}

	// Start background polling loop
	go c.pollLoop()

	return c, nil
}

func (c *Client) Close() {
	close(c.stopChan)
}

func (c *Client) pollLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := c.refresh(); err == nil {
				c.mu.RLock()
				cb := c.onUpdate
				c.mu.RUnlock()
				if cb != nil {
					cb()
				}
			}
		}
	}
}

func (c *Client) refresh() error {
	resp, err := c.httpClient.Get("http://unix/api/status")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var st StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return err
	}

	c.mu.Lock()
	c.lastStatus = &st
	c.mu.Unlock()
	return nil
}

func (c *Client) post(endpoint string, reqBody interface{}, respBody interface{}) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post("http://unix"+endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if respBody != nil {
		return json.NewDecoder(resp.Body).Decode(respBody)
	}

	var actResp ActionResponse
	_ = json.NewDecoder(resp.Body).Decode(&actResp)
	if !actResp.Success && actResp.Error != "" {
		return fmt.Errorf("%s", actResp.Error)
	}
	return nil
}

// Implement core.Backend interface

func (c *Client) GetConfig() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil {
		return &config.Config{}
	}
	return c.lastStatus.Config
}

func (c *Client) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil {
		return false
	}
	return c.lastStatus.Running
}

func (c *Client) GetStartTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil {
		return time.Now()
	}
	return c.lastStatus.StartTime
}

func (c *Client) GetUserStats(email string) core.UserStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil || c.lastStatus.Stats == nil {
		return core.UserStats{}
	}
	return c.lastStatus.Stats[email]
}

func (c *Client) SetOnUpdate(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onUpdate = fn
}

func (c *Client) AddUser(email string, limitGB float64, validDays int, customUUID, flow string) (*config.User, error) {
	req := AddUserRequest{
		Email:      email,
		LimitGB:    limitGB,
		ValidDays:  validDays,
		CustomUUID: customUUID,
		Flow:       flow,
	}
	var res struct {
		Success bool         `json:"success"`
		Error   string       `json:"error"`
		User    *config.User `json:"user"`
	}
	if err := c.post("/api/user/add", req, &res); err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, fmt.Errorf("%s", res.Error)
	}
	_ = c.refresh()
	return res.User, nil
}

func (c *Client) EditUser(index int, email, uuidStr, flow string, limitGB float64, validDays int, enabled bool) error {
	req := EditUserRequest{
		Index:     index,
		Email:     email,
		UUID:      uuidStr,
		Flow:      flow,
		LimitGB:   limitGB,
		ValidDays: validDays,
		Enabled:   enabled,
	}
	if err := c.post("/api/user/edit", req, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) DeleteUser(index int) error {
	if err := c.post("/api/user/delete", IndexRequest{Index: index}, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) ToggleUser(index int) error {
	if err := c.post("/api/user/toggle", IndexRequest{Index: index}, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) ResetUserTraffic(index int) error {
	if err := c.post("/api/user/reset", IndexRequest{Index: index}, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) ResetAllTraffic() error {
	if err := c.post("/api/user/reset-all", map[string]interface{}{}, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) UpdateReality(dest string, serverNames []string, privKey, pubKey string, shortIds []string, spiderX string) error {
	req := UpdateRealityRequest{
		Dest:        dest,
		ServerNames: serverNames,
		PrivateKey:  privKey,
		PublicKey:   pubKey,
		ShortIds:    shortIds,
		SpiderX:     spiderX,
	}
	if err := c.post("/api/reality", req, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) RegenerateRealityKeys() (priv, pub string, err error) {
	var res KeyPairResponse
	if err := c.post("/api/reality/regen", map[string]interface{}{}, &res); err != nil {
		return "", "", err
	}
	_ = c.refresh()
	return res.PrivateKey, res.PublicKey, nil
}

func (c *Client) UpdateVLESS(port int, listen, publicHost, fingerprint string) error {
	req := UpdateVLESSRequest{
		Port:        port,
		Listen:      listen,
		PublicHost:  publicHost,
		Fingerprint: fingerprint,
	}
	if err := c.post("/api/vless", req, nil); err != nil {
		return err
	}
	_ = c.refresh()
	return nil
}

func (c *Client) Restart() error {
	return c.post("/api/core/restart", map[string]interface{}{}, nil)
}

func (c *Client) Stop() error {
	return c.post("/api/core/stop", map[string]interface{}{}, nil)
}

func (c *Client) Start() error {
	return c.post("/api/core/start", map[string]interface{}{}, nil)
}

func (c *Client) GenerateVlessLink(u *config.User) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cfg := c.lastStatus.Config
	sni := "www.kernel.org"
	if len(cfg.Server.ServerNames) > 0 {
		sni = cfg.Server.ServerNames[0]
	}

	shortId := ""
	if len(cfg.Server.ShortIds) > 0 {
		shortId = cfg.Server.ShortIds[0]
	}

	fp := cfg.Server.Fingerprint
	if fp == "" {
		fp = "chrome"
	}

	spx := cfg.Server.SpiderX
	if spx == "" {
		spx = "/"
	}

	flow := u.Flow
	if flow == "" {
		flow = "xtls-rprx-vision"
	}

	query := url.Values{}
	query.Set("type", "tcp")
	query.Set("security", "reality")
	query.Set("encryption", "none")
	query.Set("pbk", cfg.Server.PublicKey)
	query.Set("fp", fp)
	query.Set("sni", sni)
	query.Set("sid", shortId)
	query.Set("spx", spx)
	query.Set("flow", flow)

	host := cfg.Server.PublicHost
	if host == "" {
		host = "127.0.0.1"
	}

	nodeName := fmt.Sprintf("Reality-%s", u.Email)

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		u.ID,
		host,
		cfg.Server.Port,
		query.Encode(),
		url.PathEscape(nodeName),
	)
}

func (c *Client) GenerateQRCodeAscii(link string) (string, error) {
	qr, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		return "", err
	}
	return qr.ToSmallString(false), nil
}
