package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/stats"
	_ "github.com/xtls/xray-core/main/distro/all"

	"xtui/config"
	"xtui/pkg/keys"
)

type UserStats struct {
	SpeedUp   int64 // bytes per second
	SpeedDown int64 // bytes per second
}

type Manager struct {
	cfg        *config.Config
	configPath string
	mu         sync.RWMutex

	instance   *core.Instance
	running    bool
	stopChan   chan struct{}

	statsUpLast   map[string]int64
	statsDownLast map[string]int64
	userStats     map[string]UserStats

	OnUpdate func() // Callback for UI refresh
}

func NewManager(cfg *config.Config, configPath string) *Manager {
	return &Manager{
		cfg:           cfg,
		configPath:    configPath,
		statsUpLast:   make(map[string]int64),
		statsDownLast: make(map[string]int64),
		userStats:     make(map[string]UserStats),
	}
}

func (m *Manager) GetConfig() *config.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) GetUserStats(email string) UserStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userStats[email]
}

// BuildXrayConfig translates our high-level config into an Xray JSON config.
func (m *Manager) BuildXrayConfig() ([]byte, error) {
	now := time.Now().Unix()

	var clients []map[string]interface{}
	for _, u := range m.cfg.Users {
		// Only active and non-exceeded users are accepted by Xray
		if !u.Enabled {
			continue
		}
		if u.ExpiryTime > 0 && now > u.ExpiryTime {
			continue
		}
		if u.LimitBytes > 0 && (u.UpBytes+u.DownBytes) >= u.LimitBytes {
			continue
		}

		flow := u.Flow
		if flow == "" {
			flow = "xtls-rprx-vision"
		}
		clients = append(clients, map[string]interface{}{
			"id":    u.ID,
			"flow":  flow,
			"email": u.Email,
		})
	}

	configMap := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "none",
		},
		"stats": map[string]interface{}{},
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{
				"0": map[string]interface{}{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]interface{}{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "reality-inbound",
				"port":     m.cfg.Server.Port,
				"listen":   m.cfg.Server.Listen,
				"protocol": "vless",
				"settings": map[string]interface{}{
					"clients":    clients,
					"decryption": "none",
				},
				"streamSettings": map[string]interface{}{
					"network":  "tcp",
					"security": "reality",
					"realitySettings": map[string]interface{}{
						"show":        false,
						"dest":        m.cfg.Server.Dest,
						"xver":        0,
						"serverNames": m.cfg.Server.ServerNames,
						"privateKey":  m.cfg.Server.PrivateKey,
						"shortIds":    m.cfg.Server.ShortIds,
					},
				},
			},
		},
		"outbounds": []interface{}{
			map[string]interface{}{
				"protocol": "freedom",
				"tag":      "direct",
			},
			map[string]interface{}{
				"protocol": "blackhole",
				"tag":      "block",
			},
		},
	}

	return json.Marshal(configMap)
}

// Start launches the in-process Xray server and the stats monitoring loop.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	configBytes, err := m.BuildXrayConfig()
	if err != nil {
		return fmt.Errorf("build config error: %w", err)
	}

	inst, err := core.StartInstance("json", configBytes)
	if err != nil {
		return fmt.Errorf("xray start error: %w", err)
	}

	m.instance = inst
	m.running = true
	m.stopChan = make(chan struct{})

	// Reset counters mapping for fresh start
	m.statsUpLast = make(map[string]int64)
	m.statsDownLast = make(map[string]int64)

	go m.statsWorker(m.stopChan)

	return nil
}

// Stop cleanly terminates the embedded Xray instance.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.stopChan)
	if m.instance != nil {
		_ = m.instance.Close()
		m.instance = nil
	}
	m.running = false
	return nil
}

// Restart restarts the core instance with latest configuration.
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return m.Start()
}

// statsWorker runs every 1 second to fetch traffic delta, update total bytes, and calculate speed.
func (m *Manager) statsWorker(stop <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	saveTicker := time.NewTicker(15 * time.Second)
	defer saveTicker.Stop()

	for {
		select {
		case <-stop:
			_ = m.Save()
			return
		case <-saveTicker.C:
			_ = m.Save()
		case <-ticker.C:
			m.pollStats()
		}
	}
}

func (m *Manager) pollStats() {
	m.mu.Lock()
	inst := m.instance
	if inst == nil || !m.running {
		m.mu.Unlock()
		return
	}

	feature := inst.GetFeature(stats.ManagerType())
	if feature == nil {
		m.mu.Unlock()
		return
	}
	sm, ok := feature.(stats.Manager)
	if !ok {
		m.mu.Unlock()
		return
	}

	needRebuild := false
	now := time.Now().Unix()

	for _, u := range m.cfg.Users {
		upName := "user>>>" + u.Email + ">>>traffic>>>uplink"
		downName := "user>>>" + u.Email + ">>>traffic>>>downlink"

		var currUp, currDown int64
		if c := sm.GetCounter(upName); c != nil {
			currUp = c.Value()
		}
		if c := sm.GetCounter(downName); c != nil {
			currDown = c.Value()
		}

		lastUp := m.statsUpLast[u.Email]
		lastDown := m.statsDownLast[u.Email]

		var dUp, dDown int64
		if currUp >= lastUp {
			dUp = currUp - lastUp
		} else {
			dUp = currUp
		}
		if currDown >= lastDown {
			dDown = currDown - lastDown
		} else {
			dDown = currDown
		}

		m.statsUpLast[u.Email] = currUp
		m.statsDownLast[u.Email] = currDown

		u.UpBytes += dUp
		u.DownBytes += dDown

		m.userStats[u.Email] = UserStats{
			SpeedUp:   dUp,
			SpeedDown: dDown,
		}

		// Check quota limits
		if u.LimitBytes > 0 && (u.UpBytes+u.DownBytes) >= u.LimitBytes && u.Enabled {
			needRebuild = true
		}
		// Check expiry
		if u.ExpiryTime > 0 && now >= u.ExpiryTime && u.Enabled {
			needRebuild = true
		}
	}
	m.mu.Unlock()

	if needRebuild {
		go func() {
			_ = m.Restart()
		}()
	}

	if m.OnUpdate != nil {
		m.OnUpdate()
	}
}

