package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	frameRate = 100 * time.Millisecond // animation tick (~10 fps)
	histLen   = 240                    // samples kept for charts
)

type viewID int

const (
	vOverview viewID = iota
	vProcs
	vNet
	vDisk
	vSys
	numViews
)

var viewNames = [numViews]string{"Genel", "Süreçler", "Ağ", "Disk", "Sistem"}

var sortNames = [4]string{"CPU", "BELLEK", "PID", "İSİM"}

type frameMsg time.Time
type statsMsg Stats
type clearToastMsg struct{}

type history struct {
	CPU, RAM, Swap, DiskR, DiskW, GPU []float64
}

type ifHist struct{ Down, Up []float64 }

type model struct {
	w, h  int
	ready bool

	stats     Stats
	hist      history
	ifHist    map[string]*ifHist // "" = aggregate of all interfaces
	collector *Collector
	tank      *Tank
	lastFrame time.Time

	cfg    Config
	view   viewID
	paused bool

	sortBy  int
	procOff int
	iface   string // "" = all interfaces

	toast string
}

func newModel() model {
	cfg := loadConfig()
	th = themes[themeIndex(cfg.Theme)]
	m := model{
		cfg:       cfg,
		collector: NewCollector(),
		tank:      NewTank(),
		lastFrame: time.Now(),
		ifHist:    map[string]*ifHist{},
		iface:     cfg.Iface,
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(frameTick(), collectNow(m.collector))
}

func frameTick() tea.Cmd {
	return tea.Tick(frameRate, func(t time.Time) tea.Msg { return frameMsg(t) })
}

func collectNow(c *Collector) tea.Cmd {
	return func() tea.Msg { return statsMsg(c.Collect()) }
}

func (m model) scheduleStats() tea.Cmd {
	c := m.collector
	return tea.Tick(time.Duration(m.cfg.RefreshMs)*time.Millisecond,
		func(time.Time) tea.Msg { return statsMsg(c.Collect()) })
}

func push(s *[]float64, v float64) {
	*s = append(*s, v)
	if len(*s) > histLen {
		*s = (*s)[len(*s)-histLen:]
	}
}

func (m *model) record(s Stats) {
	push(&m.hist.CPU, s.CPUPercent)
	push(&m.hist.RAM, s.RAMPercent)
	push(&m.hist.Swap, s.SwapPct)
	push(&m.hist.DiskR, s.DiskReadKBs)
	push(&m.hist.DiskW, s.DiskWriteKBs)
	push(&m.hist.GPU, s.GPU.Util)
	rec := func(key string, down, up float64) {
		h := m.ifHist[key]
		if h == nil {
			h = &ifHist{}
			m.ifHist[key] = h
		}
		push(&h.Down, down)
		push(&h.Up, up)
	}
	rec("", s.NetRecvKBs, s.NetSentKBs)
	for _, i := range s.Ifaces {
		rec(i.Name, i.RecvKBs, i.SentKBs)
	}
}

func (m *model) setToast(msg string) tea.Cmd {
	m.toast = msg
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearToastMsg{} })
}

func (m *model) syncWants() { m.collector.SetWantProcs(m.view == vProcs) }

func (m *model) cycleIface() {
	names := []string{""}
	for _, i := range m.stats.Ifaces {
		names = append(names, i.Name)
	}
	next := 0
	for k, n := range names {
		if n == m.iface {
			next = (k + 1) % len(names)
			break
		}
	}
	m.iface = names[next]
	m.cfg.Iface = m.iface
	m.cfg.save()
}

func (m model) exportSnapshot() (string, error) {
	name := fmt.Sprintf("system-critters-%s.json", time.Now().Format("20060102-150405"))
	b, err := json.MarshalIndent(struct {
		Time  time.Time
		Stats Stats
	}{time.Now(), m.stats}, "", "  ")
	if err != nil {
		return "", err
	}
	return name, os.WriteFile(name, b, 0o644)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch k := msg.String(); k {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "tab", "right":
			m.view = (m.view + 1) % numViews
			m.syncWants()
		case "shift+tab", "left":
			m.view = (m.view + numViews - 1) % numViews
			m.syncWants()
		case "1", "2", "3", "4", "5":
			m.view = viewID(k[0] - '1')
			m.syncWants()
		case "t":
			idx := (themeIndex(th.Name) + 1) % len(themes)
			th = themes[idx]
			m.cfg.Theme = th.Name
			m.cfg.save()
			m.tank.RefreshTheme()
			return m, m.setToast("Tema: " + th.Name)
		case "p", " ":
			m.paused = !m.paused
		case "f":
			m.tank.Feed()
		case "+", "=":
			m.tank.AdjustFish(1)
		case "-", "_":
			m.tank.AdjustFish(-1)
		case "s":
			m.sortBy = (m.sortBy + 1) % len(sortNames)
			m.procOff = 0
		case "j", "down":
			m.procOff = min(m.procOff+1, max(0, len(m.stats.Procs)-1))
		case "k", "up":
			m.procOff = max(0, m.procOff-1)
		case "pgdown":
			m.procOff = min(m.procOff+10, max(0, len(m.stats.Procs)-1))
		case "pgup":
			m.procOff = max(0, m.procOff-10)
		case "n":
			m.cycleIface()
		case "e":
			if name, err := m.exportSnapshot(); err != nil {
				return m, m.setToast("Kaydedilemedi: " + err.Error())
			} else {
				return m, m.setToast("📸 Kaydedildi: " + name)
			}
		}
	case tea.WindowSizeMsg:
		m.w, m.h, m.ready = msg.Width, msg.Height, true
	case clearToastMsg:
		m.toast = ""
	case statsMsg:
		if !m.paused {
			m.stats = Stats(msg)
			m.record(m.stats)
			m.tank.SetStats(m.stats.CPUPercent, m.stats.RAMPercent, m.stats.NetRecvKBs+m.stats.NetSentKBs)
		}
		return m, m.scheduleStats()
	case frameMsg:
		now := time.Now()
		dt := now.Sub(m.lastFrame).Seconds()
		m.lastFrame = now
		if dt > 0.5 {
			dt = 0.5 // avoid a huge jump after the terminal was hidden
		}
		if !m.paused {
			m.tank.Update(dt)
		}
		return m, frameTick()
	}
	return m, nil
}
