package main

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// TestCollect checks that a real snapshot returns sane, non-zero values on
// this machine (temperature is allowed to be absent).
func TestCollect(t *testing.T) {
	c := NewCollector()
	time.Sleep(150 * time.Millisecond) // let the CPU sampler accumulate a delta
	s := c.Collect()

	t.Logf("CPU=%.1f%% RAM=%.1f%% (%s/%s) DISK=%.1f%% (%s/%s) NET ↓%.1f ↑%.1f KB/s TEMP=%v(%.1f°C) OS=%q up=%s",
		s.CPUPercent, s.RAMPercent, humanBytes(s.RAMUsed), humanBytes(s.RAMTotal),
		s.DiskPct, humanBytes(s.DiskUsed), humanBytes(s.DiskTotal),
		s.NetRecvKBs, s.NetSentKBs, s.HasTemp, s.TempC, s.OS, formatDuration(s.Uptime))

	if s.RAMTotal == 0 {
		t.Error("RAMTotal is 0 — memory reading failed")
	}
	if s.DiskTotal == 0 {
		t.Error("DiskTotal is 0 — disk reading failed")
	}
	if s.OS == "" {
		t.Error("OS is empty — host info failed")
	}
}

// TestTankRender verifies the aquarium produces exactly h rows, each exactly w
// visible columns wide, after stats and a few animation frames.
func TestTankRender(t *testing.T) {
	tank := NewTank()
	tank.Resize(60, 12)
	tank.SetStats(90, 70, 300) // high CPU + RAM + network

	for i := 0; i < 30; i++ {
		tank.Update(0.1)
	}

	out := tank.Render()
	lines := strings.Split(out, "\n")
	if len(lines) != 12 {
		t.Fatalf("expected 12 rows, got %d", len(lines))
	}
	for i, ln := range lines {
		if w := utf8.RuneCountInString(stripANSI(ln)); w != 60 {
			t.Errorf("row %d visible width = %d, want 60", i, w)
		}
	}
	if len(tank.fish) < 3 {
		t.Errorf("expected the school to be populated, got %d fish", len(tank.fish))
	}
	t.Logf("fish=%d bubbles=%d\nsample row: %q", len(tank.fish), len(tank.bubbles),
		lipgloss.NewStyle().Render(lines[6]))
}

// TestDashboardView renders a full frame and prints it so the layout can be
// eyeballed. It also asserts the dashboard fills exactly the terminal height.
func TestDashboardView(t *testing.T) {
	m := newModel()
	m.w, m.h, m.ready = 84, 24, true
	m.stats = Stats{
		CPUPercent: 42.5, RAMPercent: 63.0,
		RAMUsed: 9_600_000_000, RAMTotal: 16_300_000_000,
		DiskPct: 51.9, DiskUsed: 130_000_000_000, DiskTotal: 251_000_000_000,
		NetRecvKBs: 850, NetSentKBs: 120,
		HasTemp: false, Uptime: 250 * time.Hour, OS: "Microsoft Windows 11 Pro 25H2",
	}
	m.tank.SetStats(m.stats.CPUPercent, m.stats.RAMPercent, m.stats.NetRecvKBs+m.stats.NetSentKBs)
	for i := 0; i < 25; i++ {
		m.tank.Update(0.1)
	}

	out := m.View()
	lines := strings.Split(out, "\n")
	if len(lines) != 24 {
		t.Errorf("dashboard height = %d lines, want 24", len(lines))
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != 84 {
			t.Errorf("row %d display width = %d, want 84", i, w)
		}
	}
}

// stripANSI removes escape sequences so we can measure real column width.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm'):
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
