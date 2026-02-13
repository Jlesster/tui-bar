package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
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
	signature := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")

	if signature == "" {
		return nil, fmt.Errorf("not running in hyprland")
	}

	return &HyprlandClient{
		listeners: make([]chan HyprlandEvent, 0),
		signature: signature,
	}, nil
}

func (hc *HyprlandClient) sendCommand(command string) ([]byte, error) {
	socketPath := fmt.Sprintf("/tmp/hypr/%s/.socket.sock", hc.signature)

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
	data, err := hc.sendCommand("j/monitors")
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
	cmd := fmt.Sprintf("dispatch workspace %d", workspace)
	_, err := hc.sendCommand(cmd)
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

func (hc *HyprlandClient) StartEventListener() error {
	socketPath := fmt.Sprintf("/tmp/hypr/%s/.socket2.sock", hc.signature)
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to event socket: %v", err)
	}
	hc.eventConn = conn

	go hc.readEvents()
	log.Println("Connected to Hyprland event socket")
	return nil
}

func (hc *HyprlandClient) readEvents() {
	defer hc.eventConn.Close()

	scanner := bufio.NewScanner(hc.eventConn)
	for scanner.Scan() {
		line := scanner.Text()
		event := hc.parseEvent(line)
		if event != nil {
			hc.dispatchEvent(*event)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from event socket: %v", err)
	}
}

func (hc *HyprlandClient) parseEvent(line string) *HyprlandEvent {
	parts := strings.SplitN(line, ">>", 2)
	if len(parts) != 2 {
		return nil
	}

	eventType := parts[0]
	eventData := strings.Split(parts[1], ",")

	return &HyprlandEvent{
		Type: eventType,
		Data: eventData,
	}
}

func (hc *HyprlandClient) dispatchEvent(event HyprlandEvent) {
	hc.eventMux.RLock()
	defer hc.eventMux.RUnlock()

	for _, listener := range hc.listeners {
		select {
		case listener <- event:
		default:
		}
	}
}

func (hc *HyprlandClient) Subscribe() chan HyprlandEvent {
	hc.eventMux.Lock()
	defer hc.eventMux.Unlock()

	ch := make(chan HyprlandEvent, 100)
	hc.listeners = append(hc.listeners, ch)
	return ch
}

func (hc *HyprlandClient) Unsubscribe(ch chan HyprlandEvent) {
	hc.eventMux.Lock()
	defer hc.eventMux.Unlock()

	for i, listener := range hc.listeners {
		if listener == ch {
			hc.listeners = append(hc.listeners[:i], hc.listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

func (hc *HyprlandClient) Close() {
	if hc.eventConn != nil {
		hc.eventConn.Close()
	}
	hc.eventMux.Lock()
	for _, ch := range hc.listeners {
		close(ch)
	}
	hc.listeners = nil
	hc.eventMux.Unlock()
}

// helpers
func getActiveWorkspace() int {
	client, err := NewHyprlandClient()
	if err != nil {
		return 1
	}
	ws, err := client.GetActiveWorkspace()
	if err != nil {
		return 1
	}
	return ws.ID
}

func getActiveWindow() string {
	client, err := NewHyprlandClient()
	if err != nil {
		return ""
	}

	win, err := client.GetActiveWindow()
	if err != nil {
		return ""
	}
	if win.Title != "" {
		return win.Title
	}
	return win.Class
}

func (hc *HyprlandClient) GetWorkspaceWindows(workspaceID int) ([]HyprlandWindow, error) {
	windows, err := hc.GetWindows()
	if err != nil {
		return nil, err
	}

	var wsWindows []HyprlandWindow
	for _, win := range windows {
		if win.Workspace.ID == workspaceID {
			wsWindows = append(wsWindows, win)
		}
	}
	return wsWindows, nil
}

func (hc *HyprlandClient) IsWorkspaceEmpty(workspaceID int) (bool, error) {
	windows, err := hc.GetWorkspaceWindows(workspaceID)
	if err != nil {
		return false, err
	}
	return len(windows) == 0, nil
}

func (hc *HyprlandClient) GetWorkspaceByName(name string) (*HyprlandWorkspace, error) {
	workspaces, err := hc.GetWorkspaces()
	if err != nil {
		return nil, err
	}
	for _, ws := range workspaces {
		if ws.Name == name {
			return &ws, nil
		}
	}
	return nil, fmt.Errorf("workspace not found: %s", name)
}