func (m *Manager) Save() error {
	m.mu.RLock()
	cfg := m.cfg
	path := m.configPath
	m.mu.RUnlock()

	return config.SaveConfig(path, cfg)
}

// User operations

func (m *Manager) AddUser(email string, limitGB float64, validDays int) (*config.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.cfg.Users {
		if u.Email == email {
			return nil, fmt.Errorf("user with name '%s' already exists", email)
		}
	}

	var limitBytes int64
	if limitGB > 0 {
		limitBytes = int64(limitGB * 1024 * 1024 * 1024)
	}

	var expiryTime int64
	if validDays > 0 {
		expiryTime = time.Now().AddDate(0, 0, validDays).Unix()
	}

	user := &config.User{
		ID:         keys.GenerateUUID(),
		Email:      email,
		Flow:       "xtls-rprx-vision",
		UpBytes:    0,
		DownBytes:  0,
		LimitBytes: limitBytes,
		ExpiryTime: expiryTime,
		Enabled:    true,
		CreatedAt:  time.Now().Unix(),
	}

	m.cfg.Users = append(m.cfg.Users, user)
	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return user, nil
}

func (m *Manager) DeleteUser(index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.cfg.Users) {
		return fmt.Errorf("invalid user index: %d", index)
	}

	email := m.cfg.Users[index].Email
	delete(m.statsUpLast, email)
	delete(m.statsDownLast, email)
	delete(m.userStats, email)

	m.cfg.Users = append(m.cfg.Users[:index], m.cfg.Users[index+1:]...)
	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return nil
}

func (m *Manager) ToggleUser(index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.cfg.Users) {
		return fmt.Errorf("invalid user index")
	}

	m.cfg.Users[index].Enabled = !m.cfg.Users[index].Enabled
	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return nil
}

func (m *Manager) ResetUserTraffic(index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.cfg.Users) {
		return fmt.Errorf("invalid user index")
	}

	u := m.cfg.Users[index]
	u.UpBytes = 0
	u.DownBytes = 0
	delete(m.statsUpLast, u.Email)
	delete(m.statsDownLast, u.Email)

	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return nil
}

func (m *Manager) ResetAllTraffic() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.cfg.Users {
		u.UpBytes = 0
		u.DownBytes = 0
	}
	m.statsUpLast = make(map[string]int64)
	m.statsDownLast = make(map[string]int64)

	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return nil
}

func (m *Manager) UpdateServer(port int, host, dest, sni string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg.Server.Port = port
	m.cfg.Server.PublicHost = host
	m.cfg.Server.Dest = dest
	m.cfg.Server.ServerNames = []string{sni}

	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return nil
}

func (m *Manager) RegenerateRealityKeys() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	priv, pub, err := keys.GenerateX25519()
	if err != nil {
		return err
	}
	m.cfg.Server.PrivateKey = priv
	m.cfg.Server.PublicKey = pub
	m.cfg.Server.ShortIds = []string{keys.GenerateShortId()}

	_ = config.SaveConfig(m.configPath, m.cfg)

	go func() {
		_ = m.Restart()
	}()
	return nil
}

// GenerateVlessLink formats a standard VLESS Reality subscription link.
func (m *Manager) GenerateVlessLink(u *config.User) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sni := "gateway.icloud.com"
	if len(m.cfg.Server.ServerNames) > 0 {
		sni = m.cfg.Server.ServerNames[0]
	}

	shortId := ""
	if len(m.cfg.Server.ShortIds) > 0 {
		shortId = m.cfg.Server.ShortIds[0]
	}

	query := url.Values{}
	query.Set("type", "tcp")
	query.Set("security", "reality")
	query.Set("encryption", "none")
	query.Set("pbk", m.cfg.Server.PublicKey)
	query.Set("fp", "chrome")
	query.Set("sni", sni)
	query.Set("sid", shortId)
	query.Set("flow", "xtls-rprx-vision")

	host := m.cfg.Server.PublicHost
	if host == "" {
		host = "127.0.0.1"
	}

	nodeName := fmt.Sprintf("Reality-%s", u.Email)

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		u.ID,
		host,
		m.cfg.Server.Port,
		query.Encode(),
		url.PathEscape(nodeName),
	)
}

// GenerateQRCodeAscii generates an inverted or regular small ASCII QR code for terminal display.
func (m *Manager) GenerateQRCodeAscii(link string) (string, error) {
	qr, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		return "", err
	}
	return qr.ToSmallString(false), nil
}
