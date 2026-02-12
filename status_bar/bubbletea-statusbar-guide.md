# Building a TUI Statusbar with Bubbletea

A complete guide to creating a custom statusbar that matches your terminal aesthetic using Go and Bubbletea.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Project Setup](#project-setup)
3. [Understanding Bubbletea Architecture](#understanding-bubbletea-architecture)
4. [Building the Basic Statusbar](#building-the-basic-statusbar)
5. [Adding System Information](#adding-system-information)
6. [Styling with Lipgloss](#styling-with-lipgloss)
7. [Integration with Wayland/Hyprland](#integration-with-waylandhyprland)
8. [Advanced Features](#advanced-features)
9. [Running as a Service](#running-as-a-service)

---

## Prerequisites

### Required Software

```bash
# Install Go (1.21 or later)
# On Arch Linux
sudo pacman -S go

# On Ubuntu/Debian
sudo apt install golang-go

# On macOS
brew install go
```

### Verify Installation

```bash
go version
# Should output: go version go1.21.x or later
```

---

## Project Setup

### 1. Create Project Directory

```bash
mkdir ~/projects/tui-statusbar
cd ~/projects/tui-statusbar
```

### 2. Initialize Go Module

```bash
go mod init github.com/yourusername/tui-statusbar
```

### 3. Install Dependencies

```bash
# Core Bubbletea framework
go get github.com/charmbracelet/bubbletea

# Styling library
go get github.com/charmbracelet/lipgloss

# System information
go get github.com/shirou/gopsutil/v3/cpu
go get github.com/shirou/gopsutil/v3/mem
go get github.com/shirou/gopsutil/v3/disk
go get github.com/shirou/gopsutil/v3/net
go get github.com/shirou/gopsutil/v3/host

# Battery information
go get github.com/distatus/battery
```

### 4. Project Structure

```
tui-statusbar/
├── main.go           # Entry point
├── model.go          # Application state
├── update.go         # State updates
├── view.go           # Rendering
├── styles.go         # Lipgloss styles
├── sysinfo.go        # System information gathering
└── config.go         # Configuration
```

---

## Understanding Bubbletea Architecture

Bubbletea follows The Elm Architecture with three core components:

### 1. **Model** - Your application state
```go
type model struct {
    time     string
    cpu      float64
    memory   float64
    battery  int
    network  string
}
```

### 2. **Update** - How state changes over time
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages and update state
}
```

### 3. **View** - How to render the UI
```go
func (m model) View() string {
    // Return a string representation of your UI
}
```

---

## Building the Basic Statusbar

### Step 1: Create `main.go`

```go
package main

import (
    "fmt"
    "os"
    tea "github.com/charmbracelet/bubbletea"
)

func main() {
    p := tea.NewProgram(
        initialModel(),
        tea.WithAltScreen(),       // Use alternate screen buffer
        tea.WithMouseCellMotion(), // Enable mouse support if needed
    )

    if _, err := p.Run(); err != nil {
        fmt.Printf("Error running program: %v\n", err)
        os.Exit(1)
    }
}
```

### Step 2: Create `model.go`

```go
package main

import (
    "time"
    tea "github.com/charmbracelet/bubbletea"
)

type model struct {
    // Time
    currentTime time.Time

    // System stats
    cpuUsage    float64
    memUsage    float64
    diskUsage   float64
    
    // Network
    networkName  string
    networkState string
    
    // Battery
    batteryLevel int
    batteryState string
    
    // Window/workspace info (from Hyprland)
    activeWorkspace int
    windowTitle     string
    
    // Dimensions
    width  int
    height int
}

func initialModel() model {
    return model{
        currentTime:     time.Now(),
        cpuUsage:        0,
        memUsage:        0,
        diskUsage:       0,
        networkName:     "wlan0",
        networkState:    "disconnected",
        batteryLevel:    0,
        batteryState:    "unknown",
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
    )
}
```

### Step 3: Create `update.go`

```go
package main

import (
    "time"
    tea "github.com/charmbracelet/bubbletea"
)

// Message types
type tickMsg time.Time
type sysInfoMsg struct {
    cpu    float64
    mem    float64
    disk   float64
}
type batteryMsg struct {
    level int
    state string
}
type networkMsg struct {
    name  string
    state string
}

// Commands
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

// Update function
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    
    case tickMsg:
        m.currentTime = time.Time(msg)
        return m, tea.Batch(
            tickCmd(),
            getSystemInfo(),
            getBatteryInfo(),
        )
    
    case sysInfoMsg:
        m.cpuUsage = msg.cpu
        m.memUsage = msg.mem
        m.diskUsage = msg.disk
    
    case batteryMsg:
        m.batteryLevel = msg.level
        m.batteryState = msg.state
    
    case networkMsg:
        m.networkName = msg.name
        m.networkState = msg.state
    }
    
    return m, nil
}
```

---

## Adding System Information

### Create `sysinfo.go`

```go
package main

import (
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/distatus/battery"
    "math"
)

// Fetch CPU, Memory, and Disk usage
func fetchSystemStats() (float64, float64, float64) {
    // CPU
    cpuPercent, err := cpu.Percent(0, false)
    cpuUsage := 0.0
    if err == nil && len(cpuPercent) > 0 {
        cpuUsage = math.Round(cpuPercent[0]*10) / 10
    }
    
    // Memory
    memInfo, err := mem.VirtualMemory()
    memUsage := 0.0
    if err == nil {
        memUsage = math.Round(memInfo.UsedPercent*10) / 10
    }
    
    // Disk
    diskInfo, err := disk.Usage("/")
    diskUsage := 0.0
    if err == nil {
        diskUsage = math.Round(diskInfo.UsedPercent*10) / 10
    }
    
    return cpuUsage, memUsage, diskUsage
}

// Fetch battery information
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

// Get network information
func fetchNetworkInfo() (string, string) {
    // This is simplified - you'd want to check actual network state
    // Could use net.Interfaces() or read from /sys/class/net/
    return "wlan0", "connected"
}

// Get Hyprland workspace info (if using Hyprland)
func fetchHyprlandInfo() (int, string) {
    // Read from hyprctl or hyprland socket
    // This is a placeholder
    return 1, "nvim"
}
```

### Helper for Battery Icons

```go
package main

func getBatteryIcon(level int, state string) string {
    if state == "charging" {
        return "󰂄" // Charging icon
    }
    
    switch {
    case level >= 90:
        return "󰁹" // Full
    case level >= 80:
        return "󰂂"
    case level >= 70:
        return "󰂁"
    case level >= 60:
        return "󰂀"
    case level >= 50:
        return "󰁿"
    case level >= 40:
        return "󰁾"
    case level >= 30:
        return "󰁽"
    case level >= 20:
        return "󰁼"
    case level >= 10:
        return "󰁻"
    default:
        return "󰁺" // Critical
    }
}

func getNetworkIcon(state string) string {
    if state == "connected" {
        return "󰖩" // WiFi connected
    }
    return "󰖪" // WiFi disconnected
}
```

---

## Styling with Lipgloss

### Create `styles.go`

```go
package main

import (
    "github.com/charmbracelet/lipgloss"
)

var (
    // Colors from your theme
    primary   = lipgloss.Color("#D7BAFF")
    surface   = lipgloss.Color("#16121B")
    text      = lipgloss.Color("#E9DFEE")
    textDim   = lipgloss.Color("#E9DFEE")
    purple    = lipgloss.Color("#D9BDE3")
    pink      = lipgloss.Color("#EAB6E5")
    green     = lipgloss.Color("#B5CCBA")
    yellow    = lipgloss.Color("#f9e2af")
    red       = lipgloss.Color("#FFB4AB")
    
    // Base box style - matches your waybar aesthetic
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
    
    // Active/highlighted box
    activeBoxStyle = boxStyle.Copy().
        BorderForeground(primary).
        Foreground(primary).
        Bold(true)
    
    // Workspace styles
    workspaceStyle = boxStyle.Copy().
        Foreground(textDim).
        Padding(0, 1)
    
    workspaceActiveStyle = workspaceStyle.Copy().
        Background(lipgloss.Color("#D7BAFF")).
        Foreground(surface).
        Bold(true)
    
    // Module-specific styles
    cpuStyle = boxStyle.Copy().
        Foreground(purple).
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
```

---

## View Rendering

### Create `view.go`

```go
package main

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
    if m.width == 0 {
        return "Initializing..."
    }
    
    // Left section - Workspaces
    workspaces := renderWorkspaces(m.activeWorkspace)
    
    // Center section - Clock
    clock := renderClock(m.currentTime)
    
    // Right section - System info
    sysInfo := renderSystemInfo(m)
    
    // Calculate spacing
    leftWidth := lipgloss.Width(workspaces)
    centerWidth := lipgloss.Width(clock)
    rightWidth := lipgloss.Width(sysInfo)
    
    totalContentWidth := leftWidth + centerWidth + rightWidth
    availableSpace := m.width - totalContentWidth
    
    leftPadding := availableSpace / 3
    rightPadding := availableSpace - leftPadding
    
    // Build the statusbar
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
    timeStr := t.Format("15:04:05 | Mon 02 Jan")
    return clockStyle.Render(timeStr)
}

func renderSystemInfo(m model) string {
    modules := []string{}
    
    // CPU
    cpu := fmt.Sprintf("󰻠 %.1f%%", m.cpuUsage)
    modules = append(modules, cpuStyle.Render(cpu))
    
    // Memory
    memory := fmt.Sprintf("󰍛 %.1f%%", m.memUsage)
    modules = append(modules, memoryStyle.Render(memory))
    
    // Disk
    disk := fmt.Sprintf("󰋊 %.1f%%", m.diskUsage)
    modules = append(modules, diskStyle.Render(disk))
    
    // Network
    netIcon := getNetworkIcon(m.networkState)
    network := fmt.Sprintf("%s %s", netIcon, m.networkName)
    modules = append(modules, networkStyle.Render(network))
    
    // Battery
    batIcon := getBatteryIcon(m.batteryLevel, m.batteryState)
    battery := fmt.Sprintf("%s %d%%", batIcon, m.batteryLevel)
    
    var batStyle lipgloss.Style
    if m.batteryState == "charging" {
        batStyle = batteryChargingStyle
    } else if m.batteryLevel < 20 {
        batStyle = batteryLowStyle
    } else {
        batStyle = batteryStyle
    }
    
    modules = append(modules, batStyle.Render(battery))
    
    return lipgloss.JoinHorizontal(lipgloss.Top, modules...)
}
```

---

## Integration with Wayland/Hyprland

### Reading Hyprland State

Create `hyprland.go`:

```go
package main

import (
    "encoding/json"
    "os/exec"
    "strconv"
    "strings"
)

type HyprlandWorkspace struct {
    ID     int    `json:"id"`
    Name   string `json:"name"`
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

// Listen to Hyprland events (advanced)
func listenHyprlandEvents() tea.Cmd {
    return func() tea.Msg {
        // Connect to Hyprland socket and listen for workspace changes
        // This is more complex and requires socket programming
        return nil
    }
}
```

### Running as a Bar Replacement

For actual statusbar replacement, you have two options:

#### Option 1: Use with existing bar (easiest)
Run your TUI statusbar in a terminal window and pin it to the top with Hyprland rules:

```conf
# hyprland.conf
windowrulev2 = float, class:^(tui-statusbar)$
windowrulev2 = pin, class:^(tui-statusbar)$
windowrulev2 = size 100% 30, class:^(tui-statusbar)$
windowrulev2 = move 0 0, class:^(tui-statusbar)$
windowrulev2 = noblur, class:^(tui-statusbar)$
windowrulev2 = noshadow, class:^(tui-statusbar)$
```

Launch script:
```bash
#!/bin/bash
kitty --class tui-statusbar -e ~/go/bin/tui-statusbar
```

#### Option 2: Layer Shell Integration (advanced)
Use GTK with layer-shell to create a proper Wayland bar:

```go
// This requires CGO and GTK bindings
import (
    "github.com/gotk3/gotk3/gtk"
    "github.com/dlasky/gotk3-layershell/layershell"
)

func createLayerShellWindow() {
    gtk.Init(nil)
    
    win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
    layershell.InitForWindow(win)
    layershell.SetLayer(win, layershell.LAYER_TOP)
    layershell.SetAnchor(win, layershell.EDGE_TOP, true)
    layershell.SetAnchor(win, layershell.EDGE_LEFT, true)
    layershell.SetAnchor(win, layershell.EDGE_RIGHT, true)
    
    // Render your bubbletea output in the GTK window
}
```

---

## Advanced Features

### 1. Click Handlers

For terminal-based version, you can use mouse events:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.MouseMsg:
        if msg.Type == tea.MouseLeft {
            // Detect which module was clicked based on coordinates
            // Launch corresponding app
        }
    }
    return m, nil
}
```

### 2. Configuration File

Create `config.go`:

```go
package main

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    RefreshInterval int      `json:"refresh_interval"`
    Modules         []string `json:"modules"`
    Colors          Colors   `json:"colors"`
}

type Colors struct {
    Primary string `json:"primary"`
    Surface string `json:"surface"`
    Text    string `json:"text"`
}

func loadConfig() (*Config, error) {
    configPath := filepath.Join(os.Getenv("HOME"), ".config", "tui-statusbar", "config.json")
    
    file, err := os.Open(configPath)
    if err != nil {
        return defaultConfig(), nil
    }
    defer file.Close()
    
    var config Config
    if err := json.NewDecoder(file).Decode(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}

func defaultConfig() *Config {
    return &Config{
        RefreshInterval: 1,
        Modules:         []string{"workspaces", "clock", "cpu", "memory", "battery"},
        Colors: Colors{
            Primary: "#D7BAFF",
            Surface: "#16121B",
            Text:    "#E9DFEE",
        },
    }
}
```

Example config file (`~/.config/tui-statusbar/config.json`):

```json
{
  "refresh_interval": 1,
  "modules": [
    "workspaces",
    "window",
    "clock",
    "cpu",
    "memory",
    "disk",
    "network",
    "battery"
  ],
  "colors": {
    "primary": "#D7BAFF",
    "surface": "#16121B",
    "text": "#E9DFEE"
  }
}
```

### 3. Module System

Create pluggable modules:

```go
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
```

---

## Running as a Service

### Systemd User Service

Create `~/.config/systemd/user/tui-statusbar.service`:

```ini
[Unit]
Description=TUI Statusbar
After=graphical-session.target

[Service]
Type=simple
ExecStart=%h/go/bin/tui-statusbar
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

Enable and start:

```bash
systemctl --user enable tui-statusbar
systemctl --user start tui-statusbar
```

### Auto-start with Hyprland

Add to `~/.config/hypr/hyprland.conf`:

```conf
exec-once = kitty --class tui-statusbar -e ~/go/bin/tui-statusbar
```

---

## Building and Installing

### Build Script

Create `build.sh`:

```bash
#!/bin/bash

# Build
go build -o tui-statusbar

# Install
mkdir -p ~/go/bin
cp tui-statusbar ~/go/bin/

# Make executable
chmod +x ~/go/bin/tui-statusbar

echo "Built and installed to ~/go/bin/tui-statusbar"
```

### Makefile

Create `Makefile`:

```makefile
.PHONY: build install run clean

build:
	go build -o tui-statusbar

install: build
	mkdir -p ~/go/bin
	cp tui-statusbar ~/go/bin/
	chmod +x ~/go/bin/tui-statusbar

run: build
	./tui-statusbar

clean:
	rm -f tui-statusbar

dev:
	go run .
```

Usage:
```bash
make build    # Build the binary
make install  # Build and install
make run      # Build and run
make dev      # Run without building binary
```

---

## Testing

### Unit Tests

Create `sysinfo_test.go`:

```go
package main

import "testing"

func TestFetchSystemStats(t *testing.T) {
    cpu, mem, disk := fetchSystemStats()
    
    if cpu < 0 || cpu > 100 {
        t.Errorf("CPU usage out of range: %f", cpu)
    }
    
    if mem < 0 || mem > 100 {
        t.Errorf("Memory usage out of range: %f", mem)
    }
    
    if disk < 0 || disk > 100 {
        t.Errorf("Disk usage out of range: %f", disk)
    }
}
```

Run tests:
```bash
go test -v
```

---

## Troubleshooting

### Common Issues

1. **Icons not showing**
   - Install a Nerd Font: `sudo pacman -S ttf-jetbrains-mono-nerd`
   - Set terminal font to JetBrainsMono Nerd Font

2. **High CPU usage**
   - Increase refresh interval in config
   - Optimize system info gathering (cache values)

3. **Colors not showing**
   - Ensure terminal supports 24-bit color
   - Set `COLORTERM=truecolor` environment variable

4. **Hyprland info not updating**
   - Check hyprctl is in PATH
   - Verify Hyprland socket permissions

---

## Complete Example

Here's a minimal complete example you can start with:

```go
package main

import (
    "fmt"
    "os"
    "time"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    time string
}

func (m model) Init() tea.Cmd {
    return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    case tickMsg:
        m.time = time.Now().Format("15:04:05")
        return m, tick()
    }
    return m, nil
}

func (m model) View() string {
    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("#D7BAFF")).
        Padding(0, 1).
        Foreground(lipgloss.Color("#E9DFEE"))
    
    return boxStyle.Render(m.time)
}

type tickMsg time.Time

func tick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func main() {
    p := tea.NewProgram(model{time: time.Now().Format("15:04:05")})
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
```

Save this as `main.go`, run `go mod tidy`, then `go run .` to see it in action!

---

## Next Steps

1. Start with the minimal example above
2. Add system information gradually
3. Implement styling to match your theme
4. Add Hyprland integration
5. Create configuration system
6. Build a module system for extensibility

## Resources

- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
- [Bubbletea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples)
- [Gopsutil Documentation](https://github.com/shirou/gopsutil)
- [Nerd Fonts Icons](https://www.nerdfonts.com/cheat-sheet)

Happy coding! 🚀