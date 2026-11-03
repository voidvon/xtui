package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"xtui/config"
	"xtui/core"
	"xtui/ipc"
	"xtui/ui"
)

const AppVersion = "1.0.0"

func getDefaultSockPath() string {
	return filepath.Join(os.TempDir(), "xtui.sock")
}

func printBanner(cfg *config.Config, sockPath string) {
	sni := "gateway.icloud.com"
	if len(cfg.Server.ServerNames) > 0 {
		sni = cfg.Server.ServerNames[0]
	}

	fmt.Printf("\n==============================================================\n")
	fmt.Printf("  ⚡ XTUI - VLESS + Reality Proxy Service (v%s)\n", AppVersion)
	fmt.Printf("--------------------------------------------------------------\n")
	fmt.Printf("  ● 服务状态 (Status):   \033[32m● 运行中 (RUNNING)\033[0m\n")
	fmt.Printf("  ● 监听地址 (Listen):   %s:%d\n", cfg.Server.Listen, cfg.Server.Port)
	fmt.Printf("  ● 节点公网 (Host):     %s\n", cfg.Server.PublicHost)
	fmt.Printf("  ● 目标网站 (Dest):     %s (SNI: %s)\n", cfg.Server.Dest, sni)
	fmt.Printf("  ● 注册用户 (Users):    %d 位\n", len(cfg.Users))
	fmt.Printf("  ● 管理接口 (IPC):      %s\n", sockPath)
	fmt.Printf("--------------------------------------------------------------\n")
	fmt.Printf("  💡 \033[33m管理配置面板: 执行 '%s ui' (或 '%s tui')\033[0m\n", os.Args[0], os.Args[0])
	fmt.Printf("  按 Ctrl+C 停止服务\n")
	fmt.Printf("==============================================================\n\n")
}

func main() {
	var configPath string
	var sockPath string
	var showVersion bool
	var isDaemonWorker bool

	flag.StringVar(&configPath, "c", "xtui.json", "Path to config file")
	flag.StringVar(&sockPath, "s", getDefaultSockPath(), "Path to IPC socket file")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&isDaemonWorker, "d", false, "Internal flag for background worker")
	flag.Usage = printHelp

	// Filter out subcommand before parsing flags
	var subcommand string
	var remainingArgs []string
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		subcommand = os.Args[1]
		remainingArgs = append([]string{os.Args[0]}, os.Args[2:]...)
	} else {
		remainingArgs = os.Args
	}

	_ = flag.CommandLine.Parse(remainingArgs[1:])

	if showVersion {
		fmt.Printf("XTUI version %s (Embedded Xray-core)\n", AppVersion)
		return
	}

	switch subcommand {
	case "ui", "tui", "menu", "config":
		runTUIMode(configPath, sockPath)
	case "status":
		runStatusMode(sockPath)
	case "stop":
		runStopMode(sockPath)
	case "restart":
		runRestartMode(sockPath)
	case "help", "-h", "--help":
		printHelp()
	default:
		// Default mode: run daemon/service
		runServiceMode(configPath, sockPath, isDaemonWorker)
	}
}

func printHelp() {
	fmt.Printf(`XTUI - 极简单文件 VLESS + Reality 代理管理工具 (v%s)

使用方法:
  xtui                     启动代理后台服务 (默认模式，常驻运行)
  xtui ui                  打开交互式 TUI 管理面板 (别名: tui, menu, config)
  xtui status              查看当前后台服务运行状态
  xtui restart             平滑重启后台代理服务
  xtui stop                停止后台代理服务

选项参数:
  -c <文件路径>            指定配置文件路径 (默认: ./xtui.json)
  -s <Socket路径>          指定 IPC Socket 路径 (默认: %s)
  -v                       查看版本信息
  -h                       查看帮助
`, AppVersion, getDefaultSockPath())
}

// runServiceMode runs the proxy service in the foreground or background (Service Mode).
func runServiceMode(configPath, sockPath string, isDaemonWorker bool) {
	if ipc.IsDaemonRunning(sockPath) {
		fmt.Printf("XTUI 服务已经在运行中 (Socket: %s)。\n", sockPath)
		fmt.Printf("执行 '%s ui' 打开管理面板，或执行 '%s restart' 重启服务。\n", os.Args[0], os.Args[0])
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	mgr := core.NewManager(cfg, configPath)
	if err := mgr.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动 Xray 内核失败: %v\n", err)
		os.Exit(1)
	}

	ipcSrv, err := ipc.StartIPCServer(sockPath, mgr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "启动 IPC 管理接口失败: %v\n", err)
		_ = mgr.Stop()
		os.Exit(1)
	}

	if !isDaemonWorker {
		printBanner(cfg, sockPath)
	}

	// Wait for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	if !isDaemonWorker {
		fmt.Println("\n正在停止 XTUI 服务并保存数据...")
	}

	_ = ipcSrv.Close()
	_ = mgr.Stop()

	if !isDaemonWorker {
		fmt.Println("XTUI 服务已安全停止。")
	}
}

