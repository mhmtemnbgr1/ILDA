package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
)

// Stats is a snapshot of the machine's current state.
type Stats struct {
	CPUPercent float64
	RAMPercent float64
	RAMUsed    uint64
	RAMTotal   uint64
	DiskPct    float64
	DiskUsed   uint64
	DiskTotal  uint64
	NetRecvKBs float64
	NetSentKBs float64
	TempC      float64
	HasTemp    bool
	Uptime     time.Duration
	OS         string
	Hostname   string
}

// Collector holds the state needed to compute deltas between snapshots
// (e.g. network throughput) across calls to Collect.
type Collector struct {
	lastRecv uint64
	lastSent uint64
	lastTime time.Time
	primed   bool
}

// NewCollector primes the CPU sampler so the first real reading is meaningful.
func NewCollector() *Collector {
	_, _ = cpu.Percent(0, false) // prime: cpu.Percent(0) measures since the previous call
	return &Collector{}
}

func rootPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\"
	}
	return "/"
}

// Collect gathers a fresh snapshot. Every field degrades gracefully: if a
// source is unavailable the field simply stays at its zero value.
func (c *Collector) Collect() Stats {
	s := Stats{}

	if p, err := cpu.Percent(0, false); err == nil && len(p) > 0 {
		s.CPUPercent = p[0]
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		s.RAMPercent = vm.UsedPercent
		s.RAMUsed = vm.Used
		s.RAMTotal = vm.Total
	}
	if du, err := disk.Usage(rootPath()); err == nil {
		s.DiskPct = du.UsedPercent
		s.DiskUsed = du.Used
		s.DiskTotal = du.Total
	}
	if io, err := net.IOCounters(false); err == nil && len(io) > 0 {
		now := time.Now()
		recv, sent := io[0].BytesRecv, io[0].BytesSent
		if c.primed {
			if dt := now.Sub(c.lastTime).Seconds(); dt > 0 {
				if recv >= c.lastRecv {
					s.NetRecvKBs = float64(recv-c.lastRecv) / dt / 1024
				}
				if sent >= c.lastSent {
					s.NetSentKBs = float64(sent-c.lastSent) / dt / 1024
				}
			}
		}
		c.lastRecv, c.lastSent, c.lastTime, c.primed = recv, sent, now, true
	}
	if temps, err := sensors.SensorsTemperatures(); err == nil {
		if t, ok := pickTemp(temps); ok {
			s.TempC = t
			s.HasTemp = true
		}
	}
	if info, err := host.Info(); err == nil {
		s.Uptime = time.Duration(info.Uptime) * time.Second
		s.OS = strings.TrimSpace(fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion))
		s.Hostname = info.Hostname
	}
	return s
}

// pickTemp chooses the most representative CPU temperature from whatever
// sensors the OS exposes. Temperature reporting is notoriously spotty
// (often empty on Windows without extra drivers), so this is best-effort.
func pickTemp(temps []sensors.TemperatureStat) (float64, bool) {
	preferred := []string{"coretemp", "package", "cpu", "k10temp", "core", "tctl"}
	for _, want := range preferred {
		for _, t := range temps {
			if t.Temperature > 0 && strings.Contains(strings.ToLower(t.SensorKey), want) {
				return t.Temperature, true
			}
		}
	}
	for _, t := range temps {
		if t.Temperature > 0 {
			return t.Temperature, true
		}
	}
	return 0, false
}
