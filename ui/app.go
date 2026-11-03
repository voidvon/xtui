package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"xtui/config"
	"xtui/core"
)

type UI struct {
	app         *tview.Application
	rootPages   *tview.Pages
	tabPages    *tview.Pages
	mgr         *core.Manager
	activeTab   int // 0: Users, 1: Reality, 2: VLESS, 3: Status

	headerBar   *tview.TextView
	table       *tview.Table
	realityForm *tview.Form
	vlessForm   *tview.Form
	statusView  *tview.TextView
	footerBar   *tview.TextView
}

func NewUI(mgr *core.Manager) *UI {
	app := tview.NewApplication()
	app.EnableMouse(true) // Enable terminal mouse clicking!

	rootPages := tview.NewPages()
	tabPages := tview.NewPages()

	ui := &UI{
		app:       app,
		rootPages: rootPages,
		tabPages:  tabPages,
		mgr:       mgr,
		activeTab: 0,
	}

	ui.setupViews()
	ui.setupKeyboard()

	// Main container layout
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.headerBar, 4, 1, false).
		AddItem(ui.tabPages, 0, 1, true).
		AddItem(ui.footerBar, 2, 1, false)

	rootPages.AddPage("main", mainFlex, true, true)
	app.SetRoot(rootPages, true)

	mgr.OnUpdate = func() {
		app.QueueUpdateDraw(func() {
			ui.refreshCurrentView()
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

func (u *UI) setupViews() {
	// Top Header
	u.headerBar = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)

	// Bottom Footer
	u.footerBar = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)

	// Tab 1: User Table
	u.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite))

	userFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(u.table, 0, 1, true)

	// Tab 2: Reality Form
	u.setupRealityForm()

	// Tab 3: VLESS Form
	u.setupVlessForm()

	// Tab 4: Status View
	u.statusView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	u.statusView.SetBorder(true).
		SetTitle(" ⚡ 核心运行状态与管理 ").
		SetBorderColor(tcell.ColorCadetBlue)

	// Register Tab pages
	u.tabPages.AddPage("tab_users", userFlex, true, true)
	u.tabPages.AddPage("tab_reality", u.realityForm, true, false)
	u.tabPages.AddPage("tab_vless", u.vlessForm, true, false)
	u.tabPages.AddPage("tab_status", u.statusView, true, false)

	u.renderHeader()
	u.renderFooter()
	u.renderTable()
	u.renderStatus()
}

func (u *UI) switchTab(tabIndex int) {
	u.activeTab = tabIndex
	switch tabIndex {
	case 0:
		u.tabPages.SwitchToPage("tab_users")
		u.app.SetFocus(u.table)
	case 1:
		u.setupRealityForm() // refresh data
		u.tabPages.SwitchToPage("tab_reality")
		u.app.SetFocus(u.realityForm)
	case 2:
		u.setupVlessForm() // refresh data
		u.tabPages.SwitchToPage("tab_vless")
		u.app.SetFocus(u.vlessForm)
	case 3:
		u.renderStatus()
		u.tabPages.SwitchToPage("tab_status")
		u.app.SetFocus(u.statusView)
	}
	u.renderHeader()
	u.renderFooter()
}

