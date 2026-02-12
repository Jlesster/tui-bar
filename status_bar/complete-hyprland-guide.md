# Complete Hyprland Integration Guide

A comprehensive guide to integrating your statusbar deeply with Hyprland, including workspaces, windows, monitors, and real-time event handling.

## Table of Contents

1. [Understanding Hyprland IPC](#understanding-hyprland-ipc)
2. [Hyprland Socket Communication](#hyprland-socket-communication)
3. [Complete Event System](#complete-event-system)
4. [Workspace Management](#workspace-management)
5. [Window Information](#window-information)
6. [Monitor Support](#monitor-support)
7. [Advanced Features](#advanced-features)
8. [Performance Optimization](#performance-optimization)
9. [Testing and Debugging](#testing-and-debugging)

---

## Understanding Hyprland IPC

Hyprland provides two IPC mechanisms:

### 1. **Command Socket** (`.socket.sock`)
- Execute commands
- Query current state
- One-time requests/responses
- Located at: `/tmp/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket.sock`

### 2. **Event Socket** (`.socket2.sock`)
- Real-time event stream
- Persistent connection
- Push-based updates
- Located at: `/tmp/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock`

### Event Types

Hyprland emits various events:

| Event | Trigger | Data |
|-------|---------|------|
| `workspace` | Workspace switched | workspace name |
| `focusedmon` | Monitor focus changed | monitor name, workspace |
| `activewindow` | Window focused | class, title |
| `activewindowv2` | Window focused | address |
| `fullscreen` | Fullscreen toggled | 0/1 |
| `monitorremoved` | Monitor disconnected | monitor name |
| `monitoradded` | Monitor connected | monitor name |
| `createworkspace` | Workspace created | workspace name |
| `destroyworkspace` | Workspace destroyed | workspace name |
| `moveworkspace` | Workspace moved | workspace, monitor |
| `activelayout` | Layout changed | keyboard, layout |
| `openwindow` | Window opened | address, workspace, class, title |
| `closewindow` | Window closed | address |
| `movewindow` | Window moved | address, workspace |
| `openlayer` | Layer opened | namespace |
| `closelayer` | Layer closed | namespace |
| `submap` | Submap changed | submap name |

---

## Hyprland Socket Communication

### Complete `hypr.go` Implementation

```go
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
)

// ============================================================================
// Data Structures
// ============================================================================

type HyprlandWorkspace struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Monitor         string `json:"monitor"`
	Windows         int    `json:"windows"`
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
	RefreshRate     float64 `json:"refreshRate"`
	X               int     `json:"x"`
	Y               int     `json:"y"`
	ActiveWorkspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"activeWorkspace"`
	Reserved      [4]int  `json:"reserved"`
	Scale         float64 `json:"scale"`
	Transform     int     `json:"transform"`
	Focused       bool    `json:"focused"`
	DpmsStatus    bool    `json:"dpmsStatus"`
	Vrr           bool    `json:"vrr"`
}

type HyprlandEvent struct {
	Type string
	Data []string
}

// ============================================================================
// Hyprland Client
// ============================================================================

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
		return nil, fmt.Errorf("not running under Hyprland")
	}

	return &HyprlandClient{
		signature: signature,
		listeners: make([]chan HyprlandEvent, 0),
	}, nil
}

// ============================================================================
// Command Socket Methods
// ============================================================================

func (hc *HyprlandClient) sendCommand(command string) ([]byte, error) {
	socketPath := fmt.Sprintf("/tmp/hypr/%s/.socket.sock", hc.signature)
	
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Hyprland: %v", err)
	}
	defer conn.Close()

	// Send command
	if _, err := conn.Write([]byte(command)); err != nil {
		return nil, err
	}

	// Read response
	buf := make([]byte, 16384)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

// GetActiveWorkspace returns the currently active workspace
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

// GetWorkspaces returns all workspaces
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

// GetActiveWindow returns the currently focused window
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

// GetWindows returns all windows
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

// GetMonitors returns all monitors
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

// GetActiveMonitor returns the currently focused monitor
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

// ============================================================================
// Dispatch Commands
// ============================================================================

// SwitchWorkspace switches to a specific workspace
func (hc *HyprlandClient) SwitchWorkspace(workspace int) error {
	cmd := fmt.Sprintf("dispatch workspace %d", workspace)
	_, err := hc.sendCommand(cmd)
	return err
}

// SwitchWorkspaceByName switches to a workspace by name
func (hc *HyprlandClient) SwitchWorkspaceByName(name string) error {
	cmd := fmt.Sprintf("dispatch workspace name:%s", name)
	_, err := hc.sendCommand(cmd)
	return err
}

// MoveToWorkspace moves active window to workspace
func (hc *HyprlandClient) MoveToWorkspace(workspace int) error {
	cmd := fmt.Sprintf("dispatch movetoworkspace %d", workspace)
	_, err := hc.sendCommand(cmd)
	return err
}

// ToggleFullscreen toggles fullscreen for active window
func (hc *HyprlandClient) ToggleFullscreen() error {
	_, err := hc.sendCommand("dispatch fullscreen")
	return err
}

// KillActiveWindow closes the active window
func (hc *HyprlandClient) KillActiveWindow() error {
	_, err := hc.sendCommand("dispatch killactive")
	return err
}

// ToggleFloating toggles floating mode for active window
func (hc *HyprlandClient) ToggleFloating() error {
	_, err := hc.sendCommand("dispatch togglefloating")
	return err
}

// FocusMonitor switches focus to a monitor
func (hc *HyprlandClient) FocusMonitor(monitor string) error {
	cmd := fmt.Sprintf("dispatch focusmonitor %s", monitor)
	_, err := hc.sendCommand(cmd)
	return err
}

// MoveWorkspaceToMonitor moves a workspace to a different monitor
func (hc *HyprlandClient) MoveWorkspaceToMonitor(workspace int, monitor string) error {
	cmd := fmt.Sprintf("dispatch moveworkspacetomonitor %d %s", workspace, monitor)
	_, err := hc.sendCommand(cmd)
	return err
}

// ============================================================================
// Event Socket Methods
// ============================================================================

// StartEventListener connects to the event socket and starts listening
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

// readEvents continuously reads and dispatches events
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

// parseEvent parses a raw event line
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

// dispatchEvent sends event to all listeners
func (hc *HyprlandClient) dispatchEvent(event HyprlandEvent) {
	hc.eventMux.RLock()
	defer hc.eventMux.RUnlock()

	for _, listener := range hc.listeners {
		select {
		case listener <- event:
		default:
			// Channel full, skip
		}
	}
}

// Subscribe returns a channel that receives Hyprland events
func (hc *HyprlandClient) Subscribe() chan HyprlandEvent {
	hc.eventMux.Lock()
	defer hc.eventMux.Unlock()

	ch := make(chan HyprlandEvent, 100)
	hc.listeners = append(hc.listeners, ch)
	return ch
}

// Unsubscribe removes a listener channel
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

// Close closes all connections
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

// ============================================================================
// Convenience Functions (backward compatible)
// ============================================================================

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

// ============================================================================
// Helper Functions
// ============================================================================

// GetWorkspaceWindows returns all windows in a specific workspace
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

// IsWorkspaceEmpty checks if a workspace has no windows
func (hc *HyprlandClient) IsWorkspaceEmpty(workspaceID int) (bool, error) {
	windows, err := hc.GetWorkspaceWindows(workspaceID)
	if err != nil {
		return false, err
	}
	return len(windows) == 0, nil
}

// GetWorkspaceByName finds a workspace by name
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
```

---

## Complete Event System

### Create `hypr_events.go` - Advanced Event Handler

```go
package main

import (
	"log"
	"strconv"
	"sync"
)

// ============================================================================
// Event Handler
// ============================================================================

type HyprlandEventHandler struct {
	client    *HyprlandClient
	callbacks map[string][]EventCallback
	mu        sync.RWMutex
	events    chan HyprlandEvent
	stopChan  chan struct{}
}

type EventCallback func(event HyprlandEvent)

func NewHyprlandEventHandler(client *HyprlandClient) *HyprlandEventHandler {
	return &HyprlandEventHandler{
		client:    client,
		callbacks: make(map[string][]EventCallback),
		stopChan:  make(chan struct{}),
	}
}

// On registers a callback for a specific event type
func (h *HyprlandEventHandler) On(eventType string, callback EventCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.callbacks[eventType] = append(h.callbacks[eventType], callback)
}

// Start begins listening for events
func (h *HyprlandEventHandler) Start() error {
	if err := h.client.StartEventListener(); err != nil {
		return err
	}

	h.events = h.client.Subscribe()
	
	go h.handleEvents()
	
	return nil
}

// Stop stops the event handler
func (h *HyprlandEventHandler) Stop() {
	close(h.stopChan)
	h.client.Unsubscribe(h.events)
}

// handleEvents processes incoming events
func (h *HyprlandEventHandler) handleEvents() {
	for {
		select {
		case event := <-h.events:
			h.processEvent(event)
		case <-h.stopChan:
			return
		}
	}
}

// processEvent dispatches event to registered callbacks
func (h *HyprlandEventHandler) processEvent(event HyprlandEvent) {
	h.mu.RLock()
	callbacks := h.callbacks[event.Type]
	h.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(event)
	}
}

// ============================================================================
// Typed Event Handlers
// ============================================================================

type WorkspaceCallback func(workspaceID int, workspaceName string)
type WindowCallback func(windowClass string, windowTitle string)
type MonitorCallback func(monitorName string, workspaceName string)
type WindowOpenCallback func(address string, workspace string, class string, title string)
type WindowCloseCallback func(address string)

// OnWorkspaceChange registers a workspace change callback
func (h *HyprlandEventHandler) OnWorkspaceChange(callback WorkspaceCallback) {
	h.On("workspace", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			// Try to parse as ID
			if id, err := strconv.Atoi(event.Data[0]); err == nil {
				callback(id, event.Data[0])
			} else {
				// Named workspace
				callback(0, event.Data[0])
			}
		}
	})
}

// OnActiveWindow registers an active window change callback
func (h *HyprlandEventHandler) OnActiveWindow(callback WindowCallback) {
	h.On("activewindow", func(event HyprlandEvent) {
		if len(event.Data) >= 2 {
			callback(event.Data[0], event.Data[1])
		}
	})
}

// OnMonitorFocus registers a monitor focus callback
func (h *HyprlandEventHandler) OnMonitorFocus(callback MonitorCallback) {
	h.On("focusedmon", func(event HyprlandEvent) {
		if len(event.Data) >= 2 {
			callback(event.Data[0], event.Data[1])
		}
	})
}

// OnWindowOpen registers a window open callback
func (h *HyprlandEventHandler) OnWindowOpen(callback WindowOpenCallback) {
	h.On("openwindow", func(event HyprlandEvent) {
		if len(event.Data) >= 4 {
			callback(event.Data[0], event.Data[1], event.Data[2], event.Data[3])
		}
	})
}

// OnWindowClose registers a window close callback
func (h *HyprlandEventHandler) OnWindowClose(callback WindowCloseCallback) {
	h.On("closewindow", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0])
		}
	})
}

