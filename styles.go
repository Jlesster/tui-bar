package main

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	primary = lipgloss.Color("4")
	surface = lipgloss.Color("1")
	text    = lipgloss.Color("3")
	textDim = lipgloss.Color("4")
	purple  = lipgloss.Color("6")
	pink    = lipgloss.Color("5")
	green   = lipgloss.Color("7")
	yellow  = lipgloss.Color("8")
	red     = lipgloss.Color("9")

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.AdaptiveColor{
			Light: "#D7BAFF",
			Dark:  "#D7BAFF",
		}).
		Padding(0, 1).
		Foreground(text).
		Background(lipgloss.AdaptiveColor{
			Light: "transparent",
			Dark:  "transparent",
		})

	activeBoxStyle = boxStyle.Copy().
			BorderForeground(primary).
			Foreground(primary).
			Bold(true)

	workspaceStyle = boxStyle.Copy().
			Foreground(textDim).
			Padding(0, 1)

	workspaceActiveStyle = workspaceStyle.Copy().
				Background(lipgloss.Color("#D7BAFF")).
				Foreground(surface).
				Bold(true)

	cpuStyle = boxStyle.Copy().
			Foreground(pink).
			BorderForeground(purple)

	memoryStyle = boxStyle.Copy().
			Foreground(pink).
			BorderForeground(pink)

	diskStyle = boxStyle.Copy().
			Foreground(text)

	batteryStyle = boxStyle.Copy().
			Foreground(text)

	batteryChargingStyle = boxStyle.Copy().
				Foreground(green).
				BorderForeground(green)

	batteryLowStyle = boxStyle.Copy().
			Foreground(red).
			BorderForeground(red)

	networkStyle = boxStyle.Copy().
			Foreground(purple).
			BorderForeground(purple)

	clockStyle = activeBoxStyle.Copy()
)
