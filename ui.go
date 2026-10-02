package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ---- styling helpers ----------------------------------------------------

var styleCache = map[[2]lipgloss.Color]lipgloss.Style{}

// sty returns a cached foreground/background style.
func sty(fg, bg lipgloss.Color) lipgloss.Style {
	k := [2]lipgloss.Color{fg, bg}
	if s, ok := styleCache[k]; ok {
		return s
	}
	s := lipgloss.NewStyle()
	if fg != "" {
		s = s.Foreground(fg)
	}
	if bg != "" {
		s = s.Background(bg)
	}
	styleCache[k] = s
	return s
}

func paint(c lipgloss.Color, s string) string {
	if s == "" {
		return ""
	}
	return sty(c, "").Render(s)
}

func bold(c lipgloss.Color, s string) string {
	if s == "" {
		return ""
	}
	return sty(c, "").Bold(true).Render(s)
}

func dim(s string) string  { return paint(th.Dim, s) }
func text(s string) string { return paint(th.Text, s) }

// gradientText colours each rune along Accent -> Accent2.
func gradientText(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		t := float64(i) / float64(max(1, len(rs)-1))
		b.WriteString(sty(accentAt(t), "").Bold(true).Render(string(r)))
	}
	return b.String()
}

// pill is a filled, rounded-looking label.
func pill(bg lipgloss.Color, s string) string {
	return sty(th.Dark, bg).Bold(true).Render(" " + s + " ")
}

// keyBadge renders "[key] label" for the footer.
func keyBadge(key, label string) string {
	return sty(th.Dark, th.Accent).Bold(true).Render(" "+key+" ") + " " + dim(label)
}

// ---- gauges and charts --------------------------------------------------

var partialBlocks = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// gauge renders a smooth, gradient-filled bar of exactly `width` cells.
func gauge(width int, pct float64) string {
	width = max(1, width)
	pct = clampF(pct, 0, 100)
	eighths := int(pct/100*float64(width*8) + 0.5)
	full, rem := eighths/8, eighths%8
	var b strings.Builder
	for i := 0; i < width; i++ {
		t := float64(i) / float64(max(1, width-1))
		switch {
		case i < full:
			b.WriteString(paint(gradAt(t), "█"))
		case i == full && rem > 0:
			b.WriteString(paint(gradAt(t), partialBlocks[rem]))
		default:
			b.WriteString(paint(trackColor(), "░"))
		}
	}
	return b.String()
}

var sparkBlocks = []rune(" ▁▂▃▄▅▆▇█")

func lastN(vals []float64, n int) []float64 {
	if len(vals) > n {
		return vals[len(vals)-n:]
	}
	return vals
}

func maxOf(vals []float64, floor float64) float64 {
	m := floor
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

// sparkline draws the last `width` samples as one row of block glyphs.
// maxV <= 0 auto-scales (with a floor of -maxV).
func sparkline(vals []float64, width int, maxV float64, colorFn func(float64) lipgloss.Color) string {
	if width < 1 {
		return ""
	}
	data := lastN(vals, width)
	if maxV <= 0 {
		maxV = maxOf(data, -maxV)
		if maxV <= 0 {
			maxV = 1
		}
	}
	var b strings.Builder
	b.WriteString(paint(trackColor(), strings.Repeat("▁", width-len(data))))
	for _, v := range data {
		f := clampF(v/maxV, 0, 1)
		idx := int(f*8 + 0.5)
		if idx == 0 {
			b.WriteString(paint(trackColor(), "▁"))
			continue
		}
		b.WriteString(paint(colorFn(f), string(sparkBlocks[idx])))
	}
	return b.String()
}

// bigChart draws an h-row area chart of the last `w` samples.
func bigChart(vals []float64, w, h int, maxV float64, colorFn func(float64) lipgloss.Color) []string {
	w, h = max(1, w), max(1, h)
	data := lastN(vals, w)
	if maxV <= 0 {
		maxV = maxOf(data, -maxV)
		if maxV <= 0 {
			maxV = 1
		}
	}
	pad := w - len(data)
	rows := make([]strings.Builder, h)
	for c := 0; c < w; c++ {
		v := 0.0
		if c >= pad {
			v = data[c-pad]
		}
		f := clampF(v/maxV, 0, 1)
		eighths := int(f*float64(h*8) + 0.5)
		for r := 0; r < h; r++ {
			level := h - 1 - r
			e := clampInt(eighths-level*8, 0, 8)
			if e == 0 {
				rows[r].WriteString(" ")
				continue
			}
			rows[r].WriteString(paint(colorFn(f), string(sparkBlocks[e])))
		}
	}
	out := make([]string, h)
	for i := range rows {
		out[i] = rows[i].String()
	}
	return out
}

// ---- boxes and layout ---------------------------------------------------

func padTo(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "")
	if d := w - lipgloss.Width(s); d > 0 {
		s += strings.Repeat(" ", d)
	}
	return s
}

// fit forces s to exactly w columns by h rows.
func fit(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	out := make([]string, h)
	for i := range out {
		if i < len(lines) {
			out[i] = padTo(lines[i], w)
		} else {
			out[i] = strings.Repeat(" ", w)
		}
	}
	return strings.Join(out, "\n")
}

// box draws a titled rounded frame exactly w columns wide. If h > 0 the body
// is padded/cropped so the box is exactly h rows tall. pad is the horizontal
// padding inside the frame.
func box(title string, lines []string, w, h int, border lipgloss.Color, pad int) string {
	w = max(w, 6)
	bc := func(s string) string { return paint(border, s) }
	tw := lipgloss.Width(title)
	if tw > w-6 {
		title = ansi.Truncate(title, max(0, w-6), "")
		tw = lipgloss.Width(title)
	}
	top := bc("╭─ ") + title + bc(" "+strings.Repeat("─", max(0, w-5-tw))+"╮")
	inner := w - 2 - 2*pad
	if h > 0 {
		body := make([]string, 0, h-2)
		for i := 0; i < h-2; i++ {
			if i < len(lines) {
				body = append(body, lines[i])
			} else {
				body = append(body, "")
			}
		}
		lines = body
	}
	var b strings.Builder
	b.WriteString(top)
	sp := strings.Repeat(" ", pad)
	for _, ln := range lines {
		b.WriteString("\n" + bc("│") + sp + padTo(ln, inner) + sp + bc("│"))
	}
	b.WriteString("\n" + bc("╰"+strings.Repeat("─", w-2)+"╯"))
	return b.String()
}

func card(title string, lines []string, w int, border lipgloss.Color) string {
	return box(title, lines, w, 0, border, 1)
}

// ---- formatting ---------------------------------------------------------

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func humanRate(kbs float64) string {
	return humanBytes(uint64(max(0, kbs)*1024)) + "/s"
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0dk"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dg %dsa", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dsa %ddk", hours, mins)
	default:
		return fmt.Sprintf("%ddk", mins)
	}
}

// tempString formats the CPU temperature, or a clear N/A when unavailable.
func tempString(s Stats) string {
	if !s.HasTemp {
		return dim("N/A")
	}
	c := th.Good
	switch {
	case s.TempC >= 80:
		c = th.Bad
	case s.TempC >= 60:
		c = th.Warn
	}
	return bold(c, fmt.Sprintf("%.0f°C", s.TempC))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
