package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"xtui/config"
	"xtui/core"
	"xtui/pkg/keys"
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
func ShowAddUserModal(pages *tview.Pages, backend core.Backend, onDone func()) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" ➕ 添加新用户 (Add User) ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorCadetBlue)

	var email string
	customUUID := keys.GenerateUUID()
	flow := "xtls-rprx-vision"
	var limitStr string = "0"
	var daysStr string = "0"

	form.AddInputField("用户名 / 邮箱", "", 24, nil, func(text string) {
		email = strings.TrimSpace(text)
	})
	form.AddInputField("用户 UUID", customUUID, 38, nil, func(text string) {
		customUUID = strings.TrimSpace(text)
	})
	form.AddDropDown("流控 (Flow)", []string{"xtls-rprx-vision", "none"}, 0, func(option string, index int) {
		flow = option
	})
	form.AddInputField("流量配额 (GB, 0为不限)", "0", 12, nil, func(text string) {
		limitStr = strings.TrimSpace(text)
	})
	form.AddInputField("有效期 (天, 0为永久)", "0", 12, nil, func(text string) {
		daysStr = strings.TrimSpace(text)
	})

	form.AddButton("创建用户", func() {
		if email == "" {
			ShowAlertModal(pages, "提示", "用户名不能为空")
			return
		}
		limitGB, _ := strconv.ParseFloat(limitStr, 64)
		validDays, _ := strconv.Atoi(daysStr)

		_, err := backend.AddUser(email, limitGB, validDays, customUUID, flow)
		if err != nil {
			ShowAlertModal(pages, "错误", err.Error())
			return
		}
		pages.RemovePage("modal")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("取消", func() {
		pages.RemovePage("modal")
	})

	pages.AddPage("modal", centerModal(form, 64, 18), true, true)
}

// ShowEditUserModal displays a form to modify an existing user.
func ShowEditUserModal(pages *tview.Pages, backend core.Backend, userIndex int, onDone func()) {
	cfg := backend.GetConfig()
	if userIndex < 0 || userIndex >= len(cfg.Users) {
		return
	}
	user := cfg.Users[userIndex]

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" ✏️ 修改用户: [%s] ", user.Email)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorCadetBlue)

	email := user.Email
	uuidStr := user.ID
	flow := user.Flow
	if flow == "" {
		flow = "xtls-rprx-vision"
	}
	enabled := user.Enabled

	limitGB := float64(user.LimitBytes) / (1024 * 1024 * 1024)
	limitStr := fmt.Sprintf("%.2f", limitGB)
	if user.LimitBytes == 0 {
		limitStr = "0"
	}

	daysStr := "0" // 0 preserves or permanent

	form.AddInputField("用户名 / 邮箱", email, 24, nil, func(text string) {
		email = strings.TrimSpace(text)
	})
	form.AddInputField("UUID", uuidStr, 38, nil, func(text string) {
		uuidStr = strings.TrimSpace(text)
	})

	flowIdx := 0
	if flow == "none" {
		flowIdx = 1
	}
	form.AddDropDown("流控 (Flow)", []string{"xtls-rprx-vision", "none"}, flowIdx, func(option string, index int) {
		flow = option
	})

	form.AddInputField("流量配额 (GB, 0为不限)", limitStr, 12, nil, func(text string) {
		limitStr = strings.TrimSpace(text)
	})
	form.AddInputField("重设有效期 (天, 留空或0不重设)", daysStr, 12, nil, func(text string) {
		daysStr = strings.TrimSpace(text)
	})
	form.AddCheckbox("启用此用户", enabled, func(checked bool) {
		enabled = checked
	})

	form.AddButton("保存修改", func() {
		if email == "" {
			ShowAlertModal(pages, "提示", "用户名不能为空")
			return
		}
		newLimitGB, _ := strconv.ParseFloat(limitStr, 64)
		newDays, _ := strconv.Atoi(daysStr)

		err := backend.EditUser(userIndex, email, uuidStr, flow, newLimitGB, newDays, enabled)
		if err != nil {
			ShowAlertModal(pages, "错误", err.Error())
			return
		}
		pages.RemovePage("modal")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("取消", func() {
		pages.RemovePage("modal")
	})

	pages.AddPage("modal", centerModal(form, 66, 20), true, true)
}

// ShowRealityConfigModal allows viewing and configuring all Reality parameters.
func ShowRealityConfigModal(pages *tview.Pages, backend core.Backend, onDone func()) {
	cfg := backend.GetConfig()

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" 🛡️ Reality 核心参数配置 ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorDarkCyan)

	destStr := cfg.Server.Dest
	sniStr := strings.Join(cfg.Server.ServerNames, ",")
	privStr := cfg.Server.PrivateKey
	pubStr := cfg.Server.PublicKey
	shortIdsStr := strings.Join(cfg.Server.ShortIds, ",")
	spiderXStr := cfg.Server.SpiderX
	if spiderXStr == "" {
		spiderXStr = "/"
	}

	presetOptions := []string{
		"自定义 (Custom)",
		"Apple iCloud (gateway.icloud.com:443)",
		"Microsoft (www.microsoft.com:443)",
		"Amazon AWS (aws.amazon.com:443)",
		"Cloudflare (www.cloudflare.com:443)",
		"Yahoo Japan (www.yahoo.co.jp:443)",
	}

	form.AddDropDown("预设目标快捷选择", presetOptions, 0, func(option string, index int) {
		if index > 0 && index <= len(core.RealityPresets) {
			p := core.RealityPresets[index-1]
			destStr = p.Dest
			sniStr = strings.Join(p.ServerNames, ",")
			// update fields in form if possible
			if destItem := form.GetFormItemByLabel("目标网站 (Dest)"); destItem != nil {
				if input, ok := destItem.(*tview.InputField); ok {
					input.SetText(destStr)
				}
			}
			if sniItem := form.GetFormItemByLabel("伪装域名 (SNI)"); sniItem != nil {
				if input, ok := sniItem.(*tview.InputField); ok {
					input.SetText(sniStr)
				}
			}
		}
	})

	form.AddInputField("目标网站 (Dest)", destStr, 34, nil, func(text string) {
		destStr = strings.TrimSpace(text)
	})
	form.AddInputField("伪装域名 (SNI)", sniStr, 34, nil, func(text string) {
		sniStr = strings.TrimSpace(text)
	})
	form.AddInputField("短 ID (ShortIds 逗号分隔)", shortIdsStr, 34, nil, func(text string) {
		shortIdsStr = strings.TrimSpace(text)
	})
	form.AddInputField("爬虫伪装路径 (SpiderX)", spiderXStr, 15, nil, func(text string) {
		spiderXStr = strings.TrimSpace(text)
	})
	form.AddInputField("私钥 (PrivateKey)", privStr, 48, nil, func(text string) {
		privStr = strings.TrimSpace(text)
		// try auto-deriving public key
		if derivedPub, err := keys.DerivePublicKey(privStr); err == nil {
			pubStr = derivedPub
			if pubItem := form.GetFormItemByLabel("公钥 (PublicKey)"); pubItem != nil {
				if input, ok := pubItem.(*tview.InputField); ok {
					input.SetText(pubStr)
				}
			}
		}
	})
	form.AddInputField("公钥 (PublicKey)", pubStr, 48, nil, func(text string) {
		pubStr = strings.TrimSpace(text)
	})

	form.AddButton("随机重新生成密钥对", func() {
		newPriv, newPub, err := keys.GenerateX25519()
		if err == nil {
			privStr = newPriv
			pubStr = newPub
			shortIdsStr = keys.GenerateShortId()
			if privItem := form.GetFormItemByLabel("私钥 (PrivateKey)"); privItem != nil {
				privItem.(*tview.InputField).SetText(privStr)
			}
			if pubItem := form.GetFormItemByLabel("公钥 (PublicKey)"); pubItem != nil {
				pubItem.(*tview.InputField).SetText(pubStr)
			}
			if sidItem := form.GetFormItemByLabel("短 ID (ShortIds 逗号分隔)"); sidItem != nil {
				sidItem.(*tview.InputField).SetText(shortIdsStr)
			}
		}
	})

	form.AddButton("保存并重启生效", func() {
		if destStr == "" {
			ShowAlertModal(pages, "提示", "目标网站 (Dest) 不能为空")
			return
		}
		if pubStr == "" || privStr == "" {
			ShowAlertModal(pages, "提示", "密钥对不能为空")
			return
		}

		snis := strings.Split(sniStr, ",")
		var cleanSNIs []string
		for _, s := range snis {
			s = strings.TrimSpace(s)
			if s != "" {
				cleanSNIs = append(cleanSNIs, s)
			}
		}
		if len(cleanSNIs) == 0 {
			cleanSNIs = []string{"gateway.icloud.com"}
		}

		sids := strings.Split(shortIdsStr, ",")
		var cleanSIDs []string
		for _, s := range sids {
			s = strings.TrimSpace(s)
			if s != "" {
				cleanSIDs = append(cleanSIDs, s)
			}
		}
		if len(cleanSIDs) == 0 {
			cleanSIDs = []string{keys.GenerateShortId()}
		}

		_ = backend.UpdateReality(destStr, cleanSNIs, privStr, pubStr, cleanSIDs, spiderXStr)
		pages.RemovePage("modal")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("取消", func() {
		pages.RemovePage("modal")
	})

	pages.AddPage("modal", centerModal(form, 76, 25), true, true)
}

// ShowVlessConfigModal allows configuring VLESS port, IP, and fingerprint.
func ShowVlessConfigModal(pages *tview.Pages, backend core.Backend, onDone func()) {
	cfg := backend.GetConfig()

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" ⚙️ VLESS 服务与网络配置 ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGoldenrod)

	portStr := strconv.Itoa(cfg.Server.Port)
	listenStr := cfg.Server.Listen
	hostStr := cfg.Server.PublicHost
	fpStr := cfg.Server.Fingerprint
	if fpStr == "" {
		fpStr = "chrome"
	}

	form.AddInputField("监听端口 (Port)", portStr, 10, nil, func(text string) {
		portStr = strings.TrimSpace(text)
	})
	form.AddInputField("监听地址 (Listen IP)", listenStr, 18, nil, func(text string) {
		listenStr = strings.TrimSpace(text)
	})
	form.AddInputField("节点公网 IP/域名 (用于分享链接)", hostStr, 32, nil, func(text string) {
		hostStr = strings.TrimSpace(text)
	})

	fps := []string{"chrome", "safari", "firefox", "edge", "ios", "random"}
	fpIdx := 0
	for i, f := range fps {
		if f == fpStr {
			fpIdx = i
			break
		}
	}
	form.AddDropDown("客户端 uTLS 伪装指纹 (Fingerprint)", fps, fpIdx, func(option string, index int) {
		fpStr = option
	})

	form.AddButton("自动重新探测公网 IP", func() {
		detected := config.FetchPublicIP()
		hostStr = detected
		if hostItem := form.GetFormItemByLabel("节点公网 IP/域名 (用于分享链接)"); hostItem != nil {
			hostItem.(*tview.InputField).SetText(detected)
		}
	})

	form.AddButton("保存并生效", func() {
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			ShowAlertModal(pages, "验证失败", "端口范围必须为 1 - 65535")
			return
		}
		if listenStr == "" {
			listenStr = "0.0.0.0"
		}
		if hostStr == "" {
			hostStr = "127.0.0.1"
		}

		_ = backend.UpdateVLESS(port, listenStr, hostStr, fpStr)
		pages.RemovePage("modal")
		if onDone != nil {
			onDone()
		}
	})

	form.AddButton("取消", func() {
		pages.RemovePage("modal")
	})

	pages.AddPage("modal", centerModal(form, 68, 19), true, true)
}

// ShowQRCodeModal displays the VLESS link, node details, and ASCII QR code.
func ShowQRCodeModal(pages *tview.Pages, backend core.Backend, user *config.User) {
	link := backend.GenerateVlessLink(user)
	qrAscii, err := backend.GenerateQRCodeAscii(link)
	if err != nil {
		qrAscii = fmt.Sprintf("二维码生成失败: %v", err)
	}

	cfg := backend.GetConfig()
	sni := "gateway.icloud.com"
	if len(cfg.Server.ServerNames) > 0 {
		sni = cfg.Server.ServerNames[0]
	}
	sid := ""
	if len(cfg.Server.ShortIds) > 0 {
		sid = cfg.Server.ShortIds[0]
	}

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)

	textView.SetBorder(true).
		SetTitle(fmt.Sprintf(" 📱 VLESS Reality 节点信息: [%s] ", user.Email)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorDarkCyan)

	text := fmt.Sprintf(`[yellow]手机客户端直接扫码导入 (V2rayN / Shadowrocket / Sing-box / Streisand / Clash):[-]

%s

[green]节点详细参数:[-]
• [gray]协议:[-] VLESS (TCP + Reality)
• [gray]流控:[-] %s
• [gray]用户 UUID:[-] [yellow]%s[-]
• [gray]地址端口:[-] %s:%d
• [gray]Reality 目标:[-] %s (SNI: %s)
• [gray]ShortID:[-] %s
• [gray]公钥 (PBK):[-] %s

[green]标准分享导入链接:[-]
[white]%s[-]

[gray]按 [Esc] 或 [Enter] 返回[-]`,
		qrAscii,
		user.Flow,
		user.ID,
		cfg.Server.PublicHost,
		cfg.Server.Port,
		cfg.Server.Dest,
		sni,
		sid,
		cfg.Server.PublicKey,
		link)

	textView.SetText(text)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			pages.RemovePage("modal")
			return nil
		}
		return event
	})

	pages.AddPage("modal", centerModal(textView, 80, 32), true, true)
}

// ShowAlertModal displays an informative dialog with an OK button.
func ShowAlertModal(pages *tview.Pages, title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"确认"}).
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
		AddButtons([]string{"确认", "取消"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("modal")
			if buttonIndex == 0 && onConfirm != nil {
				onConfirm()
			}
		})
	modal.SetTitle(" " + title + " ")
	pages.AddPage("modal", modal, true, true)
}
