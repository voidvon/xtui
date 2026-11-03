package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"xtui/config"
	"xtui/core"
)

// centerModal centers a primitive on screen with specified width and height.
func centerModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

// ShowAddUserModal displays a form to add a new VLESS Reality user.
func ShowAddUserModal(pages *tview.Pages, mgr *core.Manager, onDone func()) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" ➕ Add New User ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorCadetBlue)

	var email string
	var limitStr string = "0"
	var daysStr string = "0"

	form.AddInputField("Username / Email", "", 24, nil, func(text string) {
		email = strings.TrimSpace(text)
	})
	form.AddInputField("Traffic Quota (GB, 0 = unlimited)", "0", 12, nil, func(text string) {
		limitStr = strings.TrimSpace(text)
	})
	form.AddInputField("Validity (Days, 0 = permanent)", "0", 12, nil, func(text string) {
		daysStr = strings.TrimSpace(text)
	})

	form.AddButton("Create", func() {
		if email == "" {
			return
		}
		limitGB, _ := strconv.ParseFloat(limitStr, 64)
		validDays, _ := strconv.Atoi(daysStr)

		_, err := mgr.AddUser(email, limitGB, validDays)
		if err != nil {
			ShowAlertModal(pages, "Error", err.Error())
			return
		}
		pages.RemovePage("modal")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("Cancel", func() {
		pages.RemovePage("modal")
	})

	pages.AddPage("modal", centerModal(form, 54, 13), true, true)
}

// ShowQRCodeModal displays the VLESS link and ASCII QR code.
func ShowQRCodeModal(pages *tview.Pages, mgr *core.Manager, user *config.User) {
	link := mgr.GenerateVlessLink(user)
	qrAscii, err := mgr.GenerateQRCodeAscii(link)
	if err != nil {
		qrAscii = fmt.Sprintf("Error generating QR code: %v", err)
	}

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)

	textView.SetBorder(true).
		SetTitle(fmt.Sprintf(" 📱 VLESS Reality Node: [%s] ", user.Email)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorDarkCyan)

	text := fmt.Sprintf(`[yellow]Scan QR Code with mobile client (V2rayN / Shadowrocket / Sing-box / Streisand):[-]

%s

[green]Subscription / Share Link:[-]
[white]%s[-]

[gray]Press [Esc] or [Enter] to return.[-]`, qrAscii, link)

	textView.SetText(text)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			pages.RemovePage("modal")
			return nil
		}
		return event
	})

	pages.AddPage("modal", centerModal(textView, 76, 28), true, true)
}

// ShowServerConfigModal allows editing listening port, reality dest, SNI, and regenerating keys.
func ShowServerConfigModal(pages *tview.Pages, mgr *core.Manager, onDone func()) {
	cfg := mgr.GetConfig()

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" ⚙️ Server Configuration ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGoldenrod)

	portStr := strconv.Itoa(cfg.Server.Port)
	hostStr := cfg.Server.PublicHost
	destStr := cfg.Server.Dest
	sniStr := "gateway.icloud.com"
	if len(cfg.Server.ServerNames) > 0 {
		sniStr = cfg.Server.ServerNames[0]
	}
	regenKeys := false

	form.AddInputField("Listen Port", portStr, 10, nil, func(text string) {
		portStr = strings.TrimSpace(text)
	})
	form.AddInputField("Public IP / Domain", hostStr, 30, nil, func(text string) {
		hostStr = strings.TrimSpace(text)
	})
	form.AddInputField("Reality Target (Dest)", destStr, 30, nil, func(text string) {
		destStr = strings.TrimSpace(text)
	})
	form.AddInputField("Reality SNI", sniStr, 30, nil, func(text string) {
		sniStr = strings.TrimSpace(text)
	})
	form.AddCheckbox("Regenerate Keypair (X25519)", false, func(checked bool) {
		regenKeys = checked
	})

	form.AddButton("Save & Restart", func() {
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			ShowAlertModal(pages, "Validation Error", "Invalid port number (1-65535)")
			return
		}
		if hostStr == "" {
			hostStr = "127.0.0.1"
		}
		if destStr == "" {
			destStr = "gateway.icloud.com:443"
		}
		if sniStr == "" {
			sniStr = "gateway.icloud.com"
		}

		if regenKeys {
			_ = mgr.RegenerateRealityKeys()
		}

		_ = mgr.UpdateServer(port, hostStr, destStr, sniStr)

		pages.RemovePage("modal")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("Cancel", func() {
		pages.RemovePage("modal")
	})

	pages.AddPage("modal", centerModal(form, 62, 18), true, true)
}

// ShowAlertModal displays an informative dialog with an OK button.
func ShowAlertModal(pages *tview.Pages, title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("alert")
		})
	modal.SetTitle(" " + title + " ")
	pages.AddPage("alert", modal, true, true)
}

// ShowConfirmModal shows a Yes/No dialog.
func ShowConfirmModal(pages *tview.Pages, title, message string, onConfirm func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("modal")
			if buttonIndex == 0 && onConfirm != nil {
				onConfirm()
			}
		})
	modal.SetTitle(" " + title + " ")
	pages.AddPage("modal", modal, true, true)
}
