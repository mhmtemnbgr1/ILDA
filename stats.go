package main

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/shirou/gopsutil/v4/sensors"
)

type DiskPart struct {
	Mount, FS   string
	Pct         float64
	Used, Total uint64
}

type IfaceStat struct {
	Name                 string
	RecvKBs, SentKBs     float64
	TotalRecv, TotalSent uint64 // since the app started
	IPs                  []string
}

type ProcStat struct {
	PID    int32
	Name   string
	CPU    float64 // % of the whole machine
	MemPct float64
	RSS    uint64
}

type GPUStat struct {
	OK                bool
	Name              string
	Util, TempC       float64
	MemUsed, MemTotal uint64
}

type BatteryStat struct {
	Present  bool
	Pct      float64
	Charging bool
}

// Stats is a snapshot of the machine's current state.
type Stats struct {
	CPUPercent float64
	Cores      []float64
	CoreCount  int
	CPUModel   string
	CPUFreqMHz float64
	Load1      float64
	Load5      float64
	HasLoad    bool

	RAMPercent float64
	RAMUsed    uint64
	RAMTotal   uint64
	SwapPct    float64
	SwapUsed   uint64
	SwapTotal  uint64

	DiskPct      float64
	DiskUsed     uint64
	DiskTotal    uint64
	DiskReadKBs  float64
	DiskWriteKBs float64
	Disks        []DiskPart

	NetRecvKBs, NetSentKBs     float64
	NetTotalRecv, NetTotalSent uint64
	Ifaces                     []IfaceStat

	TempC   float64
	HasTemp bool

	Procs     []ProcStat
	ProcCount int

	GPU     GPUStat
	Battery BatteryStat

	Uptime   time.Duration
	OS       string
	Kernel   string
	Hostname string
}

type staticInfo struct {
	cpuModel string
	freq     float64
	os       string
	kernel   string
	hostname string
	boot     time.Time
}

type ifPrev struct{ recv, sent, baseRecv, baseSent uint64 }

// Collector holds the state needed to compute deltas between snapshots.
type Collector struct {
	mu     sync.Mutex
	static staticInfo

	ifaces   map[string]*ifPrev
	lastTime time.Time
	primed   bool

	lastDiskRead, lastDiskWrite uint64
	diskPrimed                  bool
	diskTime                    time.Time

	ipCache   map[string][]string
	ipTime    time.Time
	procs     map[int32]*process.Process
	procCache []ProcStat
	procTime  time.Time
	wantProcs atomic.Bool

	gpuCache   GPUStat
	gpuTime    time.Time
	gpuMissing bool
}

// NewCollector primes the CPU samplers and loads slow, static facts in the
// background so the first frame is not delayed.
func NewCollector() *Collector {
	_, _ = cpu.Percent(0, false)
	_, _ = cpu.Percent(0, true)
	c := &Collector{
		ifaces: map[string]*ifPrev{},
		procs:  map[int32]*process.Process{},
	}
	go c.loadStatic()
	return c
}

func (c *Collector) loadStatic() {
	var si staticInfo
	if info, err := host.Info(); err == nil {
		si.os = strings.TrimSpace(fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion))
		si.kernel = info.KernelVersion
		si.hostname = info.Hostname
		si.boot = time.Unix(int64(info.BootTime), 0)
	}
	if ci, err := cpu.Info(); err == nil && len(ci) > 0 {
		si.cpuModel = strings.TrimSpace(ci[0].ModelName)
		si.freq = ci[0].Mhz
	}
	c.mu.Lock()
	c.static = si
	c.mu.Unlock()
}

// SetWantProcs toggles the (comparatively expensive) process scan.
func (c *Collector) SetWantProcs(v bool) { c.wantProcs.Store(v) }

func rootPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\"
	}
	return "/"
}

func isLoopback(name string) bool {
	n := strings.ToLower(name)
	return n == "lo" || strings.Contains(n, "loopback")
}

// Collect gathers a fresh snapshot. Every field degrades gracefully.
func (c *Collector) Collect() Stats {
	s := Stats{}
	now := time.Now()

	c.mu.Lock()
	st := c.static
	c.mu.Unlock()
	s.CPUModel, s.CPUFreqMHz, s.OS, s.Kernel, s.Hostname = st.cpuModel, st.freq, st.os, st.kernel, st.hostname
	if !st.boot.IsZero() {
		s.Uptime = now.Sub(st.boot)
	}

	if p, err := cpu.Percent(0, false); err == nil && len(p) > 0 {
		s.CPUPercent = p[0]
	}
	if p, err := cpu.Percent(0, true); err == nil {
		s.Cores = p
	}
	s.CoreCount = len(s.Cores)
	if runtime.GOOS != "windows" {
		if l, err := load.Avg(); err == nil {
			s.Load1, s.Load5, s.HasLoad = l.Load1, l.Load5, true
		}
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		s.RAMPercent, s.RAMUsed, s.RAMTotal = vm.UsedPercent, vm.Used, vm.Total
	}
	if sw, err := mem.SwapMemory(); err == nil {
		s.SwapPct, s.SwapUsed, s.SwapTotal = sw.UsedPercent, sw.Used, sw.Total
	}

	c.collectDisks(&s, now)
	c.collectNet(&s, now)

	if temps, err := sensors.SensorsTemperatures(); err == nil {
		if t, ok := pickTemp(temps); ok {
			s.TempC, s.HasTemp = t, true
		}
	}

	if c.wantProcs.Load() {
		c.refreshProcs(now)
	}
	s.Procs = c.procCache
	s.ProcCount = len(c.procCache)

	if !c.gpuMissing && now.Sub(c.gpuTime) > 3*time.Second {
		first := c.gpuTime.IsZero()
		c.gpuTime = now
		c.gpuCache = readGPU()
		if first && !c.gpuCache.OK {
			c.gpuMissing = true // no NVIDIA tooling: don't keep probing
		}
	}
	s.GPU = c.gpuCache
	s.Battery = readBattery()
	return s
}

