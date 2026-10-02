//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var procGetSystemPowerStatus = syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemPowerStatus")

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

func readBattery() BatteryStat {
	var s systemPowerStatus
	r, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&s)))
	if r == 0 || s.BatteryFlag == 128 || s.BatteryLifePercent > 100 {
		return BatteryStat{}
	}
	return BatteryStat{Present: true, Pct: float64(s.BatteryLifePercent), Charging: s.ACLineStatus == 1}
}