func (u *UI) setupRealityForm() {
	cfg := u.mgr.GetConfig()
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" 🛡️ Reality 伪装参数配置 (Tab 2) ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorDarkCyan)

	destStr := cfg.Server.Dest
	sniStr := strings.Join(cfg.Server.ServerNames, ",")
	privStr := cfg.Server.PrivateKey
	pubStr := cfg.Server.PublicKey
	shortIdsStr := strings.Join(cfg.Server.ShortIds, ",")
	spiderXStr := cfg.Server.SpiderX

	presetOptions := []string{
		"选择预设模板 (自动填入Dest/SNI)",
		"Apple iCloud (gateway.icloud.com:443)",
		"Microsoft (www.microsoft.com:443)",
		"Amazon AWS (aws.amazon.com:443)",
		"Cloudflare (www.cloudflare.com:443)",
		"Yahoo Japan (www.yahoo.co.jp:443)",
	}

	form.AddDropDown("预设目标模板", presetOptions, 0, func(option string, index int) {
		if index > 0 && index <= len(core.RealityPresets) {
			p := core.RealityPresets[index-1]
			destStr = p.Dest
			sniStr = strings.Join(p.ServerNames, ",")
			if item := form.GetFormItemByLabel("目标网站 (Dest)"); item != nil {
				item.(*tview.InputField).SetText(destStr)
			}
			if item := form.GetFormItemByLabel("伪装域名 (SNI)"); item != nil {
				item.(*tview.InputField).SetText(sniStr)
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
	})
	form.AddInputField("公钥 (PublicKey)", pubStr, 48, nil, func(text string) {
		pubStr = strings.TrimSpace(text)
	})

	form.AddButton("随机重新生成密钥对 (Regen Keys)", func() {
		newPriv, newPub, err := u.mgr.RegenerateRealityKeys()
		if err == nil {
			privStr = newPriv
			pubStr = newPub
			if item := form.GetFormItemByLabel("私钥 (PrivateKey)"); item != nil {
				item.(*tview.InputField).SetText(privStr)
			}
			if item := form.GetFormItemByLabel("公钥 (PublicKey)"); item != nil {
				item.(*tview.InputField).SetText(pubStr)
			}
			ShowAlertModal(u.rootPages, "成功", "已重新生成 Reality 密钥对并重载服务！")
		}
	})

	form.AddButton("保存并重启内核 (Save & Apply)", func() {
		if destStr == "" || privStr == "" || pubStr == "" {
			ShowAlertModal(u.rootPages, "提示", "目标 Dest 和密钥对不能为空")
			return
		}
		var cleanSNIs []string
		for _, s := range strings.Split(sniStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				cleanSNIs = append(cleanSNIs, s)
			}
		}
		var cleanSIDs []string
		for _, s := range strings.Split(shortIdsStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				cleanSIDs = append(cleanSIDs, s)
			}
		}

		_ = u.mgr.UpdateReality(destStr, cleanSNIs, privStr, pubStr, cleanSIDs, spiderXStr)
		ShowAlertModal(u.rootPages, "成功", "Reality 参数已保存并应用！")
		u.renderHeader()
	})

	form.AddButton("返回用户列表 (Back)", func() {
		u.switchTab(0)
	})

	u.realityForm = form
}

func (u *UI) setupVlessForm() {
	cfg := u.mgr.GetConfig()
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" ⚙️ VLESS 服务与网络参数 (Tab 3) ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGoldenrod)

	portStr := fmt.Sprintf("%d", cfg.Server.Port)
	listenStr := cfg.Server.Listen
	hostStr := cfg.Server.PublicHost
	fpStr := cfg.Server.Fingerprint

	form.AddInputField("监听端口 (Listen Port)", portStr, 10, nil, func(text string) {
		portStr = strings.TrimSpace(text)
	})
	form.AddInputField("监听地址 (Listen IP)", listenStr, 18, nil, func(text string) {
		listenStr = strings.TrimSpace(text)
	})
	form.AddInputField("节点公网 IP/域名 (用于分享链接)", hostStr, 32, nil, func(text string) {
		hostStr = strings.TrimSpace(text)
	})

	fps := []string{"chrome", "safari", "firefox", "edge", "ios"}
	fpIdx := 0
	for i, f := range fps {
		if f == fpStr {
			fpIdx = i
			break
		}
	}
	form.AddDropDown("客户端指纹 (Fingerprint)", fps, fpIdx, func(option string, index int) {
		fpStr = option
	})

	form.AddButton("自动检测公网 IP (Detect IP)", func() {
		detected := config.FetchPublicIP()
		hostStr = detected
		if item := form.GetFormItemByLabel("节点公网 IP/域名 (用于分享链接)"); item != nil {
			item.(*tview.InputField).SetText(detected)
		}
	})

	form.AddButton("保存并重启内核 (Save & Apply)", func() {
		var port int
		fmt.Sscanf(portStr, "%d", &port)
		if port <= 0 || port > 65535 {
			ShowAlertModal(u.rootPages, "错误", "端口必须在 1 - 65535 之间")
			return
		}
		_ = u.mgr.UpdateVLESS(port, listenStr, hostStr, fpStr)
		ShowAlertModal(u.rootPages, "成功", "VLESS 参数已保存并应用！")
		u.renderHeader()
	})

	form.AddButton("返回用户列表 (Back)", func() {
		u.switchTab(0)
	})

	u.vlessForm = form
}

