package main

import (
	"math"

	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func fetchSystemStats() (float64, float64, float64) {
	cpuPercent, err := cpu.Percent(0, false)
	cpuUsage := 0.0
	if err == nil && len(cpuPercent) > 0 {
		cpuUsage = math.Round(cpuPercent[0]*10) / 10
	}

	memInfo, err := mem.VirtualMemory()
	memUsage := 0.0
	if err == nil {
		memUsage = math.Round(memInfo.UsedPercent*10) / 10
	}

	diskInfo, err := disk.Usage("/")
	diskUsage := 0.0
	if err == nil {
		diskUsage = math.Round(diskInfo.UsedPercent*10) / 10
	}
	return cpuUsage, memUsage, diskUsage
}

func fetchBatteryStats() (int, string) {
	batteries, err := battery.GetAll()
	if err != nil || len(batteries) == 0 {
		return 0, "unknown"
	}

	bat := batteries[0]
	level := int(bat.Current / bat.Full * 100)

	stateStr := bat.State.String()
	state := "discharging"

	// Fix: Use the State field correctly
	switch stateStr {
	case "Charging":
		state = "charging"
	case "Full":
		state = "full"
	case "Discharging":
		state = "discharging"
	default:
		state = "unknown"
	}
	return level, state
}

func fetchNetworkInfo() (string, string) {
	return "wlan0", "connected"
}

func fetchHyprlandInfo() (int, string) {
	return 1, "nvim"
}
