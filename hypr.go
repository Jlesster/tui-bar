package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type HyprlandWorkspace struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Monitor         string `json:"monitor"`
	Windows         string `json:"windows"`
	HasFullscreen   bool   `json:"hasfullscreen"`
	LastWindow      string `json:"lastwindow"`
	LastWindowTitle string `json:"lastwindowtitle"`
}

type HyprlandWindow struct {
	Address   string `json:"address"`
	Class     string `json:"class"`
	Title     string `json:"title"`
	Workspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"workspace"`
	Monitor    string `json:"monitor"`
	Fullscreen bool   `json:"fullscreen"`
	Floating   bool   `json:"floating"`
	Pinned     bool   `json:"pinned"`
	At         [2]int `json:"at"`
	Size       [2]int `json:"size"`
}

type HyprlandMonitor struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Make            string  `json:"make"`
	Model           string  `json:"model"`
	Serial          string  `json:"serial"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Refreshrate     float64 `json:"refreshRate"`
	X               int     `json:"x"`
	Y               int     `json:"y"`
	ActiveWorkspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"activeWorkspace"`
	Reserved   [4]int  `json:"reserved"`
	Scale      float64 `json:"scale"`
	Transform  int     `json:"transform"`
	Focused    bool    `json:"focused"`
	DpmsStatus bool    `json:"dpmsStatus"`
	Vrr        bool    `json:"vrr"`
}

type HyprlandEvent struct {
	Type string
	Data []string
}

type HyprlandClient struct {
	signature   string
	commandConn net.Conn
	eventConn   net.Conn
	eventMux    sync.RWMutex
	listeners   []chan HyprlandEvent
}

func NewHyprlandClient() (*HyprlandClient, error) {
	signature := os.Getenv("HYPRLAND_INSTANCE_SINGATURE")

	if signature == "" {
		return nil, fmt.Errorf("not running in hyprland")
	}

	return &HyprlandClient{
		listeners: make([]chan HyprlandEvent, 0),
		signature: signature,
	}, nil
}

func (hc *HyprlandClient) sendCommand(command string) ([]byte, error) {
	socketPath := fmt.Sprint("/tmp/hypr/%s/.socket.sock", hc.signature)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to hyprland")
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(command)); err != nil {
		return nil, err
	}

	buf := make([]byte, 16384)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (hc *HyprlandClient) GetActiveWorkspace() (*HyprlandWorkspace, error) {
	data, err := hc.sendCommand("j/activeworkspace")
	if err != nil {
		return nil, err
	}

	var workspace HyprlandWorkspace
	if err := json.Unmarshal(data, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (hc *HyprlandClient) GetWorkspaces() ([]HyprlandWorkspace, error) {
	data, err := hc.sendCommand("j/workspaces")
	if err != nil {
		return nil, err
	}

	var workspaces []HyprlandWorkspace
	if err := json.Unmarshal(data, &workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (hc *HyprlandClient) GetActiveWindow() (*HyprlandWindow, error) {
	data, err := hc.sendCommand("j/activewindow")
	if err != nil {
		return nil, err
	}

	var window HyprlandWindow
	if err := json.Unmarshal(data, &window); err != nil {
		return nil, err
	}
	return &window, nil
}

func (hc *HyprlandClient) GetWindows() ([]HyprlandWindow, error) {
	data, err := hc.sendCommand("j/clients")
	if err != nil {
		return nil, err
	}

	var windows []HyprlandWindow
	if err := json.Unmarshal(data, &windows); err != nil {
		return nil, err
	}
	return windows, nil
}

func (hc *HyprlandClient) GetMonitors() ([]HyprlandMonitor, error) {
	data, err := hc.sendCommand("j/,monitors")
	if err != nil {
		return nil, err
	}

	var monitors []HyprlandMonitor
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, err
	}
	return monitors, nil
}

func (hc *HyprlandClient) GetActiveMonitor() (*HyprlandMonitor, error) {
	monitors, err := hc.GetMonitors()
	if err != nil {
		return nil, err
	}
	for _, mon := range monitors {
		if mon.Focused {
			return &mon, nil
		}
	}
	return nil, fmt.Errorf("no focused monitor found")
}

func (hc *HyprlandClient) SwitchWorkspace(workspace int) error {
	_, err := hc.sendCommand(cmd)
	cmd := fmt.Sprintf("dispatch workspace %d", workspace)
	return err
}

func (hc *HyprlandClient) SwitchWorkspaceByName(name string) error {
	cmd := fmt.Sprintf("dispatch workspace name %s", name)
	_, err := hc.sendCommand(cmd)
	return err
}

func (hc *HyprlandClient) MoveToWorkspace(workspace int) error {
	cmd := fmt.Sprintf("dispatch movetoworkspace %d", workspace)
	_, err := hc.sendCommand(cmd)
	return err
}

func (hc *HyprlandClient) ToggleFullscreen() error {
	_, err := hc.sendCommand("dispatch fullscreen")
	return err
}

func (hc *HyprlandClient) KillActiveWindow() error {
	_, err := hc.sendCommand("dispatch killactive")
	return err
}

func (hc *HyprlandClient) ToggleFloating() error {
	_, err := hc.sendCommand("dispatch togglefloating")
	return err
}

func (hc *HyprlandClient) FocusMonitor(monitor string) error {
	cmd := fmt.Sprintf("dispatch focusmonitor %s", monitor)
	_, err := hc.sendCommand(cmd)
	return err
}

func (hc *HyprlandClient) MoveWorkspaceToMontior(workspace int, monitor string) error {
	cmd := fmt.Sprintf("dispatch moveworkspacetomonitor %d %s", workspace, monitor)
	_, err := hc.sendCommand(cmd)
	return err
}

//TODO event socket listeners
