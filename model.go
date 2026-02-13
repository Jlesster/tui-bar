package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

type model struct {
	currTime  time.Time
	cpuUsage  float64
	memUsage  float64
	diskUsage float64

	netName  string
	netState string

	batLevel int
	batState string

	activeWorkspace int
	windowTitle     string

	width  int
	height int

	hypr *HyprlandClient
}

func initModel() model {
	return model{
		currTime:        time.Now(),
		cpuUsage:        0,
		memUsage:        0,
		diskUsage:       0,
		netName:         "wlan0",
		netState:        "disconnected",
		batLevel:        0,
		batState:        "unknown",
		activeWorkspace: 1,
		windowTitle:     "",
		width:           0,
		height:          0,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		getSystemInfo(),
		getBatteryInfo(),
		getNetworkInfo(),
	)
}