// runTUIMode launches the interactive TUI management interface.
func runTUIMode(configPath, sockPath string) {
	// If daemon is not running, automatically spawn it in background
	if !ipc.IsDaemonRunning(sockPath) {
		fmt.Println("[XTUI] 检测到后台代理服务未运行，正在为您启动后台服务...")

		cmd := exec.Command(os.Args[0], "run", "-c", configPath, "-s", sockPath, "-d")
		cmd.Dir, _ = os.Getwd()
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // detach process group

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "启动后台服务失败: %v\n", err)
			os.Exit(1)
		}

		// Wait for socket to be ready
		ready := false
		for i := 0; i < 25; i++ {
			time.Sleep(100 * time.Millisecond)
			if ipc.IsDaemonRunning(sockPath) {
				ready = true
				break
			}
		}

		if !ready {
			fmt.Fprintf(os.Stderr, "连接后台服务超时，请先执行 '%s' 查看启动日志。\n", os.Args[0])
			os.Exit(1)
		}
	}

	// Connect to running daemon via IPC
	client, err := ipc.NewClient(sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接管理接口失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Launch TUI
	appUI := ui.NewUI(client)
	if err := appUI.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI 运行错误: %v\n", err)
	}

	// When TUI exits, the service keeps running!
	cfg := client.GetConfig()
	port := 4433
	if cfg != nil && cfg.Server.Port > 0 {
		port = cfg.Server.Port
	}

	fmt.Printf("\n[XTUI] 已退出 TUI 管理界面。\n")
	fmt.Printf("⚡ XTUI 代理服务仍在后台持续运行中 (端口: %d)\n", port)
	fmt.Printf("💡 随时可执行 '%s ui' 重新打开管理面板。\n\n", os.Args[0])
}

func runStatusMode(sockPath string) {
	if !ipc.IsDaemonRunning(sockPath) {
		fmt.Println("XTUI 服务状态: \033[31m● 未运行 (STOPPED)\033[0m")
		return
	}

	client, err := ipc.NewClient(sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询失败: %v\n", err)
		return
	}
	defer client.Close()

	cfg := client.GetConfig()
	uptime := time.Since(client.GetStartTime()).Truncate(time.Second)

	fmt.Println("\n==============================================================")
	fmt.Printf("  ⚡ XTUI 服务运行状态 (v%s)\n", AppVersion)
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("  ● 服务状态:   \033[32m● 运行中 (RUNNING)\033[0m")
	fmt.Printf("  ● 运行时间:   %s\n", uptime)
	fmt.Printf("  ● 监听端口:   %s:%d\n", cfg.Server.Listen, cfg.Server.Port)
	fmt.Printf("  ● 伪装目标:   %s\n", cfg.Server.Dest)
	fmt.Printf("  ● 节点公网:   %s\n", cfg.Server.PublicHost)
	fmt.Printf("  ● 注册用户:   %d 位\n", len(cfg.Users))
	fmt.Println("==============================================================")
	fmt.Println()
}

func runRestartMode(sockPath string) {
	if !ipc.IsDaemonRunning(sockPath) {
		fmt.Println("XTUI 服务未在运行。")
		return
	}
	client, err := ipc.NewClient(sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接失败: %v\n", err)
		return
	}
	defer client.Close()

	if err := client.Restart(); err != nil {
		fmt.Fprintf(os.Stderr, "重启失败: %v\n", err)
	} else {
		fmt.Println("XTUI 代理内核已平滑重启。")
	}
}

func runStopMode(sockPath string) {
	if !ipc.IsDaemonRunning(sockPath) {
		fmt.Println("XTUI 服务未在运行。")
		return
	}
	client, err := ipc.NewClient(sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接失败: %v\n", err)
		return
	}
	defer client.Close()

	if err := client.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "停止失败: %v\n", err)
	} else {
		fmt.Println("XTUI 服务已停止。")
	}
}