// OnFullscreenToggle registers a fullscreen toggle callback
func (h *HyprlandEventHandler) OnFullscreenToggle(callback func(fullscreen bool)) {
	h.On("fullscreen", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0] == "1")
		}
	})
}

// OnWorkspaceCreate registers a workspace creation callback
func (h *HyprlandEventHandler) OnWorkspaceCreate(callback func(workspaceName string)) {
	h.On("createworkspace", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0])
		}
	})
}

// OnWorkspaceDestroy registers a workspace destruction callback
func (h *HyprlandEventHandler) OnWorkspaceDestroy(callback func(workspaceName string)) {
	h.On("destroyworkspace", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0])
		}
	})
}

// ============================================================================
// Example Usage in GTK Statusbar
// ============================================================================

func (sb *GTKStatusbar) startHyprlandListener() {
	client, err := NewHyprlandClient()
	if err != nil {
		log.Printf("Failed to create Hyprland client: %v", err)
		return
	}

	handler := NewHyprlandEventHandler(client)

	// Register event callbacks
	handler.OnWorkspaceChange(func(id int, name string) {
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	})

	handler.OnActiveWindow(func(class string, title string) {
		glib.IdleAdd(func() {
			sb.updateWindowTitle()
		})
	})

	handler.OnMonitorFocus(func(monitor string, workspace string) {
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	})

	handler.OnWindowOpen(func(address string, workspace string, class string, title string) {
		log.Printf("Window opened: %s - %s", class, title)
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	})

	handler.OnWindowClose(func(address string) {
		log.Printf("Window closed: %s", address)
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	})

	handler.OnWorkspaceCreate(func(name string) {
		log.Printf("Workspace created: %s", name)
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	})

	handler.OnWorkspaceDestroy(func(name string) {
		log.Printf("Workspace destroyed: %s", name)
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	})

	// Start listening
	if err := handler.Start(); err != nil {
		log.Printf("Failed to start event handler: %v", err)
		return
	}

	sb.hyprHandler = handler
}
```

---

## Workspace Management

### Advanced Workspace Widget

Create `hypr_workspaces.go`:

```go
package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/gotk3/gotk3/gtk"
)

