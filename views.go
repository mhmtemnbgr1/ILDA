package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ---- frame --------------------------------------------------------------

func (m model) View() string {
	if !m.ready {
		return "system-critters başlıyor…"
	}
	w, h := max(m.w, 20), max(m.h, 6)

	header := m.header(w)
	footer := m.footer(w)
	bodyH := h - lipgloss.Height(header) - 1
	var body string
	switch m.view {
	case vProcs:
		body = m.viewProcs(w, bodyH)
	case vNet:
		body = m.viewNet(w, bodyH)
	case vDisk:
		body = m.viewDisk(w, bodyH)
	case vSys:
		body = m.viewSys(w, bodyH)
	default:
		body = m.viewOverview(w, bodyH)
	}
	return fit(header+"\n"+fit(body, w, bodyH)+"\n"+footer, w, h)
}

func (m model) statusPill() string {
	load := max(m.stats.CPUPercent, m.stats.RAMPercent)
	switch {
	case m.paused:
		return pill(th.Dim, "⏸ DURAKLATILDI")
	case load >= 85:
		return pill(th.Bad, "● KRİTİK")
	case load >= 60:
		return pill(th.Warn, "● YOĞUN")
	default:
		return pill(th.Good, "● SAKİN")
	}
}

func (m model) header(w int) string {
	left := gradientText("🦀 SYSTEM-CRITTERS") + " " + m.statusPill()
	right := dim(m.stats.Hostname+" · "+m.stats.OS+" · ") + text("up "+formatDuration(m.stats.Uptime)) +
		dim(" · ") + text(time.Now().Format("15:04:05"))
	if lipgloss.Width(left)+lipgloss.Width(right)+2 > w {
		right = text("up "+formatDuration(m.stats.Uptime)) + dim(" · ") + text(time.Now().Format("15:04"))
	}
	if lipgloss.Width(left)+lipgloss.Width(right)+2 > w {
		right = ""
	}
	gap := max(1, w-lipgloss.Width(left)-lipgloss.Width(right))
	line1 := left + strings.Repeat(" ", gap) + right

	var tabs []string
	for i := viewID(0); i < numViews; i++ {
		label := fmt.Sprintf(" %d %s ", i+1, viewNames[i])
		if i == m.view {
			tabs = append(tabs, sty(th.Dark, th.Accent).Bold(true).Render(label))
		} else {
			tabs = append(tabs, dim(label))
		}
	}
	line2 := strings.Join(tabs, dim("│"))
	if m.toast != "" {
		t := bold(th.Accent2, m.toast)
		line2 += strings.Repeat(" ", max(2, w-lipgloss.Width(line2)-lipgloss.Width(t))) + t
	}
	return line1 + "\n" + line2
}

func (m model) footer(w int) string {
	items := []string{
		keyBadge("q", "çıkış"), keyBadge("Tab", "sekme"), keyBadge("t", "tema"),
		keyBadge("p", "duraklat"), keyBadge("e", "kaydet"),
	}
	switch m.view {
	case vOverview:
		items = append(items, keyBadge("f", "yemle"), keyBadge("+/-", "balık"))
	case vProcs:
		items = append(items, keyBadge("s", "sırala"), keyBadge("j/k", "kaydır"))
	case vNet:
		items = append(items, keyBadge("n", "arayüz"))
	}
	return padTo(strings.Join(items, "  "), w)
}

// ---- alerts -------------------------------------------------------------

func alerts(s Stats) []string {
	var out []string
	if s.CPUPercent >= 85 {
		out = append(out, fmt.Sprintf("CPU %.0f%%", s.CPUPercent))
	}
	if s.RAMPercent >= 90 {
		out = append(out, fmt.Sprintf("RAM %.0f%%", s.RAMPercent))
	}
	if s.DiskPct >= 90 {
		out = append(out, fmt.Sprintf("Disk %.0f%% dolu", s.DiskPct))
	}
	if s.HasTemp && s.TempC >= 85 {
		out = append(out, fmt.Sprintf("Sıcaklık %.0f°C", s.TempC))
	}
	if b := s.Battery; b.Present && !b.Charging && b.Pct <= 15 {
		out = append(out, fmt.Sprintf("Pil %.0f%%", b.Pct))
	}
	return out
}

// ---- overview -----------------------------------------------------------