func (u *UI) renderStatus() {
	cfg := u.mgr.GetConfig()
	running := u.mgr.IsRunning()

	statusBadge := "[green]● 运行中 (RUNNING)[-]"
	if !running {
		statusBadge = "[red]● 已停止 (STOPPED)[-]"
	}

	uptimeStr := "-"
	if running {
		uptime := time.Since(u.mgr.GetStartTime()).Truncate(time.Second)
		uptimeStr = uptime.String()
	}

	var totalUp, totalDown, curSpeedUp, curSpeedDown int64
	for _, user := range cfg.Users {
		totalUp += user.UpBytes
		totalDown += user.DownBytes
		st := u.mgr.GetUserStats(user.Email)
		curSpeedUp += st.SpeedUp
		curSpeedDown += st.SpeedDown
	}

	sni := "gateway.icloud.com"
	if len(cfg.Server.ServerNames) > 0 {
		sni = cfg.Server.ServerNames[0]
	}

	text := fmt.Sprintf(`
  [yellow:b]XTUI 代理核心状态概览:[-:-:-]

  • 内核运行状态:    %s
  • 持续运行时间:    [white]%s[-]
  • 监听端口/协议:   [yellow]%d[-] (TCP + VLESS Reality)
  • 伪装目标 (Dest): [aqua]%s[-] (SNI: %s)
  • 节点公网地址:    [aqua]%s[-]
  • 注册用户总数:    [white]%d 位[-]

  [yellow:b]全局流量与速率统计:[-:-:-]
  • 实时下行速率:    ↓ [lime]%s[-]
  • 实时上行速率:    ↑ [lime]%s[-]
  • 全站累计下载:    [green]%s[-]
  • 全站累计上传:    [green]%s[-]
  • 全站总计流量:    [white]%s[-]

  [yellow:b]快捷管理操作:[-:-:-]
  按 [white][R][-] 重启代理内核   按 [white][r][-] 清零所有用户流量   按 [white][1][-] 返回用户列表   按 [white][q][-] 退出程序
`,
		statusBadge,
		uptimeStr,
		cfg.Server.Port,
		cfg.Server.Dest,
		sni,
		cfg.Server.PublicHost,
		len(cfg.Users),
		FormatSpeed(curSpeedDown),
		FormatSpeed(curSpeedUp),
		FormatBytes(totalDown),
		FormatBytes(totalUp),
		FormatBytes(totalUp+totalDown),
	)

	u.statusView.SetText(text)
}

func (u *UI) setupKeyboard() {
	u.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Global tab switching with keys 1, 2, 3, 4 (when not editing an input)
		// Or F1, F2, F3, F4
		switch event.Key() {
		case tcell.KeyF1:
			u.switchTab(0)
			return nil
		case tcell.KeyF2:
			u.switchTab(1)
			return nil
		case tcell.KeyF3:
			u.switchTab(2)
			return nil
		case tcell.KeyF4:
			u.switchTab(3)
			return nil
		}

		// When table has focus or in tab 0
		if u.activeTab == 0 && u.rootPages.HasPage("modal") == false {
			row, _ := u.table.GetSelection()
			userIndex := row - 1

			switch event.Rune() {
			case '1':
				u.switchTab(0)
				return nil
			case '2':
				u.switchTab(1)
				return nil
			case '3':
				u.switchTab(2)
				return nil
			case '4':
				u.switchTab(3)
				return nil
			case 'q', 'Q':
				u.app.Stop()
				return nil
			case 'a', 'A':
				ShowAddUserModal(u.rootPages, u.mgr, func() {
					u.renderTable()
					u.renderHeader()
				})
				return nil
			case 'e', 'E':
				if userIndex >= 0 && userIndex < len(u.mgr.GetConfig().Users) {
					ShowEditUserModal(u.rootPages, u.mgr, userIndex, func() {
						u.renderTable()
						u.renderHeader()
					})
				}
				return nil
			case 'd', 'D':
				u.deleteSelectedUser(userIndex)
				return nil
			case 's', 'S':
				u.showSelectedQR(userIndex)
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
				u.switchTab(1) // switch to Reality config
				return nil
			case 'v', 'V':
				u.switchTab(2) // switch to VLESS config
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
		}

		if (event.Rune() == 'q' || event.Rune() == 'Q') && u.activeTab == 3 {
			u.app.Stop()
			return nil
		}

		return event
	})
}

func (u *UI) showSelectedQR(userIndex int) {
	cfg := u.mgr.GetConfig()
	if userIndex >= 0 && userIndex < len(cfg.Users) {
		user := cfg.Users[userIndex]
		ShowQRCodeModal(u.rootPages, u.mgr, user)
	}
}