// ============================================================================
// Workspace Manager
// ============================================================================

type WorkspaceManager struct {
	client          *HyprlandClient
	container       *gtk.Box
	buttons         map[int]*gtk.Button
	activeWorkspace int
	maxWorkspaces   int
	showEmpty       bool
	persistent      []int // Always show these workspaces
}

func NewWorkspaceManager(client *HyprlandClient, maxWorkspaces int) *WorkspaceManager {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.SetName("workspace-box")

	return &WorkspaceManager{
		client:        client,
		container:     box,
		buttons:       make(map[int]*gtk.Button),
		maxWorkspaces: maxWorkspaces,
		showEmpty:     false,
		persistent:    []int{1, 2, 3, 4, 5}, // Always show first 5
	}
}

// Update refreshes the workspace display
func (wm *WorkspaceManager) Update() error {
	// Get all workspaces from Hyprland
	workspaces, err := wm.client.GetWorkspaces()
	if err != nil {
		return err
	}

	// Get active workspace
	activeWS, err := wm.client.GetActiveWorkspace()
	if err != nil {
		return err
	}
	wm.activeWorkspace = activeWS.ID

	// Build set of existing workspace IDs
	existingWS := make(map[int]bool)
	for _, ws := range workspaces {
		existingWS[ws.ID] = true
	}

	// Add persistent workspaces
	for _, id := range wm.persistent {
		existingWS[id] = true
	}

	// Get sorted list of workspace IDs
	var wsIDs []int
	for id := range existingWS {
		if id > 0 && id <= wm.maxWorkspaces {
			wsIDs = append(wsIDs, id)
		}
	}
	sort.Ints(wsIDs)

	// Update buttons
	wm.updateButtons(wsIDs, workspaces)

	return nil
}

// updateButtons creates/updates workspace buttons
func (wm *WorkspaceManager) updateButtons(wsIDs []int, workspaces []HyprlandWorkspace) {
	// Remove buttons that don't exist anymore
	for id, btn := range wm.buttons {
		found := false
		for _, wsID := range wsIDs {
			if wsID == id {
				found = true
				break
			}
		}
		if !found {
			wm.container.Remove(btn)
			delete(wm.buttons, id)
		}
	}

	// Create/update buttons for each workspace
	for _, id := range wsIDs {
		btn, exists := wm.buttons[id]
		
		if !exists {
			// Create new button
			btn = wm.createWorkspaceButton(id)
			wm.buttons[id] = btn
			
			// Insert in sorted order
			position := 0
			for i, existingID := range wsIDs {
				if existingID < id {
					position = i + 1
				}
			}
			wm.container.Add(btn)
			wm.container.ReorderChild(btn, position)
		}

		// Update button state
		wm.updateButtonState(btn, id, workspaces)
	}

	wm.container.ShowAll()
}

