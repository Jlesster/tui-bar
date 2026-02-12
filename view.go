package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 {
		return "Initializing.."
	}

	workspaces := renderWorkspaces(m.activeWorkspace)
	clock := renderClock(m.currTime)
	sysInfo := renderSystemInfo(m)

	leftWidth := lipgloss.Width(workspaces)
	centerWidth := lipgloss.Width(clock)
	rightWidth := lipgloss.Width(sysInfo)

	totalContentWidth := leftWidth + centerWidth + rightWidth
	avaliableSpace := m.width - totalContentWidth

	leftPadding := avaliableSpace / 3
	rightPadding := avaliableSpace - leftPadding

	statusbar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		workspaces,
		strings.Repeat(" ", leftPadding),
		clock,
		strings.Repeat(" ", rightPadding),
		sysInfo,
	)

	return statusbar
}

func renderWorkspaces(active int) string {
	workspaces := []string{}

	for i := 1; i <= 4; i++ {
		ws := fmt.Sprintf("%d", i)
		if i == active {
			workspaces = append(workspaces, workspaceActiveStyle.Render(ws))
		} else {
			workspaces = append(workspaces, workspaceStyle.Render(ws))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, workspaces...)
}

func renderClock(t time.Time) string {
	timeStr := t.Format("15:04;05 | Mon 02 Jan")
	return clockStyle.Render(timeStr)
}

func renderSystemInfo(m model) string {
	modules := []string{}

	cpu := fmt.Sprintf("󰻠 %.1f%%", m.cpuUsage)
	modules = append(modules, cpuStyle.Render(cpu))

	memory := fmt.Sprintf("󰍛 %.1f%%", m.memUsage)
	modules = append(modules, memoryStyle.Render(memory))

	disk := fmt.Sprintf("󰋊 %.1f%%", m.diskUsage)
	modules = append(modules, diskStyle.Render(disk))

	netIcon := getNetworkIcon(m.netState)
	network := fmt.Sprintf("%s %s", netIcon, m.netName)
	modules = append(modules, networkStyle.Render(network))

	batIcon := getBatteryIcon(m.batLevel, m.batState)
	battery := fmt.Sprintf("%s %d%%", batIcon, m.batLevel)

	var batStyle lipgloss.Style
	if m.batState == "charging" {
		batStyle = batteryChargingStyle
	} else if m.batLevel < 20 {
		batStyle = batteryLowStyle
	} else {
		batStyle = batteryStyle
	}

	modules = append(modules, batStyle.Render(battery))
	return lipgloss.JoinHorizontal(lipgloss.Top, modules...)
}