func (u *UI) deleteSelectedUser(userIndex int) {
	cfg := u.mgr.GetConfig()
	if userIndex < 0 || userIndex >= len(cfg.Users) {
		return
	}
	user := cfg.Users[userIndex]
	ShowConfirmModal(u.rootPages, "确认删除",
		fmt.Sprintf("确定要彻底删除用户 '%s' 吗？", user.Email),
		func() {
			_ = u.mgr.DeleteUser(userIndex)
			u.renderTable()
			u.renderHeader()
		})
}

func (u *UI) resetSelectedTraffic(userIndex int) {
	cfg := u.mgr.GetConfig()
	if userIndex < 0 || userIndex >= len(cfg.Users) {
		return
	}
	user := cfg.Users[userIndex]
	ShowConfirmModal(u.rootPages, "重置流量",
		fmt.Sprintf("确定要将用户 '%s' 的累计流量清零吗？", user.Email),
		func() {
			_ = u.mgr.ResetUserTraffic(userIndex)
			u.renderTable()
		})
}

func (u *UI) refreshCurrentView() {
	u.renderHeader()
	if u.activeTab == 0 {
		u.renderTable()
	} else if u.activeTab == 3 {
		u.renderStatus()
	}
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

	// Tabs representation
	tab1 := "[gray][1] 👥 用户管理[-]"
	tab2 := "[gray][2] 🛡️ Reality配置[-]"
	tab3 := "[gray][3] ⚙️ VLESS服务[-]"
	tab4 := "[gray][4] ⚡ 运行状态[-]"

	switch u.activeTab {
	case 0:
		tab1 = "[black:yellow:b] [1] 👥 用户管理 [-:-:-]"
	case 1:
		tab2 = "[black:yellow:b] [2] 🛡️ Reality配置 [-:-:-]"
	case 2:
		tab3 = "[black:yellow:b] [3] ⚙️ VLESS服务 [-:-:-]"
	case 3:
		tab4 = "[black:yellow:b] [4] ⚡ 运行状态 [-:-:-]"
	}

	line1 := fmt.Sprintf("  [teal:b]XTUI[-] %s   %s   %s   %s   %s",
		statusBadge, tab1, tab2, tab3, tab4)

	line2 := fmt.Sprintf("  [gray]端口:[-] [yellow]%d[-] [gray]|[-] [gray]目标:[-] [aqua]%s[-] [gray]|[-] [gray]用户:[-] [white]%d[-] [gray]|[-] 实时网速: ↑ [lime]%s[-]  ↓ [lime]%s[-]",
		cfg.Server.Port, cfg.Server.Dest, len(cfg.Users), FormatSpeed(totalSpeedUp), FormatSpeed(totalSpeedDown))

	divider := "  " + strings.Repeat("─", 86)

	u.headerBar.SetText(fmt.Sprintf("%s\n%s\n%s", line1, line2, divider))
}

func (u *UI) renderFooter() {
	if u.activeTab == 0 {
		u.footerBar.SetText("  [yellow][1-4][-] 切换标签  [yellow][a][-] 添加用户  [yellow][e][-] 修改用户  [yellow][d][-] 删除  [yellow][s/Enter][-] 二维码/链接  [yellow][t][-] 启用/禁用  [yellow][r][-] 清零流量  [yellow][q][-] 退出")
	} else if u.activeTab == 1 || u.activeTab == 2 {
		u.footerBar.SetText("  [yellow][1-4][-] 切换标签  [yellow][Tab/Shift+Tab][-] 切换表单项  [yellow][Enter/Space][-] 选择/点击  [yellow][1][-] 返回用户列表  [yellow][q][-] 退出")
	} else {
		u.footerBar.SetText("  [yellow][1-4][-] 切换标签  [yellow][R][-] 重启代理内核  [yellow][r][-] 清零所有流量  [yellow][1][-] 返回用户列表  [yellow][q][-] 退出")
	}
}

func (u *UI) renderTable() {
	cfg := u.mgr.GetConfig()
	selectedRow, _ := u.table.GetSelection()

	u.table.Clear()

	headers := []string{"#", "用户名 / 邮箱", "状态", "↑ 上行", "↓ 下行", "累计使用", "流量配额", "实时速率 (↑/↓)", "到期时间"}
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
		quotaStr := "不限制"
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

	if len(cfg.Users) > 0 {
		if selectedRow <= 0 {
			selectedRow = 1
		} else if selectedRow > len(cfg.Users) {
			selectedRow = len(cfg.Users)
		}
		u.table.Select(selectedRow, 0)
	}
}