// createWorkspaceButton creates a new workspace button
func (wm *WorkspaceManager) createWorkspaceButton(id int) *gtk.Button {
	btn, _ := gtk.ButtonNewWithLabel(fmt.Sprintf("%d", id))
	btn.SetName("workspace-button")
	btn.SetRelief(gtk.RELIEF_NONE)
	btn.SetCanFocus(false)

	// Click handler
	btn.Connect("clicked", func() {
		if err := wm.client.SwitchWorkspace(id); err != nil {
			log.Printf("Failed to switch workspace: %v", err)
		}
	})

	// Right-click handler - move active window
	btn.Connect("button-press-event", func(_ *gtk.Button, event *gdk.Event) bool {
		btnEvent := gdk.EventButtonNewFromEvent(event)
		if btnEvent.Button() == 3 { // Right click
			if err := wm.client.MoveToWorkspace(id); err != nil {
				log.Printf("Failed to move window: %v", err)
			}
			return true
		}
		return false
	})

	return btn
}

// updateButtonState updates the visual state of a button
func (wm *WorkspaceManager) updateButtonState(btn *gtk.Button, id int, workspaces []HyprlandWorkspace) {
	ctx, _ := btn.GetStyleContext()

	// Remove all state classes
	ctx.RemoveClass("active")
	ctx.RemoveClass("occupied")
	ctx.RemoveClass("urgent")
	ctx.RemoveClass("empty")

	// Find workspace info
	var ws *HyprlandWorkspace
	for i := range workspaces {
		if workspaces[i].ID == id {
			ws = &workspaces[i]
			break
		}
	}

	// Set state classes
	if id == wm.activeWorkspace {
		ctx.AddClass("active")
	} else if ws != nil && ws.Windows > 0 {
		ctx.AddClass("occupied")
	} else {
		ctx.AddClass("empty")
	}

	// Set window count as tooltip
	if ws != nil {
		tooltip := fmt.Sprintf("Workspace %d", id)
		if ws.Windows > 0 {
			tooltip = fmt.Sprintf("%s (%d windows)", tooltip, ws.Windows)
		}
		if ws.HasFullscreen {
			tooltip += " [F]"
		}
		btn.SetTooltipText(tooltip)
	}
}

// GetContainer returns the GTK container widget
func (wm *WorkspaceManager) GetContainer() *gtk.Box {
	return wm.container
}

// SetShowEmpty controls whether empty workspaces are shown
func (wm *WorkspaceManager) SetShowEmpty(show bool) {
	wm.showEmpty = show
}

// SetPersistent sets which workspaces are always shown
func (wm *WorkspaceManager) SetPersistent(ids []int) {
	wm.persistent = ids
}
```

### CSS for Advanced Workspaces

```css
/* Workspace states */
#workspace-button {
    background-color: transparent;
    border: 1px solid #4a4458;
    border-radius: 4px;
    color: #6e6a86;
    padding: 2px 10px;
    margin: 0px 2px;
    min-width: 25px;
    transition: all 200ms ease;
}

#workspace-button:hover {
    background-color: rgba(215, 186, 255, 0.2);
    border-color: #D7BAFF;
}

#workspace-button.active {
    background-color: #D7BAFF;
    color: #16121B;
    font-weight: bold;
    border-color: #D7BAFF;
}

#workspace-button.occupied {
    border-color: #D9BDE3;
    color: #D9BDE3;
}

#workspace-button.empty {
    border-color: #4a4458;
    color: #6e6a86;
    opacity: 0.6;
}

#workspace-button.urgent {
    background-color: #FFB4AB;
    color: #16121B;
    border-color: #FFB4AB;
    animation: urgentBlink 1s infinite;
}

