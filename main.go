package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// printOnce renders static frames to stdout with full colour and exits. It is
// handy for screenshots and quick checks.
//
//	system-critters --once [WxH] [tab 1-5] [theme]   (real data)
//	system-critters --demo [WxH] [tab 1-5] [theme]   (synthetic data)
func printOnce(args []string, demo bool) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	w, h := 110, 32
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%dx%d", &w, &h)
	}
	only := -1
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil && n >= 1 && n <= int(numViews) {
			only = n - 1
		}
	}
	var m model
	if demo {
		m = demoModel()
		m.iface = ""
	} else {
		m = newModel()
	}
	if len(args) > 2 {
		th = themes[themeIndex(args[2])]
		m.tank.RefreshTheme()
	}
	if !demo {
		m.collector.SetWantProcs(true)
		time.Sleep(600 * time.Millisecond)
		m.stats = m.collector.Collect()
		for i := 0; i < 4; i++ {
			time.Sleep(700 * time.Millisecond)
			m.stats = m.collector.Collect()
			m.record(m.stats)
		}
		m.tank.SetStats(m.stats.CPUPercent, m.stats.RAMPercent, m.stats.NetRecvKBs+m.stats.NetSentKBs)
		for i := 0; i < 60; i++ {
			m.tank.Update(0.1)
		}
	}
	m.w, m.h, m.ready = w, h, true
	for v := viewID(0); v < numViews; v++ {
		if only >= 0 && int(v) != only {
			continue
		}
		m.view = v
		fmt.Println(m.View())
	}
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--once" || os.Args[1] == "--demo") {
		printOnce(os.Args[2:], os.Args[1] == "--demo")
		return
	}
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "system-critters error:", err)
		os.Exit(1)
	}
}
