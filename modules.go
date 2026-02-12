package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Module interface {
	Name() string
	Update() error
	Render() string
	Style() lipgloss.Style
}

type CPUModule struct {
	usage float64
}

func (m *CPUModule) Name() string {
	return "cpu"
}

func (m *CPUModule) Update() error {
	usage, _, _ := fetchSystemStats()
	m.usage = usage
	return nil
}

func (m *CPUModule) Render() string {
	return fmt.Sprintf("󰻠 %.1f%%", m.usage)
}

func (m *CPUModule) Style() lipgloss.Style {
	return cpuStyle
}