@keyframes urgentBlink {
    0%, 100% { opacity: 1.0; }
    50% { opacity: 0.6; }
}
```

---

## Window Information

### Window Title Widget

Create `hypr_window.go`:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// ============================================================================
// Window Title Widget
// ============================================================================

type WindowTitleWidget struct {
	client    *HyprlandClient
	container *gtk.Box
	icon      *gtk.Label
	title     *gtk.Label
	maxLength int
}

func NewWindowTitleWidget(client *HyprlandClient, maxLength int) *WindowTitleWidget {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	box.SetName("window-title-box")

	icon, _ := gtk.LabelNew("")
	icon.SetName("window-icon")
	icon.SetUseMarkup(true)

	title, _ := gtk.LabelNew("")
	title.SetName("window-title")
	title.SetEllipsize(pango.ELLIPSIZE_END)
	title.SetMaxWidthChars(maxLength)
	title.SetSelectable(false)

	box.PackStart(icon, false, false, 0)
	box.PackStart(title, false, false, 0)

	return &WindowTitleWidget{
		client:    client,
		container: box,
		icon:      icon,
		title:     title,
		maxLength: maxLength,
	}
}

// Update refreshes the window title display
func (wtw *WindowTitleWidget) Update() error {
	window, err := wtw.client.GetActiveWindow()
	if err != nil || window == nil {
		wtw.container.Hide()
		return err
	}

	// Get icon for window class
	iconText := wtw.getIconForClass(window.Class)
	wtw.icon.SetMarkup(fmt.Sprintf("<span size='large'>%s</span>", iconText))

	// Set title
	displayTitle := window.Title
	if displayTitle == "" {
		displayTitle = window.Class
	}

	// Add indicators
	indicators := ""
	if window.Fullscreen {
		indicators += " [F]"
	}
	if window.Floating {
		indicators += " [~]"
	}
	if window.Pinned {
		indicators += " [P]"
	}

	wtw.title.SetText(displayTitle + indicators)
	wtw.container.Show()

	return nil
}

// getIconForClass returns an appropriate icon for window class
func (wtw *WindowTitleWidget) getIconForClass(class string) string {
	class = strings.ToLower(class)

	iconMap := map[string]string{
		"kitty":         "",
		"alacritty":     "",
		"foot":          "",
		"firefox":       "",
		"chromium":      "",
		"chrome":        "",
		"brave":         "",
		"code":          "",
		"nvim":          "",
		"vim":           "",
		"discord":       "󰙯",
		"spotify":       "",
		"thunar":        "",
		"nautilus":      "",
		"dolphin":       "",
		"gimp":          "",
		"inkscape":      "",
		"blender":       "󰂫",
		"steam":         "",
		"lutris":        "",
		"obs":           "",
		"mpv":           "",
		"vlc":           "󰕼",
		"thunderbird":   "",
		"slack":         "󰒱",
		"telegram":      "",
		"signal":        "",
		"zoom":          "",
	}

	for key, icon := range iconMap {
		if strings.Contains(class, key) {
			return icon
		}
	}

	return "" // Default icon
}

// GetContainer returns the GTK container
func (wtw *WindowTitleWidget) GetContainer() *gtk.Box {
	return wtw.container
}

// SetMaxLength sets maximum title length
func (wtw *WindowTitleWidget) SetMaxLength(length int) {
	wtw.maxLength = length
	wtw.title.SetMaxWidthChars(length)
}
```

---

## Monitor Support

### Multi-Monitor Workspace Management

Create `hypr_monitors.go`:

```go
package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/gotk3/gotk3/gtk"
)

// ============================================================================
// Monitor Workspace Manager
// ============================================================================

type MonitorWorkspaceManager struct {
	client     *HyprlandClient
	containers map[string]*gtk.Box // Monitor name -> container
	managers   map[string]*WorkspaceManager
}

func NewMonitorWorkspaceManager(client *HyprlandClient) *MonitorWorkspaceManager {
	return &MonitorWorkspaceManager{
		client:     client,
		containers: make(map[string]*gtk.Box),
		managers:   make(map[string]*WorkspaceManager),
	}
}

// Update refreshes workspace display for all monitors
func (mwm *MonitorWorkspaceManager) Update() error {
	monitors, err := mwm.client.GetMonitors()
	if err != nil {
		return err
	}

	workspaces, err := mwm.client.GetWorkspaces()
	if err != nil {
		return err
	}

	// Group workspaces by monitor
	monitorWorkspaces := make(map[string][]HyprlandWorkspace)
	for _, ws := range workspaces {
		monitorWorkspaces[ws.Monitor] = append(monitorWorkspaces[ws.Monitor], ws)
	}

	// Update each monitor's workspaces
	for _, mon := range monitors {
		if err := mwm.updateMonitor(mon, monitorWorkspaces[mon.Name]); err != nil {
			log.Printf("Failed to update monitor %s: %v", mon.Name, err)
		}
	}

	return nil
}

// updateMonitor updates workspace display for a specific monitor
func (mwm *MonitorWorkspaceManager) updateMonitor(
	monitor HyprlandMonitor,
	workspaces []HyprlandWorkspace,
) error {
	// Get or create workspace manager for this monitor
	manager, exists := mwm.managers[monitor.Name]
	if !exists {
		manager = NewWorkspaceManager(mwm.client, 10)
		mwm.managers[monitor.Name] = manager
	}

	// Get workspace IDs for this monitor
	var wsIDs []int
	for _, ws := range workspaces {
		wsIDs = append(wsIDs, ws.ID)
	}
	sort.Ints(wsIDs)

	// Update buttons
	manager.updateButtons(wsIDs, workspaces)

	return nil
}

// GetContainer returns workspace container for a monitor
func (mwm *MonitorWorkspaceManager) GetContainer(monitorName string) *gtk.Box {
	if manager, exists := mwm.managers[monitorName]; exists {
		return manager.GetContainer()
	}
	return nil
}

// GetFocusedContainer returns the container for the focused monitor
func (mwm *MonitorWorkspaceManager) GetFocusedContainer() *gtk.Box {
	monitor, err := mwm.client.GetActiveMonitor()
	if err != nil {
		return nil
	}
	return mwm.GetContainer(monitor.Name)
}
```

### Monitor Indicator Widget

