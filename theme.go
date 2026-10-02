package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a complete colour palette for the dashboard and the aquarium.
type Theme struct {
	Name string

	Text, Dim, Border, Accent, Accent2 lipgloss.Color
	Good, Warn, Bad, Dark              lipgloss.Color

	WaterTop, WaterBot, Ray, Sand, Weed, Bubble, Food lipgloss.Color
	Fish                                              []lipgloss.Color
}

var themes = []Theme{
	{
		Name: "Okyanus",
		Text: "#e6f1ff", Dim: "#8aa0b8", Border: "#2f7f9e", Accent: "#00e5ff", Accent2: "#ff6ec7",
		Good: "#4cf0a0", Warn: "#ffd166", Bad: "#ff5c7a", Dark: "#06222f",
		WaterTop: "#0d5a80", WaterBot: "#031826", Ray: "#7fdfff", Sand: "#c2a26b",
		Weed: "#2ecc71", Bubble: "#bdeeff", Food: "#ffd166",
		Fish: []lipgloss.Color{"#ffb347", "#ff6b6b", "#4ecdc4", "#f7d794", "#a29bfe", "#55efc4", "#fd79a8"},
	},
	{
		Name: "Neon",
		Text: "#f2e9ff", Dim: "#9a8cc0", Border: "#6b46c1", Accent: "#ff4fd8", Accent2: "#00f0ff",
		Good: "#39ff9c", Warn: "#ffe14d", Bad: "#ff3d6e", Dark: "#12081f",
		WaterTop: "#3a1580", WaterBot: "#0b0420", Ray: "#b78bff", Sand: "#8a6a9f",
		Weed: "#39ff9c", Bubble: "#d9c4ff", Food: "#ffe14d",
		Fish: []lipgloss.Color{"#ff4fd8", "#00f0ff", "#ffe14d", "#39ff9c", "#ff8a3d", "#b78bff"},
	},
	{
		Name: "Retro",
		Text: "#c8ffc8", Dim: "#5fa05f", Border: "#2e8b3a", Accent: "#39ff14", Accent2: "#c6ff00",
		Good: "#39ff14", Warn: "#e6ff3d", Bad: "#ff5252", Dark: "#021002",
		WaterTop: "#0a4010", WaterBot: "#010d02", Ray: "#5cff5c", Sand: "#557f35",
		Weed: "#27c227", Bubble: "#a8ffa8", Food: "#e6ff3d",
		Fish: []lipgloss.Color{"#39ff14", "#c6ff00", "#7dff7d", "#9dff00", "#4dff9d"},
	},
	{
		Name: "Şeker",
		Text: "#fff0f6", Dim: "#d3a0bc", Border: "#e056a0", Accent: "#ff7eb6", Accent2: "#ffd86e",
		Good: "#7dffc4", Warn: "#ffd86e", Bad: "#ff5c8a", Dark: "#2a0a1c",
		WaterTop: "#8a3a8e", WaterBot: "#2a0f3a", Ray: "#ffc2e2", Sand: "#f6c6a1",
		Weed: "#7dffc4", Bubble: "#ffe0f0", Food: "#ffd86e",
		Fish: []lipgloss.Color{"#ffd86e", "#7dffc4", "#9ad7ff", "#ffb3d9", "#c8a2ff"},
	},
}

// th is the active theme.
var th = themes[0]

func themeIndex(name string) int {
	for i, t := range themes {
		if t.Name == name {
			return i
		}
	}
	return 0
}

func rgb(c lipgloss.Color) (float64, float64, float64) {
	var r, g, b uint8
	if len(c) == 7 {
		fmt.Sscanf(string(c[1:]), "%02x%02x%02x", &r, &g, &b)
	}
	return float64(r), float64(g), float64(b)
}

func lerpColor(a, b lipgloss.Color, t float64) lipgloss.Color {
	t = clampF(t, 0, 1)
	ar, ag, ab := rgb(a)
	br, bg, bb := rgb(b)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
		uint8(ar+(br-ar)*t), uint8(ag+(bg-ag)*t), uint8(ab+(bb-ab)*t)))
}

// gradAt maps 0..1 onto Good -> Warn -> Bad.
func gradAt(t float64) lipgloss.Color {
	if t < 0.65 {
		return lerpColor(th.Good, th.Warn, t/0.65)
	}
	return lerpColor(th.Warn, th.Bad, (t-0.65)/0.35)
}

// accentAt maps 0..1 onto Accent -> Accent2 (used for neutral data like network).
func accentAt(t float64) lipgloss.Color { return lerpColor(th.Accent, th.Accent2, t) }

func trackColor() lipgloss.Color { return lerpColor(th.Dark, th.Dim, 0.35) }

func levelColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 85:
		return th.Bad
	case pct >= 60:
		return th.Warn
	default:
		return th.Good
	}
}
