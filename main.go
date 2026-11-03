package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"xtui/config"
	"xtui/core"
	"xtui/ui"
)

const AppVersion = "1.0.0"

func main() {
	configPath := flag.String("c", "xtui.json", "Path to config file")
	showVersion := flag.Bool("v", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("XTUI version %s (Embedded Xray-core)\n", AppVersion)
		return
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	mgr := core.NewManager(cfg, *configPath)

	// Start Xray core
	if err := mgr.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Xray core: %v\n", err)
		os.Exit(1)
	}

	// Setup signal handling for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	appUI := ui.NewUI(mgr)

	go func() {
		<-sigChan
		appUI.Stop()
		_ = mgr.Stop()
		os.Exit(0)
	}()

	// Run TUI event loop
	if err := appUI.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}

	// Clean stop upon normal exit
	_ = mgr.Stop()
}