func rateColor(f float64) lipgloss.Color { return accentAt(f) }

func borderFor(pct float64) lipgloss.Color {
	if pct >= 60 {
		return lerpColor(th.Border, levelColor(pct), 0.8)
	}
	return th.Border
}

func gaugeLine(iw int, pct float64) string {
	gw := max(4, iw-7)
	return gauge(gw, pct) + " " + bold(levelColor(pct), fmt.Sprintf("%5.1f%%", pct))
}

func (m model) cardCPU(cw, rows int) string {
	s, iw := m.stats, cw-4
	freq := ""
	if s.CPUFreqMHz > 0 {
		freq = fmt.Sprintf(" · %.1f GHz", s.CPUFreqMHz/1000)
	}
	lines := []string{
		gaugeLine(iw, s.CPUPercent),
		sparkline(m.hist.CPU, iw, 100, gradAt),
		dim("SICAKLIK ") + tempString(s) + dim(fmt.Sprintf(freq+" · %d çekirdek", s.CoreCount)),
	}
	return card("🧠 "+bold(th.Text, "CPU"), lines[:rows], cw, borderFor(s.CPUPercent))
}

func (m model) cardRAM(cw, rows int) string {
	s, iw := m.stats, cw-4
	lines := []string{
		gaugeLine(iw, s.RAMPercent),
		sparkline(m.hist.RAM, iw, 100, gradAt),
		text(humanBytes(s.RAMUsed)+" / "+humanBytes(s.RAMTotal)) + dim(fmt.Sprintf(" · takas %.0f%%", s.SwapPct)),
	}
	return card("💾 "+bold(th.Text, "RAM"), lines[:rows], cw, borderFor(s.RAMPercent))
}

func (m model) cardDisk(cw, rows int) string {
	s, iw := m.stats, cw-4
	lines := []string{
		gaugeLine(iw, s.DiskPct),
		text(humanBytes(s.DiskUsed) + " / " + humanBytes(s.DiskTotal)),
		dim("oku ") + text(humanRate(s.DiskReadKBs)) + dim(" · yaz ") + text(humanRate(s.DiskWriteKBs)),
	}
	return card("💿 "+bold(th.Text, "DİSK"), lines[:rows], cw, borderFor(s.DiskPct))
}

func (m model) cardNet(cw, rows int) string {
	s, iw := m.stats, cw-4
	h := m.ifHist[""]
	if h == nil {
		h = &ifHist{}
	}
	sw := max(4, iw-14)
	lines := []string{
		paint(th.Good, "↓ ") + padTo(bold(th.Text, humanRate(s.NetRecvKBs)), 12) + sparkline(h.Down, sw, -100, rateColor),
		paint(th.Accent2, "↑ ") + padTo(bold(th.Text, humanRate(s.NetSentKBs)), 12) + sparkline(h.Up, sw, -100, rateColor),
		dim("toplam ↓ ") + text(humanBytes(s.NetTotalRecv)) + dim("  ↑ ") + text(humanBytes(s.NetTotalSent)),
	}
	return card("🌐 "+bold(th.Text, "AĞ"), lines[:rows], cw, th.Border)
}

func (m model) viewOverview(w, h int) string {
	cards := []func(int, int) string{m.cardCPU, m.cardRAM, m.cardDisk, m.cardNet}
	cols := 2
	if w < 64 {
		cols = 1
	}
	gridRows := 4 / cols

	al := alerts(m.stats)
	alertH := 0
	if len(al) > 0 {
		alertH = 1
	}
	rows := 3
	for rows > 1 && gridRows*(rows+2)+alertH+6 > h {
		rows--
	}

	var parts []string
	for r := 0; r < gridRows; r++ {
		var line []string
		for c := 0; c < cols; c++ {
			cw := (w - (cols - 1)) / cols
			if c == cols-1 {
				cw = w - (cols-1)*(cw+1)
			}
			line = append(line, cards[r*cols+c](cw, rows))
			if c < cols-1 {
				line = append(line, " ")
			}
		}
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Top, line...))
	}
	if alertH > 0 {
		banner := sty(th.Dark, th.Bad).Bold(true).Render(" ⚠ UYARI: " + strings.Join(al, " · ") + " ")
		parts = append(parts, banner)
	}
	grid := strings.Join(parts, "\n")

	tankH := h - lipgloss.Height(grid)
	if tankH < 3 {
		return grid
	}
	m.tank.Resize(max(1, w-2), max(1, tankH-2))
	hp := m.tank.Happiness()
	heart := paint(lerpColor(th.Bad, th.Good, hp/100), fmt.Sprintf("♥ %.0f%%", hp))
	title := "🌊 " + bold(th.Text, "Akvaryum") + dim(" · mutluluk ") + heart + dim(fmt.Sprintf(" · %d balık", len(m.tank.fish)))
	tank := box(title, strings.Split(m.tank.Render(), "\n"), w, tankH, th.Border, 0)
	return grid + "\n" + tank
}

