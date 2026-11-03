package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"xtui/core"
)

type UI struct {
	app    *tview.Application
	pages  *tview.Pages
	mgr    *core.Manager
	table  *tview.Table
	header *tview.TextView
	footer *tview.TextView
}

func NewUI(mgr *core.Manager) *UI {
	app := tview.NewApplication()
	pages := tview.NewPages()

	header := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite))

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)

	ui := &UI{
		app:    app,
		pages:  pages,
		mgr:    mgr,
		table:  table,
		header: header,
		footer: footer,
	}

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 3, 1, false).
		AddItem(table, 0, 1, true).
		AddItem(footer, 2, 1, false)

	pages.AddPage("main", flex, true, true)
	app.SetRoot(pages, true)

	ui.setupKeyboard()
	ui.renderHeader()
	ui.renderFooter()
	ui.renderTable()

	mgr.OnUpdate = func() {
		app.QueueUpdateDraw(func() {
			ui.renderHeader()
			ui.renderTable()
		})
	}

	return ui
}

func (u *UI) Run() error {
	return u.app.Run()
}

func (u *UI) Stop() {
	u.app.Stop()
}

func (u *UI) setupKeyboard() {
	u.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := u.table.GetSelection()
		userIndex := row - 1 // row 0 is header

		switch event.Rune() {
		case 'q', 'Q':
			u.app.Stop()
			return nil
		case 'a', 'A':
			ShowAddUserModal(u.pages, u.mgr, func() {
				u.renderTable()
			})
			return nil
		case 's', 'S':
			u.showSelectedQR(userIndex)
			return nil
		case 'd', 'D':
			u.deleteSelectedUser(userIndex)
			return nil
		case 't', 'T':
			if userIndex >= 0 && userIndex < len(u.mgr.GetConfig().Users) {
				_ = u.mgr.ToggleUser(userIndex)
				u.renderTable()
			}
			return nil
		case 'r':
			u.resetSelectedTraffic(userIndex)
			return nil
		case 'c', 'C':
			ShowServerConfigModal(u.pages, u.mgr, func() {
				u.renderHeader()
				u.renderTable()
			})
			return nil
		case 'R':
			go func() {
				_ = u.mgr.Restart()
				u.app.QueueUpdateDraw(func() {
					u.renderHeader()
				})
			}()
			return nil
		}

		if event.Key() == tcell.KeyEnter {
			u.showSelectedQR(userIndex)
			return nil
		}

		return event
	})
}

func (u *UI) showSelectedQR(userIndex int) {
	cfg := u.mgr.GetConfig()
	if userIndex >= 0 && userIndex < len(cfg.Users) {
		user := cfg.Users[userIndex]
		ShowQRCodeModal(u.pages, u.mgr, user)
	}
}

func (u *UI) deleteSelectedUser(userIndex int) {
	cfg := u.mgr.GetConfig()
	if userIndex < 0 || userIndex >= len(cfg.Users) {
		return
	}
	user := cfg.Users[userIndex]
	ShowConfirmModal(u.pages, "Confirm Delete",
		fmt.Sprintf("Are you sure you want to delete user '%s'?", user.Email),
		func() {
			_ = u.mgr.DeleteUser(userIndex)
			u.renderTable()
		})
}

func (u *UI) resetSelectedTraffic(userIndex int) {
	cfg := u.mgr.GetConfig()
	if userIndex < 0 || userIndex >= len(cfg.Users) {
		return
	}
	user := cfg.Users[userIndex]
	ShowConfirmModal(u.pages, "Reset Traffic",
		fmt.Sprintf("Reset accumulated traffic for user '%s'?", user.Email),
		func() {
			_ = u.mgr.ResetUserTraffic(userIndex)
			u.renderTable()
		})
}

