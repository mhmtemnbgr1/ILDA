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

// fakeModel builds a model with plausible data and history for rendering tests.
func fakeModel(w, h int) model {
	m := newModel()
	m.w, m.h, m.ready = w, h, true
	m.stats = Stats{
		CPUPercent: 42.5, RAMPercent: 63.0, CoreCount: 4, Cores: []float64{10, 40, 75, 95},
		RAMUsed: 9_600_000_000, RAMTotal: 16_300_000_000, SwapPct: 12,
		DiskPct: 51.9, DiskUsed: 130_000_000_000, DiskTotal: 251_000_000_000,
		NetRecvKBs: 850, NetSentKBs: 120, Uptime: 250 * time.Hour,
		OS: "Microsoft Windows 11 Pro", Hostname: "pc", CPUModel: "Test CPU", CPUFreqMHz: 3200,
		Disks:  []DiskPart{{"C:\\", "NTFS", 51.9, 130_000_000_000, 251_000_000_000}},
		Ifaces: []IfaceStat{{Name: "Wi-Fi", RecvKBs: 850, SentKBs: 120, IPs: []string{"192.168.1.5"}}},
		Procs:  []ProcStat{{1, "chrome.exe", 12.5, 3.2, 500_000_000}, {2, "go.exe", 3, 1, 90_000_000}},
	}
	for i := 0; i < 60; i++ {
		m.record(m.stats)
	}
	m.tank.SetStats(m.stats.CPUPercent, m.stats.RAMPercent, 970)
	for i := 0; i < 25; i++ {
		m.tank.Update(0.1)
	}
	return m
}

// TestAllViewsFit renders every view at several terminal sizes and checks the
// frame is exactly w x h, which is what keeps the TUI from tearing.
func TestAllViewsFit(t *testing.T) {
	sizes := [][2]int{{120, 36}, {84, 24}, {60, 18}, {40, 12}, {30, 8}}
	for _, sz := range sizes {
		for v := viewID(0); v < numViews; v++ {
			for _, paused := range []bool{false, true} {
				m := fakeModel(sz[0], sz[1])
				m.view, m.paused = v, paused
				m.toast = "test"
				lines := strings.Split(m.View(), "\n")
				if len(lines) != sz[1] {
					t.Errorf("view %d @%v: %d lines, want %d", v, sz, len(lines), sz[1])
				}
				for i, ln := range lines {
					if w := lipgloss.Width(ln); w != sz[0] {
						t.Errorf("view %d @%v row %d width %d, want %d", v, sz, i, w, sz[0])
						break
					}
				}
			}
		}
	}
}

// TestThemes makes sure every theme renders and cycles without panicking.
func TestThemes(t *testing.T) {
	defer func() { th = themes[0] }()
	for _, tm := range themes {
		th = tm
		m := fakeModel(100, 30)
		m.tank.RefreshTheme()
		if out := m.View(); out == "" {
			t.Errorf("theme %s rendered nothing", tm.Name)
		}
	}
}

// TestFeedAndKeys exercises the interactive tank controls.
func TestFeedAndKeys(t *testing.T) {
	tank := NewTank()
	tank.Resize(60, 12)
	tank.SetStats(10, 50, 0)
	n := len(tank.fish)
	tank.AdjustFish(2)
	if len(tank.fish) != n+2 {
		t.Errorf("AdjustFish(2): got %d fish, want %d", len(tank.fish), n+2)
	}
	tank.Feed()
	if len(tank.foods) == 0 {
		t.Fatal("Feed dropped no food")
	}
	for i := 0; i < 400; i++ {
		tank.Update(0.1)
	}
	if len(tank.foods) != 0 {
		t.Errorf("food should be eaten or sunk, %d left", len(tank.foods))
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
