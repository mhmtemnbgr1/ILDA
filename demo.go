package main

import (
	"math"
	"time"
)

// demoModel builds a model filled with synthetic data (no real machine info),
// used for screenshots: system-critters --demo [WxH] [tab 1-5] [theme]
func demoModel() model {
	m := newModel()
	for i := 0; i < 90; i++ {
		f := float64(i)
		s := demoStats(f)
		m.record(s)
		m.stats = s
	}
	m.stats = demoStats(90)
	m.tank.SetStats(m.stats.CPUPercent, m.stats.RAMPercent, m.stats.NetRecvKBs+m.stats.NetSentKBs)
	for i := 0; i < 80; i++ {
		m.tank.Update(0.1)
	}
	return m
}

func demoStats(t float64) Stats {
	wave := func(period, amp, base float64) float64 {
		return math.Max(0, math.Min(100, base+amp*math.Sin(t/period)+amp/3*math.Sin(t/(period/3.1))))
	}
	cores := make([]float64, 8)
	cpu := wave(9, 22, 38)
	for i := range cores {
		cores[i] = math.Max(1, math.Min(100, cpu+18*math.Sin(t/5+float64(i)*1.3)))
	}
	return Stats{
		CPUPercent: cpu, Cores: cores, CoreCount: 8, CPUModel: "Demo CPU 8-Core @ 3.4 GHz", CPUFreqMHz: 3400,
		RAMPercent: 58 + 4*math.Sin(t/20), RAMUsed: 9_300_000_000, RAMTotal: 16_000_000_000,
		SwapPct: 8, SwapUsed: 600_000_000, SwapTotal: 8_000_000_000,
		DiskPct: 62, DiskUsed: 310_000_000_000, DiskTotal: 500_000_000_000,
		DiskReadKBs:  math.Max(0, 900+800*math.Sin(t/4)),
		DiskWriteKBs: math.Max(0, 600+700*math.Sin(t/6+1)),
		Disks: []DiskPart{
			{"C:\\", "NTFS", 62, 310_000_000_000, 500_000_000_000},
			{"D:\\", "NTFS", 34, 340_000_000_000, 1_000_000_000_000},
		},
		NetRecvKBs: math.Max(0, 1400+1300*math.Sin(t/7)), NetSentKBs: math.Max(0, 260+240*math.Sin(t/5+2)),
		NetTotalRecv: 1_450_000_000, NetTotalSent: 210_000_000,
		Ifaces: []IfaceStat{
			{Name: "Ethernet", RecvKBs: math.Max(0, 1400+1300*math.Sin(t/7)), SentKBs: math.Max(0, 260+240*math.Sin(t/5+2)),
				TotalRecv: 1_450_000_000, TotalSent: 210_000_000, IPs: []string{"192.0.2.10"}},
			{Name: "Wi-Fi", RecvKBs: 0, SentKBs: 0, TotalRecv: 12_000_000, TotalSent: 3_000_000, IPs: []string{"192.0.2.11"}},
		},
		HasTemp: true, TempC: 52 + cpu/6,
		Procs: []ProcStat{
			{4120, "browser", 9.4, 4.1, 650_000_000},
			{2210, "code-editor", 6.1, 3.2, 520_000_000},
			{880, "game-launcher", 3.8, 2.4, 380_000_000},
			{3340, "music-player", 2.2, 1.1, 170_000_000},
			{1500, "terminal", 1.4, 0.6, 95_000_000},
			{720, "system", 1.1, 0.3, 48_000_000},
			{5012, "chat-app", 0.9, 1.8, 290_000_000},
			{990, "backup-agent", 0.4, 0.5, 80_000_000},
		},
		ProcCount: 8,
		GPU:       GPUStat{OK: true, Name: "Demo GPU 8GB", Util: 31, TempC: 56, MemUsed: 2_100_000_000, MemTotal: 8_000_000_000},
		Battery:   BatteryStat{Present: true, Pct: 78, Charging: true},
		Uptime:    52 * time.Hour,
		OS:        "Demo OS 11", Kernel: "10.0", Hostname: "demo-pc",
	}
}