func (u *UI) renderHeader() {
	cfg := u.mgr.GetConfig()
	running := u.mgr.IsRunning()

	statusBadge := "[black:green:b] RUNNING [-:-:-]"
	if !running {
		statusBadge = "[black:red:b] STOPPED [-:-:-]"
	}

	var totalSpeedUp, totalSpeedDown int64
	for _, user := range cfg.Users {
		stats := u.mgr.GetUserStats(user.Email)
		totalSpeedUp += stats.SpeedUp
		totalSpeedDown += stats.SpeedDown
	}

	sni := "gateway.icloud.com"
	if len(cfg.Server.ServerNames) > 0 {
		sni = cfg.Server.ServerNames[0]
	}

	line1 := fmt.Sprintf("  [teal:b]XTUI[-] [white:d]v1.0[-]  %s  [gray]|[-] Port: [yellow]%d[-] [gray]|[-] SNI: [aqua]%s[-] [gray]|[-] Host: [aqua]%s[-]",
		statusBadge, cfg.Server.Port, sni, cfg.Server.PublicHost)

	line2 := fmt.Sprintf("  [gray]Users:[-] [white]%d[-]  [gray]|[-] [gray]Real-time Bandwidth:[-] ↑ [lime]%s[-]  ↓ [lime]%s[-]",
		len(cfg.Users), FormatSpeed(totalSpeedUp), FormatSpeed(totalSpeedDown))

	divider := "  " + strings.Repeat("─", 84)

	u.header.SetText(fmt.Sprintf("%s\n%s\n%s", line1, line2, divider))
}

func (u *UI) renderFooter() {
	shortcuts := "  [yellow][a][-] Add  [yellow][s/Enter][-] QR/Link  [yellow][t][-] Toggle  [yellow][d][-] Del  [yellow][r][-] Reset  [yellow][c][-] Config  [yellow][R][-] Restart  [yellow][q][-] Quit"
	u.footer.SetText(shortcuts)
}

func (u *UI) renderTable() {
	cfg := u.mgr.GetConfig()

	// Remember selection
	selectedRow, _ := u.table.GetSelection()

	u.table.Clear()

	headers := []string{"#", "User (Email)", "Status", "↑ Upload", "↓ Download", "Total", "Quota", "Speed (↑/↓)", "Expire"}
	for col, h := range headers {
		cell := tview.NewTableCell("[yellow:b]" + h + "[-:-:-]").
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if col == 0 {
			cell.SetAlign(tview.AlignCenter)
		}
		u.table.SetCell(0, col, cell)
	}

	for i, user := range cfg.Users {
		row := i + 1
		stats := u.mgr.GetUserStats(user.Email)
		statusStr, colorTag := UserStatusText(user)

		totalBytes := user.UpBytes + user.DownBytes
		quotaStr := "Unlimited"
		if user.LimitBytes > 0 {
			quotaStr = FormatBytes(user.LimitBytes)
		}

		speedStr := fmt.Sprintf("↑%s ↓%s", FormatSpeed(stats.SpeedUp), FormatSpeed(stats.SpeedDown))

		u.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf(" %02d ", row)).SetAlign(tview.AlignCenter).SetTextColor(tcell.ColorDarkGray))
		u.table.SetCell(row, 1, tview.NewTableCell(" "+user.Email+" ").SetTextColor(tcell.ColorWhite))
		u.table.SetCell(row, 2, tview.NewTableCell(" "+colorTag+statusStr+"[-] "))
		u.table.SetCell(row, 3, tview.NewTableCell(" "+FormatBytes(user.UpBytes)+" ").SetTextColor(tcell.ColorLightCyan))
		u.table.SetCell(row, 4, tview.NewTableCell(" "+FormatBytes(user.DownBytes)+" ").SetTextColor(tcell.ColorLightGreen))
		u.table.SetCell(row, 5, tview.NewTableCell(" "+FormatBytes(totalBytes)+" ").SetTextColor(tcell.ColorLightYellow))
		u.table.SetCell(row, 6, tview.NewTableCell(" "+quotaStr+" ").SetTextColor(tcell.ColorWheat))
		u.table.SetCell(row, 7, tview.NewTableCell(" "+speedStr+" ").SetTextColor(tcell.ColorCadetBlue))
		u.table.SetCell(row, 8, tview.NewTableCell(" "+FormatDate(user.ExpiryTime)+" ").SetTextColor(tcell.ColorLightSlateGray))
	}

	// Restore selection
	if len(cfg.Users) > 0 {
		if selectedRow <= 0 {
			selectedRow = 1
		} else if selectedRow > len(cfg.Users) {
			selectedRow = len(cfg.Users)
		}
		u.table.Select(selectedRow, 0)
	}
}
