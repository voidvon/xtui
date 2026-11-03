package ui

import (
	"fmt"
	"time"

	"xtui/config"
)

// FormatBytes formats byte counts into human-readable strings (e.g. 1.25 GB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FormatSpeed formats transfer rates into human-readable string (e.g. 2.45 MB/s).
func FormatSpeed(bps int64) string {
	if bps <= 0 {
		return "-    "
	}
	return fmt.Sprintf("%s/s", FormatBytes(bps))
}

// FormatDate converts Unix timestamp to human readable date or "Permanent".
func FormatDate(ts int64) string {
	if ts <= 0 {
		return "Permanent"
	}
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02")
}

// UserStatusText returns the status label and color code for tview formatting.
func UserStatusText(u *config.User) (status string, colorTag string) {
	now := time.Now().Unix()

	if !u.Enabled {
		return "Disabled", "[gray]"
	}
	if u.ExpiryTime > 0 && now >= u.ExpiryTime {
		return "Expired", "[red]"
	}
	if u.LimitBytes > 0 && (u.UpBytes+u.DownBytes) >= u.LimitBytes {
		return "Exceeded", "[red]"
	}
	return "Active", "[green]"
}