```go
package main

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

type MonitorIndicator struct {
	client    *HyprlandClient
	container *gtk.Box
	labels    map[string]*gtk.Label
}

func NewMonitorIndicator(client *HyprlandClient) *MonitorIndicator {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.SetName("monitor-indicator")

	return &MonitorIndicator{
		client:    client,
		container: box,
		labels:    make(map[string]*gtk.Label),
	}
}

func (mi *MonitorIndicator) Update() error {
	monitors, err := mi.client.GetMonitors()
	if err != nil {
		return err
	}

	// Clear existing labels
	mi.container.GetChildren().Foreach(func(item interface{}) {
		if widget, ok := item.(*gtk.Widget); ok {
			mi.container.Remove(widget)
		}
	})

	// Create label for each monitor
	for _, mon := range monitors {
		label, _ := gtk.LabelNew("")
		label.SetName("monitor-label")

		// Format: Monitor name + active workspace
		text := fmt.Sprintf("󰍹 %s:%d", mon.Name, mon.ActiveWorkspace.ID)
		label.SetText(text)

		// Highlight focused monitor
		ctx, _ := label.GetStyleContext()
		if mon.Focused {
			ctx.AddClass("focused")
		} else {
			ctx.RemoveClass("focused")
		}

		mi.container.PackStart(label, false, false, 0)
		mi.labels[mon.Name] = label
	}

	mi.container.ShowAll()
	return nil
}

func (mi *MonitorIndicator) GetContainer() *gtk.Box {
	return mi.container
}
```

---

## Advanced Features

### Workspace Previews (Thumbnails)

```go
package main

import (
	"fmt"
	"os/exec"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

type WorkspacePreview struct {
	client      *HyprlandClient
	workspaceID int
	image       *gtk.Image
	popover     *gtk.Popover
}

func NewWorkspacePreview(client *HyprlandClient, wsID int) *WorkspacePreview {
	image, _ := gtk.ImageNew()
	popover, _ := gtk.PopoverNew(nil)

	return &WorkspacePreview{
		client:      client,
		workspaceID: wsID,
		image:       image,
		popover:     popover,
	}
}

// CapturePreview takes a screenshot of the workspace
func (wp *WorkspacePreview) CapturePreview() error {
	// Use grim to capture workspace
	output := fmt.Sprintf("/tmp/ws-preview-%d.png", wp.workspaceID)
	
	cmd := exec.Command("grim", "-o", wp.getMonitorForWorkspace(), output)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Load into image widget
	pixbuf, err := gdk.PixbufNewFromFile(output)
	if err != nil {
		return err
	}

	// Scale down
	scaled, err := pixbuf.ScaleSimple(320, 180, gdk.INTERP_BILINEAR)
	if err != nil {
		return err
	}

	wp.image.SetFromPixbuf(scaled)
	return nil
}

func (wp *WorkspacePreview) getMonitorForWorkspace() string {
	workspaces, _ := wp.client.GetWorkspaces()
	for _, ws := range workspaces {
		if ws.ID == wp.workspaceID {
			return ws.Monitor
		}
	}
	return ""
}

// Show displays the preview popover
func (wp *WorkspacePreview) Show(relativeTo gtk.IWidget) {
	wp.popover.SetRelativeTo(relativeTo)
	wp.popover.Add(wp.image)
	wp.popover.ShowAll()
	wp.popover.Popup()
}
```

### Scratchpad Management

```go
package main

import (
	"fmt"

	"github.com/gotk3/gotk3/gtk"
)

type ScratchpadManager struct {
	client    *HyprlandClient
	container *gtk.Box
	button    *gtk.Button
	count     int
}

func NewScratchpadManager(client *HyprlandClient) *ScratchpadManager {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	button, _ := gtk.ButtonNewWithLabel("󱂬")
	button.SetName("scratchpad-button")
	button.SetTooltipText("Scratchpad")

	box.PackStart(button, false, false, 0)

	sm := &ScratchpadManager{
		client:    client,
		container: box,
		button:    button,
	}

	// Click handler - toggle scratchpad
	button.Connect("clicked", func() {
		sm.toggleScratchpad()
	})

	return sm
}

func (sm *ScratchpadManager) Update() error {
	// Get windows in special workspace
	windows, err := sm.client.GetWorkspaceWindows(-99) // Scratchpad workspace
	if err != nil {
		return err
	}

	sm.count = len(windows)

	// Update button label
	if sm.count > 0 {
		sm.button.SetLabel(fmt.Sprintf("󱂬 %d", sm.count))
		sm.container.Show()
	} else {
		sm.button.SetLabel("󱂬")
		sm.container.Hide()
	}

	return nil
}

func (sm *ScratchpadManager) toggleScratchpad() {
	cmd := "dispatch togglespecialworkspace"
	sm.client.sendCommand(cmd)
}

func (sm *ScratchpadManager) GetContainer() *gtk.Box {
	return sm.container
}
```

### Submap Indicator

