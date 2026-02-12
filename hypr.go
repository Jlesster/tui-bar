package main

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type HyprlandWorkspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type HyprlandWindow struct {
	Class string `json:"class"`
	Title string `json:"title"`
}

func getActiveWorkspace() int {
	cmd := exec.Command("hyprctl", "activeworkspace", "-j")
	output, err := cmd.Output()
	if err != nil {
		return 1
	}

	var workspace HyprlandWorkspace
	if err := json.Unmarshal(output, &workspace); err != nil {
		return 1
	}
	return workspace.ID
}

func getActiveWindow() string {
	cmd := exec.Command("hyprctl", "activewindow", "-j")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	var window HyprlandWindow
	if err := json.Unmarshal(output, &window); err != nil {
		return ""
	}

	if window.Title != "" {
		return window.Title
	}
	return window.Class
}

func listenHyprEvents() tea.Cmd {
	return func() tea.Msg {
		//TODO write hyprland event listener
		return nil
	}
}
