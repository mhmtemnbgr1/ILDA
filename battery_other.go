//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readBattery() BatteryStat {
	dirs, _ := filepath.Glob("/sys/class/power_supply/BAT*")
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "capacity"))
		if err != nil {
			continue
		}
		pct, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
		if err != nil {
			continue
		}
		st, _ := os.ReadFile(filepath.Join(d, "status"))
		status := strings.TrimSpace(string(st))
		return BatteryStat{Present: true, Pct: pct, Charging: status == "Charging" || status == "Full"}
	}
	return BatteryStat{}
}