// ---- processes ----------------------------------------------------------

func (m model) viewProcs(w, h int) string {
	procs := append([]ProcStat(nil), m.stats.Procs...)
	sort.SliceStable(procs, func(i, j int) bool {
		a, b := procs[i], procs[j]
		switch m.sortBy {
		case 1:
			return a.RSS > b.RSS
		case 2:
			return a.PID < b.PID
		case 3:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return a.CPU > b.CPU
	})

	iw := w - 4
	showBar := iw >= 72
	fixed := 7 + 1 + 6 + 1 + 10 + 1 + 6 // pid, cpu, mem, mem%
	if showBar {
		fixed += 11
	}
	nameW := max(8, iw-fixed-1)

	hd := func(label string, width int, idx int, right bool) string {
		s := label
		if idx == m.sortBy {
			s += "▼"
		}
		cell := fmt.Sprintf("%-*s", width, s)
		if right {
			cell = fmt.Sprintf("%*s", width, s)
		}
		if idx == m.sortBy {
			return bold(th.Accent, cell)
		}
		return dim(cell)
	}
	head := hd("PID", 7, 2, true) + " " + hd("İSİM", nameW, 3, false) + " " + hd("CPU", 6, 0, true)
	if showBar {
		head += strings.Repeat(" ", 11)
	}
	head += " " + hd("BELLEK", 10, 1, true) + " " + dim(fmt.Sprintf("%6s", "BEL%"))

	lines := []string{head}
	if len(procs) == 0 {
		lines = append(lines, dim("süreçler toplanıyor…"))
	}
	visible := max(1, h-4)
	off := clampInt(m.procOff, 0, max(0, len(procs)-visible))
	for i := off; i < len(procs) && i < off+visible; i++ {
		p := procs[i]
		ln := dim(fmt.Sprintf("%7d", p.PID)) + " " + text(padTo(p.Name, nameW)) + " " +
			bold(levelColor(p.CPU*2), fmt.Sprintf("%5.1f%%", p.CPU))
		if showBar {
			ln += " " + gauge(10, clampF(p.CPU*2, 0, 100))
		}
		ln += " " + text(fmt.Sprintf("%10s", humanBytes(p.RSS))) + " " + dim(fmt.Sprintf("%5.1f%%", p.MemPct))
		lines = append(lines, ln)
	}
	for len(lines) < h-3 {
		lines = append(lines, "")
	}
	lines = append(lines, dim(fmt.Sprintf("%d süreç · sıralama: ", len(procs)))+bold(th.Accent, sortNames[m.sortBy])+
		dim(fmt.Sprintf(" · %d–%d gösteriliyor", min(off+1, len(procs)), min(off+visible, len(procs)))))
	return box("📋 "+bold(th.Text, "Süreçler"), lines, w, h, th.Border, 1)
}

// ---- network ------------------------------------------------------------

func (m model) selectedNet() (down, up float64, name string, hist *ifHist) {
	hist = m.ifHist[m.iface]
	if hist == nil {
		hist = &ifHist{}
	}
	if m.iface == "" {
		return m.stats.NetRecvKBs, m.stats.NetSentKBs, "tüm arayüzler", hist
	}
	for _, i := range m.stats.Ifaces {
		if i.Name == m.iface {
			return i.RecvKBs, i.SentKBs, i.Name, hist
		}
	}
	return 0, 0, m.iface + " (yok)", hist
}

func (m model) chartBox(title string, vals []float64, w, h int, floor float64, color lipgloss.Color) string {
	iw := w - 4
	ch := bigChart(vals, iw, max(1, h-2), -floor, func(f float64) lipgloss.Color { return lerpColor(color, th.Accent2, f) })
	return box(title, ch, w, h, th.Border, 1)
}

func (m model) viewNet(w, h int) string {
	down, up, name, hist := m.selectedNet()
	chartH := clampInt((h-4)/2, 4, 9)
	if w >= 100 {
		chartH = clampInt(h-8, 4, 7)
	}
	dTitle := paint(th.Good, "↓ ") + bold(th.Text, "İndirme ") + text(humanRate(down)) + dim(" · "+name)
	uTitle := paint(th.Accent2, "↑ ") + bold(th.Text, "Yükleme ") + text(humanRate(up))
	var charts string
	if w >= 100 {
		cw := (w - 1) / 2
		charts = lipgloss.JoinHorizontal(lipgloss.Top,
			m.chartBox(dTitle, hist.Down, cw, chartH+2, 100, th.Good), " ",
			m.chartBox(uTitle, hist.Up, w-cw-1, chartH+2, 100, th.Accent))
	} else {
		charts = m.chartBox(dTitle, hist.Down, w, chartH+2, 100, th.Good) + "\n" +
			m.chartBox(uTitle, hist.Up, w, chartH+2, 100, th.Accent)
	}

	iw := w - 4
	lines := []string{dim(fmt.Sprintf("%-18s %12s %12s %10s %10s  %s", "ARAYÜZ", "↓ hız", "↑ hız", "↓ toplam", "↑ toplam", "IP"))}
	for _, i := range m.stats.Ifaces {
		mark, nm := "  ", text(padTo(i.Name, 16))
		if i.Name == m.iface {
			mark, nm = bold(th.Accent, "▸ "), bold(th.Accent, padTo(i.Name, 16))
		}
		ip := ""
		if len(i.IPs) > 0 {
			ip = i.IPs[0]
		}
		lines = append(lines, mark+nm+" "+fmt.Sprintf("%12s %12s %10s %10s  ", humanRate(i.RecvKBs), humanRate(i.SentKBs),
			humanBytes(i.TotalRecv), humanBytes(i.TotalSent))+dim(ip))
	}
	_ = iw
	tableH := max(3, h-lipgloss.Height(charts))
	table := box("🌐 "+bold(th.Text, "Arayüzler")+dim(" · n ile seç"), lines, w, tableH, th.Border, 1)
	return charts + "\n" + table
}

// ---- disk ---------------------------------------------------------------

func (m model) viewDisk(w, h int) string {
	s := m.stats
	iw := w - 4
	var lines []string
	for _, d := range s.Disks {
		label := padTo(text(d.Mount), 10) + padTo(dim(d.FS), 7)
		tail := fmt.Sprintf(" %5.1f%%  %s / %s", d.Pct, humanBytes(d.Used), humanBytes(d.Total))
		gw := max(6, iw-17-lipgloss.Width(tail))
		lines = append(lines, label+gauge(gw, d.Pct)+bold(levelColor(d.Pct), fmt.Sprintf(" %5.1f%%", d.Pct))+
			text(fmt.Sprintf("  %s / %s", humanBytes(d.Used), humanBytes(d.Total))))
	}
	if len(lines) == 0 {
		lines = []string{dim("disk bilgisi okunamadı")}
	}
	listH := min(len(lines)+2, max(3, h-8))
	list := box("💿 "+bold(th.Text, "Diskler"), lines, w, listH, th.Border, 1)

	rest := h - listH
	chartH := clampInt(rest, 3, 12)
	rt := paint(th.Good, "↓ ") + bold(th.Text, "Okuma ") + text(humanRate(s.DiskReadKBs))
	wt := paint(th.Accent2, "↑ ") + bold(th.Text, "Yazma ") + text(humanRate(s.DiskWriteKBs))
	var io string
	if w >= 90 {
		cw := (w - 1) / 2
		io = lipgloss.JoinHorizontal(lipgloss.Top,
			m.chartBox(rt, m.hist.DiskR, cw, chartH, 100, th.Good), " ",
			m.chartBox(wt, m.hist.DiskW, w-cw-1, chartH, 100, th.Accent))
	} else {
		half := max(3, chartH/2)
		io = m.chartBox(rt, m.hist.DiskR, w, half, 100, th.Good) + "\n" +
			m.chartBox(wt, m.hist.DiskW, w, max(3, chartH-half), 100, th.Accent)
	}
	return list + "\n" + io
}

// ---- system -------------------------------------------------------------

func (m model) viewSys(w, h int) string {
	s := m.stats

	// Per-core bars in two columns.
	iwCores := 0
	wide := w >= 100
	lw := w
	if wide {
		lw = (w - 1) * 3 / 5
	}
	iwCores = lw - 4
	colW := iwCores
	perRow := 1
	if iwCores >= 56 {
		perRow, colW = 2, (iwCores-2)/2
	}
	var coreLines []string
	for i := 0; i < len(s.Cores); i += perRow {
		var row []string
		for k := 0; k < perRow && i+k < len(s.Cores); k++ {
			c := s.Cores[i+k]
			gw := max(4, colW-12)
			row = append(row, dim(fmt.Sprintf("C%-2d ", i+k))+gauge(gw, c)+" "+bold(levelColor(c), fmt.Sprintf("%3.0f%%", c)))
		}
		coreLines = append(coreLines, strings.Join(row, "  "))
	}
	if len(coreLines) == 0 {
		coreLines = []string{dim("çekirdek verisi yok")}
	}

	// Info panel.
	kv := func(k, v string) string { return dim(fmt.Sprintf("%-10s", k)) + text(v) }
	info := []string{
		kv("Makine", s.Hostname),
		kv("OS", s.OS),
		kv("CPU", s.CPUModel),
		kv("Çekirdek", fmt.Sprintf("%d · %.2f GHz", s.CoreCount, s.CPUFreqMHz/1000)),
		kv("Çalışma", formatDuration(s.Uptime)),
		kv("Bellek", humanBytes(s.RAMUsed)+" / "+humanBytes(s.RAMTotal)),
		kv("Takas", fmt.Sprintf("%s / %s (%.0f%%)", humanBytes(s.SwapUsed), humanBytes(s.SwapTotal), s.SwapPct)),
	}
	if s.HasLoad {
		info = append(info, kv("Yük", fmt.Sprintf("%.2f · %.2f", s.Load1, s.Load5)))
	}
	if s.HasTemp {
		info = append(info, kv("Sıcaklık", fmt.Sprintf("%.0f°C", s.TempC)))
	}
	if g := s.GPU; g.OK {
		info = append(info, kv("GPU", g.Name),
			kv("GPU yük", fmt.Sprintf("%.0f%% · %.0f°C", g.Util, g.TempC)),
			kv("VRAM", humanBytes(g.MemUsed)+" / "+humanBytes(g.MemTotal)))
	} else {
		info = append(info, kv("GPU", "NVIDIA bulunamadı"))
	}
	if b := s.Battery; b.Present {
		state := "pilde"
		if b.Charging {
			state = "şarjda"
		}
		info = append(info, kv("Pil", fmt.Sprintf("%.0f%% · %s", b.Pct, state)))
	} else {
		info = append(info, kv("Pil", "yok (masaüstü)"))
	}

	need := max(len(info), len(coreLines)) + 2
	chartH := clampInt(h-need, 6, 12)
	topH := h - chartH
	if h < 14 {
		topH, chartH = h, 0
	} else if topH > need && wide {
		topH = need
	}
	var top string
	if wide {
		top = lipgloss.JoinHorizontal(lipgloss.Top,
			box("🧵 "+bold(th.Text, "Çekirdekler"), coreLines, lw, topH, th.Border, 1), " ",
			box("💻 "+bold(th.Text, "Sistem"), info, w-lw-1, topH, th.Border, 1))
	} else {
		ch := clampInt(len(coreLines)+2, 3, topH/2+1)
		top = box("🧵 "+bold(th.Text, "Çekirdekler"), coreLines, w, ch, th.Border, 1) + "\n" +
			box("💻 "+bold(th.Text, "Sistem"), info, w, max(3, topH-ch), th.Border, 1)
	}
	if chartH == 0 {
		return top
	}
	hist := box("📈 "+bold(th.Text, "CPU geçmişi")+dim(" · son dakikalar"),
		bigChart(m.hist.CPU, w-4, chartH-2, 100, gradAt), w, chartH, th.Border, 1)
	return top + "\n" + hist
}
