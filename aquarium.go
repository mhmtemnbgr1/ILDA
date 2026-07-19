package main

import (
	"math"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// ---- colors -------------------------------------------------------------

var (
	waterColor  = lipgloss.Color("#04212f")
	bubbleColor = lipgloss.Color("#a8e6ff")
	weedColor   = lipgloss.Color("#2ecc71")
	sandColor   = lipgloss.Color("#c2a26b")
	stressColor = lipgloss.Color("#ff3b3b")

	fishPalette = []lipgloss.Color{
		"#ffb347", "#ff6b6b", "#4ecdc4", "#f7d794", "#a29bfe", "#55efc4", "#fd79a8",
	}
)

// ---- entities -----------------------------------------------------------

type fish struct {
	x, y   float64
	vx     float64 // signed horizontal velocity (cells/sec)
	baseVx float64 // magnitude of the fish's natural speed
	right  bool
	kind   int
	color  lipgloss.Color
}

type bubble struct {
	x, y float64
	vy   float64
}

type seaweed struct {
	x, h int
}

type cell struct {
	r  rune
	fg lipgloss.Color // "" means no foreground (plain water)
}

// Tank is the little ecosystem drawn at the bottom of the dashboard. Its
// inhabitants react to live system stats: fish swim faster as the CPU heats
// up, the school grows with RAM usage, and bubbles stream with network I/O.
type Tank struct {
	w, h    int
	fish    []*fish
	bubbles []*bubble
	seaweed []seaweed
	rng     *rand.Rand

	cpu, ram, net float64
}

func NewTank() *Tank {
	return &Tank{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// SetStats feeds fresh readings into the ecosystem.
func (t *Tank) SetStats(cpu, ram, net float64) {
	t.cpu, t.ram, t.net = cpu, ram, net
	t.syncPopulation()
}

// Resize updates the tank dimensions and keeps inhabitants in bounds.
func (t *Tank) Resize(w, h int) {
	if w == t.w && h == t.h {
		return
	}
	t.w, t.h = w, h
	for _, f := range t.fish {
		if f.x > float64(w-1) {
			f.x = float64(w - 1)
		}
		if f.y > float64(h-2) {
			f.y = float64(h - 2)
		}
		if f.y < 0 {
			f.y = 0
		}
	}
	t.initSeaweed()
}

func (t *Tank) initSeaweed() {
	t.seaweed = t.seaweed[:0]
	if t.w < 6 {
		return
	}
	n := t.w / 22
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		t.seaweed = append(t.seaweed, seaweed{x: t.rng.Intn(t.w), h: 2 + t.rng.Intn(3)})
	}
}

// syncPopulation grows or shrinks the school toward a size driven by RAM usage.
func (t *Tank) syncPopulation() {
	target := 3 + int(t.ram/100*7) // 3..10 fish
	if target < 3 {
		target = 3
	}
	if target > 10 {
		target = 10
	}
	for len(t.fish) < target {
		t.addFish()
	}
	if len(t.fish) > target {
		t.fish = t.fish[:target]
	}
}

func (t *Tank) addFish() {
	w, h := t.w, t.h
	if w <= 0 {
		w = 40
	}
	if h <= 0 {
		h = 12
	}
	f := &fish{
		x:      t.rng.Float64() * float64(max(1, w-6)),
		y:      1 + t.rng.Float64()*float64(max(1, h-2)),
		baseVx: 3 + t.rng.Float64()*6,
		kind:   t.rng.Intn(2),
		color:  fishPalette[t.rng.Intn(len(fishPalette))],
		right:  t.rng.Intn(2) == 0,
	}
	f.vx = f.baseVx
	if !f.right {
		f.vx = -f.vx
	}
	t.fish = append(t.fish, f)
}

// Update advances the animation by dt seconds.
func (t *Tank) Update(dt float64) {
	if t.w <= 0 || t.h <= 0 {
		return
	}
	speed := 1 + t.cpu/100*3 // fish dart up to 4x faster at full CPU load

	for _, f := range t.fish {
		f.x += f.vx * speed * dt
		sw := utf8.RuneCountInString(fishSprite(f))
		switch {
		case f.x < 0:
			f.x, f.right, f.vx = 0, true, math.Abs(f.baseVx)
		case f.x+float64(sw) > float64(t.w):
			f.x, f.right, f.vx = float64(t.w-sw), false, -math.Abs(f.baseVx)
		}
		if t.rng.Float64() < 0.03 { // occasional vertical wander
			f.y += t.rng.Float64()*2 - 1
			f.y = clampF(f.y, 0, float64(t.h-2))
		}
	}

	// Bubbles rise from the floor; spawn rate scales with network activity.
	spawnProb := 0.04 + math.Min(t.net/600, 0.45)
	if t.rng.Float64() < spawnProb {
		t.bubbles = append(t.bubbles, &bubble{
			x:  t.rng.Float64() * float64(t.w),
			y:  float64(t.h - 2),
			vy: 4 + t.rng.Float64()*4,
		})
	}
	kept := t.bubbles[:0]
	for _, b := range t.bubbles {
		b.y -= b.vy * dt
		b.x += math.Sin(b.y) * 0.15 // gentle sideways wobble
		if b.y > 0 {
			kept = append(kept, b)
		}
	}
	t.bubbles = kept
}

// Render paints the current frame as a styled, multi-line string of exactly
// t.h rows and t.w visible columns.
func (t *Tank) Render() string {
	w, h := t.w, t.h
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	grid := make([][]cell, h)
	for y := range grid {
		grid[y] = make([]cell, w)
		for x := range grid[y] {
			grid[y][x] = cell{r: ' '}
		}
	}

	// Sandy floor.
	for x := 0; x < w; x++ {
		grid[h-1][x] = cell{r: '▒', fg: sandColor}
	}
	// Seaweed swaying up from the floor.
	for _, s := range t.seaweed {
		if s.x >= w {
			continue
		}
		for i := 0; i < s.h; i++ {
			y := h - 2 - i
			if y < 0 {
				continue
			}
			r := '('
			if i%2 == 1 {
				r = ')'
			}
			grid[y][s.x] = cell{r: r, fg: weedColor}
		}
	}
	// Bubbles.
	for _, b := range t.bubbles {
		x, y := int(b.x), int(b.y)
		if x >= 0 && x < w && y >= 0 && y < h-1 {
			grid[y][x] = cell{r: '°', fg: bubbleColor}
		}
	}
	// Fish, drawn last so they sit on top of everything.
	for _, f := range t.fish {
		y := int(f.y)
		if y < 0 || y >= h-1 {
			continue
		}
		fg := f.color
		if t.cpu > 85 {
			fg = stressColor // the whole school panics under heavy load
		}
		for i, r := range []rune(fishSprite(f)) {
			x := int(f.x) + i
			if x >= 0 && x < w {
				grid[y][x] = cell{r: r, fg: fg}
			}
		}
	}

	lines := make([]string, h)
	for y := 0; y < h; y++ {
		lines[y] = renderTankLine(grid[y])
	}
	return strings.Join(lines, "\n")
}

func fishSprite(f *fish) string {
	switch f.kind {
	case 1:
		if f.right {
			return "><(((º>"
		}
		return "<º)))><"
	default:
		if f.right {
			return "><>"
		}
		return "<><"
	}
}

// renderTankLine turns a row of cells into a styled string, coalescing runs of
// the same foreground so we emit as few ANSI escapes as possible.
func renderTankLine(cells []cell) string {
	var b strings.Builder
	for i := 0; i < len(cells); {
		j := i
		fg := cells[i].fg
		var seg strings.Builder
		for j < len(cells) && cells[j].fg == fg {
			seg.WriteRune(cells[j].r)
			j++
		}
		st := lipgloss.NewStyle().Background(waterColor)
		if fg != "" {
			st = st.Foreground(fg)
		}
		b.WriteString(st.Render(seg.String()))
		i = j
	}
	return b.String()
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