```go
package main

import (
	"github.com/gotk3/gotk3/gtk"
)

type SubmapIndicator struct {
	container     *gtk.Box
	label         *gtk.Label
	currentSubmap string
}

func NewSubmapIndicator() *SubmapIndicator {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.SetName("submap-indicator")

	label, _ := gtk.LabelNew("")
	label.SetName("submap-label")
	label.SetUseMarkup(true)

	box.PackStart(label, false, false, 0)
	box.Hide() // Hidden by default

	return &SubmapIndicator{
		container: box,
		label:     label,
	}
}

func (si *SubmapIndicator) SetSubmap(submap string) {
	si.currentSubmap = submap

	if submap == "" || submap == "default" {
		si.container.Hide()
		return
	}

	// Show submap with icon
	si.label.SetMarkup(fmt.Sprintf("<b> %s</b>", submap))
	si.container.Show()
}

func (si *SubmapIndicator) GetContainer() *gtk.Box {
	return si.container
}
```

---

## Performance Optimization

### Event Debouncing

```go
package main

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	duration time.Duration
}

func NewDebouncer(duration time.Duration) *Debouncer {
	return &Debouncer{
		duration: duration,
	}
}

func (d *Debouncer) Debounce(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.duration, f)
}

// Usage in event handler
func (sb *GTKStatusbar) startHyprlandListener() {
	// ... existing code ...

	workspaceDebouncer := NewDebouncer(50 * time.Millisecond)
	windowDebouncer := NewDebouncer(100 * time.Millisecond)

	handler.OnWorkspaceChange(func(id int, name string) {
		workspaceDebouncer.Debounce(func() {
			glib.IdleAdd(sb.updateWorkspaces)
		})
	})

	handler.OnActiveWindow(func(class string, title string) {
		windowDebouncer.Debounce(func() {
			glib.IdleAdd(sb.updateWindowTitle)
		})
	})
}
```

### Caching Hyprland State

```go
package main

import (
	"sync"
	"time"
)

type HyprlandCache struct {
	client     *HyprlandClient
	workspaces []HyprlandWorkspace
	windows    []HyprlandWindow
	monitors   []HyprlandMonitor
	mu         sync.RWMutex
	lastUpdate time.Time
	ttl        time.Duration
}

func NewHyprlandCache(client *HyprlandClient, ttl time.Duration) *HyprlandCache {
	return &HyprlandCache{
		client: client,
		ttl:    ttl,
	}
}

func (hc *HyprlandCache) GetWorkspaces() ([]HyprlandWorkspace, error) {
	hc.mu.RLock()
	if time.Since(hc.lastUpdate) < hc.ttl {
		ws := hc.workspaces
		hc.mu.RUnlock()
		return ws, nil
	}
	hc.mu.RUnlock()

	// Fetch fresh data
	ws, err := hc.client.GetWorkspaces()
	if err != nil {
		return nil, err
	}

	hc.mu.Lock()
	hc.workspaces = ws
	hc.lastUpdate = time.Now()
	hc.mu.Unlock()

	return ws, nil
}

// Similar methods for Windows and Monitors...

func (hc *HyprlandCache) Invalidate() {
	hc.mu.Lock()
	hc.lastUpdate = time.Time{}
	hc.mu.Unlock()
}
```

---

## Testing and Debugging

### Hyprland Event Monitor

Create `hyprctl-monitor.go` for debugging:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	verbose := flag.Bool("v", false, "Verbose output")
	eventType := flag.String("type", "", "Filter by event type")
	flag.Parse()

	client, err := NewHyprlandClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	handler := NewHyprlandEventHandler(client)

	// Log all events
	handler.On("*", func(event HyprlandEvent) {
		if *eventType != "" && event.Type != *eventType {
			return
		}

		if *verbose {
			fmt.Printf("[%s] %v\n", event.Type, event.Data)
		} else {
			fmt.Printf("%s\n", event.Type)
		}
	})

	if err := handler.Start(); err != nil {
		log.Fatalf("Failed to start handler: %v", err)
	}

	fmt.Println("Monitoring Hyprland events... (Ctrl+C to quit)")

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	handler.Stop()
	client.Close()
}
```

### Testing Utilities

```go
package main

import (
	"testing"
	"time"
)

func TestHyprlandClient(t *testing.T) {
	client, err := NewHyprlandClient()
	if err != nil {
		t.Skip("Not running under Hyprland")
	}

	t.Run("GetWorkspaces", func(t *testing.T) {
		workspaces, err := client.GetWorkspaces()
		if err != nil {
			t.Fatalf("Failed to get workspaces: %v", err)
		}
		if len(workspaces) == 0 {
			t.Error("Expected at least one workspace")
		}
	})

	t.Run("GetActiveWorkspace", func(t *testing.T) {
		ws, err := client.GetActiveWorkspace()
		if err != nil {
			t.Fatalf("Failed to get active workspace: %v", err)
		}
		if ws.ID == 0 {
			t.Error("Invalid workspace ID")
		}
	})

	t.Run("SwitchWorkspace", func(t *testing.T) {
		// Get current workspace
		current, _ := client.GetActiveWorkspace()
		
		// Switch to different workspace
		targetWS := 2
		if current.ID == 2 {
			targetWS = 1
		}

		err := client.SwitchWorkspace(targetWS)
		if err != nil {
			t.Fatalf("Failed to switch workspace: %v", err)
		}

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Verify
		newWS, _ := client.GetActiveWorkspace()
		if newWS.ID != targetWS {
			t.Errorf("Expected workspace %d, got %d", targetWS, newWS.ID)
		}

		// Switch back
		client.SwitchWorkspace(current.ID)
	})
}

