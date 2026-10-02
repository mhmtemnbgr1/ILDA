package main

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// readGPU asks nvidia-smi for utilisation, memory and temperature. It returns
// OK=false when no NVIDIA GPU / tool is present.
func readGPU() GPUStat {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return GPUStat{}
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	parts := strings.Split(line, ",")
	if len(parts) < 5 {
		return GPUStat{}
	}
	num := func(s string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return v
	}
	return GPUStat{
		OK:       true,
		Name:     strings.TrimSpace(parts[0]),
		Util:     num(parts[1]),
		MemUsed:  uint64(num(parts[2]) * 1024 * 1024),
		MemTotal: uint64(num(parts[3]) * 1024 * 1024),
		TempC:    num(parts[4]),
	}
}
