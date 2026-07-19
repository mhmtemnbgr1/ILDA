package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	frameRate = 100 * time.Millisecond // animation tick (~10 fps)
	statsRate = 1 * time.Second        // how often we re-read the machine
)

type frameMsg time.Time
type statsMsg Stats

type model struct {
	w, h      int
	ready     bool
	stats     Stats
	collector *Collector
	tank      *Tank
	lastFrame time.Time
}

func newModel() model {
	return model{
		collector: NewCollector(),
		tank:      NewTank(),
		lastFrame: time.Now(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(frameTick(), collectNow(m.collector))
}

func frameTick() tea.Cmd {
	return tea.Tick(frameRate, func(t time.Time) tea.Msg { return frameMsg(t) })
}

// collectNow reads stats immediately (used once at startup).
func collectNow(c *Collector) tea.Cmd {
	return func() tea.Msg { return statsMsg(c.Collect()) }
}

// scheduleStats reads stats again after statsRate.
func scheduleStats(c *Collector) tea.Cmd {
	return tea.Tick(statsRate, func(t time.Time) tea.Msg { return statsMsg(c.Collect()) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.w, m.h, m.ready = msg.Width, msg.Height, true
	case statsMsg:
		m.stats = Stats(msg)
		m.tank.SetStats(m.stats.CPUPercent, m.stats.RAMPercent, m.stats.NetRecvKBs+m.stats.NetSentKBs)
		return m, scheduleStats(m.collector)
	case frameMsg:
		now := time.Now()
		dt := now.Sub(m.lastFrame).Seconds()
		m.lastFrame = now
		if dt > 0.5 {
			dt = 0.5 // avoid a huge jump after the terminal was hidden
		}
		m.tank.Update(dt)
		return m, frameTick()
	}
	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "starting system-critters…"
	}
	gw := clampInt(m.w/2, 12, 46)

	title := titleStyle.Render("🦀 system-critters")
	info := labelStyle.Render(fmt.Sprintf("⏱ %s · %s", formatDuration(m.stats.Uptime), m.stats.OS))
	gap := m.w - lipgloss.Width(title) - lipgloss.Width(info)
	if gap < 1 {
		gap = 1
	}
	line1 := title + strings.Repeat(" ", gap) + info

	cpu := statLine("CPU ", gw, m.stats.CPUPercent, fmt.Sprintf("%5.1f%%", m.stats.CPUPercent)) +
		"   " + labelStyle.Render("TEMP") + " " + tempString(m.stats)
	ram := statLine("RAM ", gw, m.stats.RAMPercent,
		fmt.Sprintf("%s / %s", humanBytes(m.stats.RAMUsed), humanBytes(m.stats.RAMTotal)))
	dsk := statLine("DISK", gw, m.stats.DiskPct,
		fmt.Sprintf("%s / %s", humanBytes(m.stats.DiskUsed), humanBytes(m.stats.DiskTotal)))
	net := labelStyle.Render("NET ") + " " + fmt.Sprintf("↓ %s/s   ↑ %s/s",
		humanBytes(uint64(m.stats.NetRecvKBs*1024)), humanBytes(uint64(m.stats.NetSentKBs*1024)))

	header := strings.Join([]string{line1, cpu, ram, dsk, net}, "\n")
	footer := footerStyle.Render("q quit  ·  fish speed = CPU  ·  school size = RAM  ·  bubbles = network")

	tankBlockH := m.h - lipgloss.Height(header) - lipgloss.Height(footer)
	if tankBlockH < 3 {
		tankBlockH = 3
	}
	innerW := clampInt(m.w-2, 1, m.w)
	innerH := clampInt(tankBlockH-2, 1, tankBlockH)
	m.tank.Resize(innerW, innerH)

	tank := tankStyle.Render(m.tank.Render())
	return lipgloss.JoinVertical(lipgloss.Left, header, tank, footer)
}
