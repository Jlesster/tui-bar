package main

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/distatus/battery"
	"math"
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
	state := "discharging"

	if bat.State == battery.Charging {
		state = "charging"
	} else if bat.State == battery.Full {
		state = "full"
	}
	return level, state
}

func fetchNetworkInfo() (string, string) {
	return "wlan0", "connected"
}

func fetchHyprlandInfo() (int, string) {
	return 1, "nvim"
}
