package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

type tickMsg time.Time
type sysInfoMsg struct {
	cpu  float64
	mem  float64
	disk float64
}
type batteryMsg struct {
	level int
	state string
}
type networkMsg struct {
	name  string
	state string
}
type hyprlandMsg struct {
	activeWorkspace int
	windowTitle     string
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func getSystemInfo() tea.Cmd {
	return func() tea.Msg {
		cpu, mem, disk := fetchSystemStats()
		return sysInfoMsg{
			cpu:  cpu,
			mem:  mem,
			disk: disk,
		}
	}
}

func getBatteryInfo() tea.Cmd {
	return func() tea.Msg {
		level, state := fetchBatteryStats()
		return batteryMsg{
			level: level,
			state: state,
		}
	}
}

func getNetworkInfo() tea.Cmd {
	return func() tea.Msg {
		name, state := fetchNetworkInfo()
		return networkMsg{
			name:  name,
			state: state,
		}
	}
}

func getHyprlandInfo() tea.Cmd {
	return func() tea.Msg {
		ws := getActiveWorkspace()
		win := getActiveWindow()
		return hyprlandMsg{
			activeWorkspace: ws,
			windowTitle:     win,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft {
			//TODO write mouse logic
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.currTime = time.Time(msg)
		return m, tea.Batch(
			tickCmd(),
			getSystemInfo(),
			getBatteryInfo(),
			getNetworkInfo(),
			getHyprlandInfo(),
		)

	case sysInfoMsg:
		m.cpuUsage = msg.cpu
		m.memUsage = msg.mem
		m.diskUsage = msg.disk

	case batteryMsg:
		m.batLevel = msg.level
		m.batState = msg.state

	case networkMsg:
		m.netName = msg.name
		m.netState = msg.state

	case hyprlandMsg:
		m.activeWorkspace = msg.activeWorkspace
		m.windowTitle = msg.windowTitle
	}
	return m, nil
}
