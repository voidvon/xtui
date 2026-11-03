package ipc

import (
	"time"

	"xtui/config"
	"xtui/core"
)

type StatusResponse struct {
	Running   bool                      `json:"running"`
	StartTime time.Time                 `json:"start_time"`
	Config    *config.Config            `json:"config"`
	Stats     map[string]core.UserStats `json:"stats"`
}

type AddUserRequest struct {
	Email      string  `json:"email"`
	LimitGB    float64 `json:"limit_gb"`
	ValidDays  int     `json:"valid_days"`
	CustomUUID string  `json:"custom_uuid"`
	Flow       string  `json:"flow"`
}

type EditUserRequest struct {
	Index      int     `json:"index"`
	Email      string  `json:"email"`
	UUID       string  `json:"uuid"`
	Flow       string  `json:"flow"`
	LimitGB    float64 `json:"limit_gb"`
	ValidDays  int     `json:"valid_days"`
	Enabled    bool    `json:"enabled"`
}

type IndexRequest struct {
	Index int `json:"index"`
}

type UpdateRealityRequest struct {
	Dest        string   `json:"dest"`
	ServerNames []string `json:"server_names"`
	PrivateKey  string   `json:"private_key"`
	PublicKey   string   `json:"public_key"`
	ShortIds    []string `json:"short_ids"`
	SpiderX     string   `json:"spider_x"`
}

type UpdateVLESSRequest struct {
	Port        int    `json:"port"`
	Listen      string `json:"listen"`
	PublicHost  string `json:"public_host"`
	Fingerprint string `json:"fingerprint"`
}

type KeyPairResponse struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type ActionResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