func (c *Collector) collectDisks(s *Stats, now time.Time) {
	if du, err := disk.Usage(rootPath()); err == nil {
		s.DiskPct, s.DiskUsed, s.DiskTotal = du.UsedPercent, du.Used, du.Total
	}
	if parts, err := disk.Partitions(false); err == nil {
		seen := map[string]bool{}
		for _, p := range parts {
			if seen[p.Mountpoint] || len(s.Disks) >= 12 {
				continue
			}
			seen[p.Mountpoint] = true
			if du, err := disk.Usage(p.Mountpoint); err == nil && du.Total > 0 {
				s.Disks = append(s.Disks, DiskPart{p.Mountpoint, p.Fstype, du.UsedPercent, du.Used, du.Total})
			}
		}
	}
	if io, err := disk.IOCounters(); err == nil {
		var r, w uint64
		for _, d := range io {
			r += d.ReadBytes
			w += d.WriteBytes
		}
		if c.diskPrimed {
			if dt := now.Sub(c.diskTime).Seconds(); dt > 0 {
				if r >= c.lastDiskRead {
					s.DiskReadKBs = float64(r-c.lastDiskRead) / dt / 1024
				}
				if w >= c.lastDiskWrite {
					s.DiskWriteKBs = float64(w-c.lastDiskWrite) / dt / 1024
				}
			}
		}
		c.lastDiskRead, c.lastDiskWrite, c.diskPrimed, c.diskTime = r, w, true, now
	}
}

func (c *Collector) collectNet(s *Stats, now time.Time) {
	if c.ipCache == nil || now.Sub(c.ipTime) > 10*time.Second {
		c.ipTime = now
		c.ipCache = map[string][]string{}
		if ifs, err := net.Interfaces(); err == nil {
			for _, i := range ifs {
				for _, a := range i.Addrs {
					ip := a.Addr
					if k := strings.Index(ip, "/"); k > 0 {
						ip = ip[:k]
					}
					c.ipCache[i.Name] = append(c.ipCache[i.Name], ip)
				}
			}
		}
	}
	io, err := net.IOCounters(true)
	if err != nil {
		return
	}
	dt := now.Sub(c.lastTime).Seconds()
	for _, n := range io {
		if isLoopback(n.Name) {
			continue
		}
		p, ok := c.ifaces[n.Name]
		if !ok {
			p = &ifPrev{recv: n.BytesRecv, sent: n.BytesSent, baseRecv: n.BytesRecv, baseSent: n.BytesSent}
			c.ifaces[n.Name] = p
		}
		is := IfaceStat{Name: n.Name, IPs: c.ipCache[n.Name]}
		if c.primed && dt > 0 {
			if n.BytesRecv >= p.recv {
				is.RecvKBs = float64(n.BytesRecv-p.recv) / dt / 1024
			}
			if n.BytesSent >= p.sent {
				is.SentKBs = float64(n.BytesSent-p.sent) / dt / 1024
			}
		}
		p.recv, p.sent = n.BytesRecv, n.BytesSent
		if n.BytesRecv >= p.baseRecv {
			is.TotalRecv = n.BytesRecv - p.baseRecv
		}
		if n.BytesSent >= p.baseSent {
			is.TotalSent = n.BytesSent - p.baseSent
		}
		if n.BytesRecv+n.BytesSent == 0 && len(is.IPs) == 0 {
			continue
		}
		s.NetRecvKBs += is.RecvKBs
		s.NetSentKBs += is.SentKBs
		s.NetTotalRecv += is.TotalRecv
		s.NetTotalSent += is.TotalSent
		s.Ifaces = append(s.Ifaces, is)
	}
	sort.Slice(s.Ifaces, func(i, j int) bool { return s.Ifaces[i].Name < s.Ifaces[j].Name })
	c.lastTime, c.primed = now, true
}

func (c *Collector) refreshProcs(now time.Time) {
	if now.Sub(c.procTime) < 2*time.Second && c.procCache != nil {
		return
	}
	c.procTime = now
	list, err := process.Processes()
	if err != nil {
		return
	}
	ncpu := float64(max(1, runtime.NumCPU()))
	var total uint64
	if vm, err := mem.VirtualMemory(); err == nil {
		total = vm.Total
	}
	alive := make(map[int32]bool, len(list))
	out := make([]ProcStat, 0, len(list))
	for _, p := range list {
		if p.Pid == 0 {
			continue
		}
		alive[p.Pid] = true
		if old, ok := c.procs[p.Pid]; ok {
			p = old
		} else {
			c.procs[p.Pid] = p
		}
		ps := ProcStat{PID: p.Pid}
		ps.Name, _ = p.Name()
		if ps.Name == "" {
			ps.Name = "?"
		}
		if pc, err := p.Percent(0); err == nil {
			ps.CPU = pc / ncpu
		}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			ps.RSS = mi.RSS
			if total > 0 {
				ps.MemPct = float64(mi.RSS) / float64(total) * 100
			}
		}
		out = append(out, ps)
	}
	for pid := range c.procs {
		if !alive[pid] {
			delete(c.procs, pid)
		}
	}
	c.procCache = out
}

// pickTemp chooses the most representative CPU temperature (best effort).
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
