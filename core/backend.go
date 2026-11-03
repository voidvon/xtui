package core

import (
	"time"

	"xtui/config"
)

// Backend is the common interface implemented by both in-process Manager and IPC Client.
type Backend interface {
	GetConfig() *config.Config
	IsRunning() bool
	GetStartTime() time.Time
	GetUserStats(email string) UserStats

	AddUser(email string, limitGB float64, validDays int, customUUID, flow string) (*config.User, error)
	EditUser(index int, email, uuidStr, flow string, limitGB float64, validDays int, enabled bool) error
	DeleteUser(index int) error
	ToggleUser(index int) error
	ResetUserTraffic(index int) error
	ResetAllTraffic() error

	UpdateReality(dest string, serverNames []string, privKey, pubKey string, shortIds []string, spiderX string) error
	UpdateVLESS(port int, listen, publicHost, fingerprint string) error
	RegenerateRealityKeys() (priv, pub string, err error)

	GenerateVlessLink(u *config.User) string
	GenerateQRCodeAscii(link string) (string, error)

	Restart() error
	Stop() error
	Start() error

	SetOnUpdate(fn func())
}
