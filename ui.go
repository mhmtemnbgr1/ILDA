package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7df9ff"))
	labelStyle  = lipgloss.NewStyle().Faint(true)
	footerStyle = lipgloss.NewStyle().Faint(true)
	trackStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#333844"))
	tankStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2b6f8a"))
)

// gauge renders a fixed-width bar coloured by how full it is.
func gauge(width int, pct float64) string {
	if width < 1 {
		width = 1
	}
	filled := int(pct/100*float64(width) + 0.5)
	filled = clampInt(filled, 0, width)
	bar := lipgloss.NewStyle().Foreground(levelColor(pct)).Render(strings.Repeat("█", filled))
	return bar + trackStyle.Render(strings.Repeat("░", width-filled))
}

// statLine is one labelled gauge row: "CPU  ██████░░░░  52.0%".
func statLine(label string, width int, pct float64, suffix string) string {
	return labelStyle.Render(label) + " " + gauge(width, pct) + " " + suffix
}

func levelColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 85:
		return lipgloss.Color("#ff5f5f")
	case pct >= 60:
		return lipgloss.Color("#ffd75f")
	default:
		return lipgloss.Color("#5fff87")
	}
}

// tempString formats the CPU temperature, or a clear N/A when the machine
// doesn't expose one (common on Windows without extra drivers).
func tempString(s Stats) string {
	if !s.HasTemp {
		return labelStyle.Render("N/A")
	}
	var c lipgloss.Color
	switch {
	case s.TempC >= 80:
		c = lipgloss.Color("#ff5f5f")
	case s.TempC >= 60:
		c = lipgloss.Color("#ffd75f")
	default:
		c = lipgloss.Color("#5fff87")
	}
	return lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%.0f°C", s.TempC))
}

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

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
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