func TestEventHandler(t *testing.T) {
	client, err := NewHyprlandClient()
	if err != nil {
		t.Skip("Not running under Hyprland")
	}

	handler := NewHyprlandEventHandler(client)

	eventReceived := make(chan bool, 1)

	handler.OnWorkspaceChange(func(id int, name string) {
		select {
		case eventReceived <- true:
		default:
		}
	})

	if err := handler.Start(); err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer handler.Stop()

	// Trigger a workspace change
	current, _ := client.GetActiveWorkspace()
	targetWS := 2
	if current.ID == 2 {
		targetWS = 1
	}

	client.SwitchWorkspace(targetWS)

	// Wait for event
	select {
	case <-eventReceived:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Did not receive workspace change event")
	}

	// Switch back
	client.SwitchWorkspace(current.ID)
}
```

---

## Integration Examples

### Complete GTK Integration

Update `gtk.go` to use all Hyprland features:

```go
func (sb *GTKStatusbar) createUI() error {
	// ... existing code ...

	// Create Hyprland client
	hyprClient, err := NewHyprlandClient()
	if err != nil {
		log.Printf("Hyprland not available: %v", err)
	} else {
		sb.hyprClient = hyprClient

		// Create Hyprland widgets
		sb.workspaceManager = NewWorkspaceManager(hyprClient, 10)
		sb.windowTitle = NewWindowTitleWidget(hyprClient, 50)
		sb.monitorIndicator = NewMonitorIndicator(hyprClient)
		sb.scratchpad = NewScratchpadManager(hyprClient)
		sb.submapIndicator = NewSubmapIndicator()

		// Add to layout
		sb.leftBox.PackStart(sb.workspaceManager.GetContainer(), false, false, 0)
		sb.centerBox.PackStart(sb.windowTitle.GetContainer(), false, false, 0)
		sb.rightBox.PackStart(sb.monitorIndicator.GetContainer(), false, false, 5)
		sb.rightBox.PackStart(sb.scratchpad.GetContainer(), false, false, 0)
		sb.rightBox.PackStart(sb.submapIndicator.GetContainer(), false, false, 0)

		// Start event listener
		sb.startHyprlandListener()
	}

	// ... rest of UI creation ...

	return nil
}
```

### Event Handler Setup

```go
func (sb *GTKStatusbar) startHyprlandListener() {
	handler := NewHyprlandEventHandler(sb.hyprClient)

	// Workspace events
	handler.OnWorkspaceChange(func(id int, name string) {
		glib.IdleAdd(func() {
			sb.workspaceManager.Update()
			sb.monitorIndicator.Update()
		})
	})

	// Window events
	handler.OnActiveWindow(func(class string, title string) {
		glib.IdleAdd(func() {
			sb.windowTitle.Update()
		})
	})

	handler.OnWindowOpen(func(address string, workspace string, class string, title string) {
		glib.IdleAdd(func() {
			sb.workspaceManager.Update()
			sb.scratchpad.Update()
		})
	})

	handler.OnWindowClose(func(address string) {
		glib.IdleAdd(func() {
			sb.workspaceManager.Update()
			sb.scratchpad.Update()
		})
	})

	// Monitor events
	handler.OnMonitorFocus(func(monitor string, workspace string) {
		glib.IdleAdd(func() {
			sb.monitorIndicator.Update()
			sb.workspaceManager.Update()
		})
	})

	// Submap events
	handler.On("submap", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			glib.IdleAdd(func() {
				sb.submapIndicator.SetSubmap(event.Data[0])
			})
		}
	})

	// Fullscreen events
	handler.OnFullscreenToggle(func(fullscreen bool) {
		glib.IdleAdd(func() {
			sb.windowTitle.Update()
		})
	})

	if err := handler.Start(); err != nil {
		log.Printf("Failed to start Hyprland event handler: %v", err)
	}

	sb.hyprHandler = handler
}
```

---

## Configuration

### Hyprland-Specific Config

Extend `config.json`:

```json
{
  "refresh_interval": 1,
  "modules": ["workspaces", "window", "clock", "cpu", "memory", "battery"],
  "colors": {
    "primary": "#D7BAFF",
    "surface": "#16121B",
    "text": "#E9DFEE"
  },
  "hyprland": {
    "workspace": {
      "max_workspaces": 10,
      "show_empty": false,
      "persistent": [1, 2, 3, 4, 5],
      "format": "{id}",
      "format_icons": {
        "1": "",
        "2": "",
        "3": "",
        "4": "󰙯",
        "5": ""
      }
    },
    "window": {
      "max_length": 50,
      "separate_outputs": true,
      "rewrite": {
        "class": {
          "firefox": "Firefox",
          "kitty": "Terminal"
        }
      }
    },
    "monitor": {
      "show_all": true,
      "format": "{name}:{workspace}"
    }
  }
}
```

---

## Conclusion

You now have complete Hyprland integration with:

✅ Real-time event system  
✅ Workspace management with dynamic creation/destruction  
✅ Window title display with icons  
✅ Multi-monitor support  
✅ Scratchpad indicator  
✅ Submap display  
✅ Performance optimizations  
✅ Comprehensive testing utilities  
✅ Full GTK integration  

This implementation provides professional-grade Hyprland integration that rivals and extends the functionality of Waybar.