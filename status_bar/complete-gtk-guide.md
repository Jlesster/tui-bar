# GTK Layer-Shell Statusbar - Complete Implementation Guide

This section extends the Bubbletea statusbar guide with a proper GTK layer-shell implementation for Wayland compositors like Hyprland.

## Table of Contents

1. [Why GTK Layer-Shell?](#why-gtk-layer-shell)
2. [Prerequisites and Dependencies](#prerequisites-and-dependencies)
3. [Project Structure for GTK](#project-structure-for-gtk)
4. [Complete GTK Implementation](#complete-gtk-implementation)
5. [CSS Styling](#css-styling)
6. [Hyprland Event Listener](#hyprland-event-listener)
7. [Building with CGO](#building-with-cgo)
8. [Running and Installation](#running-and-installation)
9. [Troubleshooting GTK Issues](#troubleshooting-gtk-issues)

---

## Why GTK Layer-Shell?

The GTK layer-shell approach offers several advantages over the terminal-based solution:

- **Native Wayland Integration**: Acts as a true Wayland bar using the layer-shell protocol
- **No Terminal Required**: Runs as a standalone application
- **Better Performance**: More efficient than rendering in a terminal
- **Compositor Integration**: Can reserve space, control layering, and interact with the compositor
- **Click Events**: Native support for mouse interactions
- **Professional Appearance**: Matches the look of native applications like Waybar

---

## Prerequisites and Dependencies

### System Requirements

```bash
# On Arch Linux
sudo pacman -S gtk3 gtk-layer-shell pkg-config gcc

# On Ubuntu/Debian
sudo apt install libgtk-3-dev gtk-layer-shell libgtk-layer-shell-dev pkg-config gcc

# On Fedora
sudo dnf install gtk3-devel gtk-layer-shell-devel pkg-config gcc
```

### Go Dependencies

```bash
# GTK3 bindings
go get github.com/gotk3/gotk3/gtk
go get github.com/gotk3/gotk3/gdk
go get github.com/gotk3/gotk3/glib
go get github.com/gotk3/gotk3/pango

# Layer-shell support
go get github.com/dlasky/gotk3-layershell/layershell

# System info (same as before)
go get github.com/shirou/gopsutil/v3/cpu
go get github.com/shirou/gopsutil/v3/mem
go get github.com/shirou/gopsutil/v3/disk
go get github.com/distatus/battery
```

### Verify GTK Installation

```bash
pkg-config --modversion gtk+-3.0
# Should output: 3.24.x or similar

pkg-config --libs gtk-layer-shell-0
# Should output library flags
```

---

## Project Structure for GTK

Update your project structure to include GTK-specific files:

```
tui-statusbar/
├── main.go           # Entry point (GTK version)
├── gtk.go            # GTK window and UI setup
├── gtk_handlers.go   # Event handlers and callbacks
├── gtk_updates.go    # Update loop for GTK
├── hypr.go           # Hyprland integration
├── hypr_events.go    # Hyprland event listener
├── model.go          # Shared state model
├── sysinfo.go        # System information
├── config.go         # Configuration
├── styles.css        # GTK CSS styling
└── batIcons.go       # Icon helpers
```

---

## Complete GTK Implementation

### Step 1: Update `main.go` for GTK

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gotk3/gotk3/gtk"
)

func main() {
	// Parse command line flags
	useGTK := flag.Bool("gtk", true, "Use GTK layer-shell mode")
	flag.Parse()

	if *useGTK {
		// Run GTK version
		gtk.Init(nil)
		
		sb, err := NewGTKStatusbar()
		if err != nil {
			log.Fatalf("Failed to create GTK statusbar: %v", err)
		}

		sb.window.ShowAll()
		gtk.Main()
	} else {
		// Fall back to bubbletea version
		runBubbletea()
	}
}

func runBubbletea() {
	// Original bubbletea implementation
	fmt.Println("Running Bubbletea version...")
	// ... (existing code)
}
```

### Step 2: Create `gtk.go` - Main GTK Window

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/dlasky/gotk3-layershell/layershell"
)

type GTKStatusbar struct {
	window      *gtk.Window
	mainBox     *gtk.Box
	leftBox     *gtk.Box
	centerBox   *gtk.Box
	rightBox    *gtk.Box
	
	// Workspace widgets
	workspaceBox     *gtk.Box
	workspaceButtons []*gtk.Button
	
	// Module labels
	clockLabel   *gtk.Label
	cpuLabel     *gtk.Label
	memLabel     *gtk.Label
	diskLabel    *gtk.Label
	networkLabel *gtk.Label
	batteryLabel *gtk.Label
	windowLabel  *gtk.Label
	
	// State
	model  *model
	config *Config
	
	// Update ticker
	ticker *time.Ticker
}

func NewGTKStatusbar() (*GTKStatusbar, error) {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		config = defaultConfig()
	}

	// Create statusbar instance
	sb := &GTKStatusbar{
		model:            initModel(),
		config:           config,
		workspaceButtons: make([]*gtk.Button, 10),
		ticker:           time.NewTicker(time.Second),
	}

	// Initialize window
	if err := sb.createWindow(); err != nil {
		return nil, err
	}

	// Setup layer shell
	if err := sb.setupLayerShell(); err != nil {
		return nil, err
	}

	// Create UI
	if err := sb.createUI(); err != nil {
		return nil, err
	}

	// Apply CSS
	if err := sb.applyCSS(); err != nil {
		log.Printf("Warning: Failed to apply CSS: %v", err)
	}

	// Start update loops
	sb.startUpdateLoop()
	sb.startHyprlandListener()

	return sb, nil
}

func (sb *GTKStatusbar) createWindow() error {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	win.SetTitle("GTK Statusbar")
	win.SetDecorated(false)
	win.SetResizable(false)
	win.SetAppPaintable(true)
	
	// Connect destroy signal
	win.Connect("destroy", func() {
		sb.ticker.Stop()
		gtk.MainQuit()
	})

	sb.window = win
	return nil
}

func (sb *GTKStatusbar) setupLayerShell() error {
	// Initialize layer shell for this window
	layershell.InitForWindow(sb.window)
	
	// Set as top layer
	layershell.SetLayer(sb.window, layershell.LAYER_TOP)
	
	// Anchor to top edge and full width
	layershell.SetAnchor(sb.window, layershell.EDGE_TOP, true)
	layershell.SetAnchor(sb.window, layershell.EDGE_LEFT, true)
	layershell.SetAnchor(sb.window, layershell.EDGE_RIGHT, true)
	layershell.SetAnchor(sb.window, layershell.EDGE_BOTTOM, false)
	
	// Set namespace (useful for compositor rules)
	layershell.SetNamespace(sb.window, "statusbar")
	
	// Reserve exclusive zone (bar height)
	layershell.SetExclusiveZone(sb.window, 30)
	
	// Set margins
	layershell.SetMargin(sb.window, layershell.EDGE_TOP, 0)
	layershell.SetMargin(sb.window, layershell.EDGE_BOTTOM, 0)
	layershell.SetMargin(sb.window, layershell.EDGE_LEFT, 0)
	layershell.SetMargin(sb.window, layershell.EDGE_RIGHT, 0)
	
	return nil
}

func (sb *GTKStatusbar) createUI() error {
	// Main horizontal box
	mainBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	mainBox.SetName("main-box")
	mainBox.SetHomogeneous(false)
	sb.mainBox = mainBox

	// Left section (workspaces)
	leftBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	leftBox.SetName("left-box")
	sb.leftBox = leftBox

	// Center section (clock/window title)
	centerBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	centerBox.SetName("center-box")
	centerBox.SetHAlign(gtk.ALIGN_CENTER)
	sb.centerBox = centerBox

	// Right section (system info)
	rightBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	rightBox.SetName("right-box")
	rightBox.SetHAlign(gtk.ALIGN_END)
	sb.rightBox = rightBox

	// Pack into main box
	mainBox.PackStart(leftBox, false, false, 0)
	mainBox.PackStart(centerBox, true, true, 0)
	mainBox.PackEnd(rightBox, false, false, 0)

	// Create widgets
	if err := sb.createWorkspaceWidget(); err != nil {
		return err
	}
	if err := sb.createCenterWidgets(); err != nil {
		return err
	}
	if err := sb.createSystemWidgets(); err != nil {
		return err
	}

	sb.window.Add(mainBox)
	return nil
}

func (sb *GTKStatusbar) createWorkspaceWidget() error {
	workspaceBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	if err != nil {
		return err
	}
	workspaceBox.SetName("workspace-box")
	workspaceBox.SetMarginStart(10)

	// Create workspace buttons (1-10)
	for i := 0; i < 10; i++ {
		wsNum := i + 1
		btn, err := gtk.ButtonNewWithLabel(fmt.Sprintf("%d", wsNum))
		if err != nil {
			return err
		}
		
		btn.SetName("workspace-button")
		btn.SetRelief(gtk.RELIEF_NONE)
		btn.SetCanFocus(false)
		
		// Click handler
		btn.Connect("clicked", func() {
			sb.switchWorkspace(wsNum)
		})
		
		sb.workspaceButtons[i] = btn
		workspaceBox.PackStart(btn, false, false, 0)
	}

	sb.workspaceBox = workspaceBox
	sb.leftBox.PackStart(workspaceBox, false, false, 0)
	return nil
}

func (sb *GTKStatusbar) createCenterWidgets() error {
	// Window title (optional)
	windowLabel, err := gtk.LabelNew("")
	if err != nil {
		return err
	}
	windowLabel.SetName("window-title")
	windowLabel.SetEllipsize(pango.ELLIPSIZE_END)
	windowLabel.SetMaxWidthChars(50)
	windowLabel.SetMarginEnd(20)
	sb.windowLabel = windowLabel

	// Clock
	clockLabel, err := gtk.LabelNew("")
	if err != nil {
		return err
	}
	clockLabel.SetName("clock")
	clockLabel.SetMarginStart(20)
	sb.clockLabel = clockLabel

	sb.centerBox.PackStart(windowLabel, false, false, 0)
	sb.centerBox.PackStart(clockLabel, false, false, 0)
	return nil
}

func (sb *GTKStatusbar) createSystemWidgets() error {
	// Create container for system modules
	sysBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	sysBox.SetName("system-box")
	sysBox.SetMarginEnd(10)

	// CPU
	cpuBox, cpuLabel, err := sb.createModule("cpu", "")
	if err != nil {
		return err
	}
	sb.cpuLabel = cpuLabel
	sysBox.PackStart(cpuBox, false, false, 0)

	// Memory
	memBox, memLabel, err := sb.createModule("memory", "")
	if err != nil {
		return err
	}
	sb.memLabel = memLabel
	sysBox.PackStart(memBox, false, false, 0)

	// Disk
	diskBox, diskLabel, err := sb.createModule("disk", "")
	if err != nil {
		return err
	}
	sb.diskLabel = diskLabel
	sysBox.PackStart(diskBox, false, false, 0)

	// Network
	netBox, netLabel, err := sb.createModule("network", "")
	if err != nil {
		return err
	}
	sb.networkLabel = netLabel
	sysBox.PackStart(netBox, false, false, 0)

	// Battery
	batBox, batLabel, err := sb.createModule("battery", "")
	if err != nil {
		return err
	}
	sb.batteryLabel = batLabel
	sysBox.PackStart(batBox, false, false, 0)

	sb.rightBox.PackStart(sysBox, false, false, 0)
	return nil
}

func (sb *GTKStatusbar) createModule(name, initialText string) (*gtk.Box, *gtk.Label, error) {
	box, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	if err != nil {
		return nil, nil, err
	}
	box.SetName(fmt.Sprintf("module-%s", name))
	box.SetMarginStart(5)
	box.SetMarginEnd(5)

	label, err := gtk.LabelNew(initialText)
	if err != nil {
		return nil, nil, err
	}
	label.SetName(fmt.Sprintf("%s-label", name))
	label.SetUseMarkup(true)

	box.PackStart(label, false, false, 0)
	return box, label, nil
}
```

### Step 3: Create `gtk_updates.go` - Update Loop

```go
package main

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/glib"
)

func (sb *GTKStatusbar) startUpdateLoop() {
	// Initial update
	sb.updateAll()

	// Start ticker for periodic updates
	go func() {
		for range sb.ticker.C {
			// Schedule UI update in GTK main thread
			glib.IdleAdd(func() {
				sb.updateAll()
			})
		}
	}()
}

func (sb *GTKStatusbar) updateAll() {
	sb.updateClock()
	sb.updateSystemInfo()
	sb.updateBattery()
	sb.updateNetwork()
	sb.updateWorkspaces()
	sb.updateWindowTitle()
}

func (sb *GTKStatusbar) updateClock() {
	now := time.Now()
	timeStr := now.Format("15:04:05 | Mon 02 Jan")
	sb.clockLabel.SetMarkup(fmt.Sprintf("<b>%s</b>", timeStr))
}

func (sb *GTKStatusbar) updateSystemInfo() {
	cpu, mem, disk := fetchSystemStats()
	
	sb.model.cpuUsage = cpu
	sb.model.memUsage = mem
	sb.model.diskUsage = disk

	// Update labels with markup
	sb.cpuLabel.SetMarkup(fmt.Sprintf("  <b>%.1f%%</b>", cpu))
	sb.memLabel.SetMarkup(fmt.Sprintf("  <b>%.1f%%</b>", mem))
	sb.diskLabel.SetMarkup(fmt.Sprintf("  <b>%.1f%%</b>", disk))
}

func (sb *GTKStatusbar) updateBattery() {
	level, state := fetchBatteryStats()
	
	sb.model.batLevel = level
	sb.model.batState = state

	icon := getBatteryIcon(level, state)
	sb.batteryLabel.SetMarkup(fmt.Sprintf("%s <b>%d%%</b>", icon, level))
	
	// Update CSS class based on state
	ctx, _ := sb.batteryLabel.GetStyleContext()
	ctx.RemoveClass("charging")
	ctx.RemoveClass("low")
	
	if state == "charging" {
		ctx.AddClass("charging")
	} else if level < 20 {
		ctx.AddClass("low")
	}
}

func (sb *GTKStatusbar) updateNetwork() {
	name, state := fetchNetworkInfo()
	
	sb.model.netName = name
	sb.model.netState = state

	icon := getNetworkIcon(state)
	sb.networkLabel.SetMarkup(fmt.Sprintf("%s <b>%s</b>", icon, name))
}

func (sb *GTKStatusbar) updateWorkspaces() {
	activeWS := getActiveWorkspace()
	sb.model.activeWorkspace = activeWS

	// Update button styles
	for i, btn := range sb.workspaceButtons {
		ctx, _ := btn.GetStyleContext()
		
		if i+1 == activeWS {
			ctx.AddClass("active")
		} else {
			ctx.RemoveClass("active")
		}
	}
}

func (sb *GTKStatusbar) updateWindowTitle() {
	title := getActiveWindow()
	sb.model.windowTitle = title
	
	if title != "" {
		sb.windowLabel.SetText(title)
		sb.windowLabel.Show()
	} else {
		sb.windowLabel.Hide()
	}
}
```

### Step 4: Create `gtk_handlers.go` - Event Handlers

```go
package main

import (
	"log"
	"os/exec"
)

func (sb *GTKStatusbar) switchWorkspace(num int) {
	cmd := exec.Command("hyprctl", "dispatch", "workspace", fmt.Sprintf("%d", num))
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to switch workspace: %v", err)
	}
}

// Add click handlers for modules
func (sb *GTKStatusbar) setupModuleHandlers() {
	// CPU click - open system monitor
	if cpuBox := sb.cpuLabel.GetParent(); cpuBox != nil {
		eventBox, _ := gtk.EventBoxNew()
		eventBox.Add(cpuBox)
		eventBox.Connect("button-press-event", func() {
			exec.Command("kitty", "-e", "htop").Start()
		})
	}

	// Battery click - open power settings
	if batBox := sb.batteryLabel.GetParent(); batBox != nil {
		eventBox, _ := gtk.EventBoxNew()
		eventBox.Add(batBox)
		eventBox.Connect("button-press-event", func() {
			exec.Command("xdg-open", "gnome-control-center", "power").Start()
		})
	}

	// Network click - open network settings
	if netBox := sb.networkLabel.GetParent(); netBox != nil {
		eventBox, _ := gtk.EventBoxNew()
		eventBox.Add(netBox)
		eventBox.Connect("button-press-event", func() {
			exec.Command("kitty", "-e", "nmtui").Start()
		})
	}
}
```

### Step 5: Create `hypr_events.go` - Real-time Hyprland Events

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/gotk3/gotk3/glib"
)

func (sb *GTKStatusbar) startHyprlandListener() {
	go sb.listenHyprlandEvents()
}

func (sb *GTKStatusbar) listenHyprlandEvents() {
	// Get Hyprland instance signature
	signature := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if signature == "" {
		log.Println("Not running under Hyprland, skipping event listener")
		return
	}

	// Connect to Hyprland socket
	socketPath := fmt.Sprintf("/tmp/hypr/%s/.socket2.sock", signature)
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Printf("Failed to connect to Hyprland socket: %v", err)
		return
	}
	defer conn.Close()

	log.Println("Connected to Hyprland event socket")

	// Read events
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		sb.handleHyprlandEvent(line)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from Hyprland socket: %v", err)
	}
}

func (sb *GTKStatusbar) handleHyprlandEvent(event string) {
	parts := strings.SplitN(event, ">>", 2)
	if len(parts) != 2 {
		return
	}

	eventType := parts[0]
	// eventData := parts[1]

	switch eventType {
	case "workspace":
		// Workspace changed
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})

	case "focusedmon":
		// Monitor focus changed
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})

	case "activewindow":
		// Active window changed
		glib.IdleAdd(func() {
			sb.updateWindowTitle()
		})

	case "createworkspace", "destroyworkspace":
		// Workspace created/destroyed
		glib.IdleAdd(func() {
			sb.updateWorkspaces()
		})
	}
}
```

---

## CSS Styling

### Create `styles.css`

```css
/* Main container */
#main-box {
    background-color: #16121B;
    color: #E9DFEE;
    font-family: "JetBrainsMono Nerd Font";
    font-size: 12px;
    padding: 0px 10px;
    min-height: 30px;
}

/* Workspace buttons */
#workspace-box button {
    background-color: transparent;
    border: 1px solid #D7BAFF;
    border-radius: 4px;
    color: #E9DFEE;
    padding: 2px 10px;
    margin: 0px 2px;
    min-width: 25px;
    transition: all 200ms ease;
}

#workspace-box button:hover {
    background-color: rgba(215, 186, 255, 0.2);
}

#workspace-box button.active {
    background-color: #D7BAFF;
    color: #16121B;
    font-weight: bold;
}

/* Clock */
#clock {
    color: #D7BAFF;
    font-weight: bold;
    padding: 0px 15px;
}

/* Window title */
#window-title {
    color: #E9DFEE;
    font-style: italic;
}

/* System modules */
#system-box {
    spacing: 0px;
}

.module-cpu,
.module-memory,
.module-disk,
.module-network,
.module-battery {
    border: 1px solid #D7BAFF;
    border-radius: 4px;
    padding: 2px 10px;
    margin: 0px 3px;
}

/* CPU module */
#module-cpu {
    border-color: #D9BDE3;
}

#cpu-label {
    color: #D9BDE3;
}

/* Memory module */
#module-memory {
    border-color: #EAB6E5;
}

#memory-label {
    color: #EAB6E5;
}

/* Disk module */
#module-disk {
    border-color: #E9DFEE;
}

#disk-label {
    color: #E9DFEE;
}

/* Network module */
#module-network {
    border-color: #D9BDE3;
}

#network-label {
    color: #D9BDE3;
}

/* Battery module */
#module-battery {
    border-color: #E9DFEE;
}

#battery-label {
    color: #E9DFEE;
}

#battery-label.charging {
    color: #B5CCBA;
    border-color: #B5CCBA;
}

#battery-label.low {
    color: #FFB4AB;
    border-color: #FFB4AB;
}

/* Hover effects for clickable modules */
#module-cpu:hover,
#module-battery:hover,
#module-network:hover {
    background-color: rgba(215, 186, 255, 0.1);
    cursor: pointer;
}
```

### Apply CSS in `gtk.go`

Add this method to the GTKStatusbar struct:

```go
func (sb *GTKStatusbar) applyCSS() error {
	cssProvider, err := gtk.CssProviderNew()
	if err != nil {
		return err
	}

	// Try to load from file first
	cssPath := filepath.Join(os.Getenv("HOME"), ".config", "tui-statusbar", "styles.css")
	if _, err := os.Stat(cssPath); err == nil {
		if err := cssProvider.LoadFromPath(cssPath); err != nil {
			return err
		}
	} else {
		// Load default CSS
		defaultCSS := getDefaultCSS()
		if err := cssProvider.LoadFromData(defaultCSS); err != nil {
			return err
		}
	}

	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		return err
	}

	gtk.AddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	return nil
}

func getDefaultCSS() string {
	return `
	/* Embed the CSS from styles.css here as a string */
	#main-box {
		background-color: #16121B;
		color: #E9DFEE;
		font-family: "JetBrainsMono Nerd Font";
		font-size: 12px;
		padding: 0px 10px;
		min-height: 30px;
	}
	/* ... rest of CSS ... */
	`
}
```

---

## Building with CGO

### Environment Setup

CGO is required for GTK bindings. Set up your build environment:

```bash
# Export PKG_CONFIG_PATH if needed
export PKG_CONFIG_PATH=/usr/lib/pkgconfig

# Verify pkg-config can find GTK
pkg-config --cflags --libs gtk+-3.0
```

### Build Script

Create `build-gtk.sh`:

```bash
#!/bin/bash

set -e

echo "Building GTK statusbar..."

# Enable CGO
export CGO_ENABLED=1

# Build
go build -tags gtk -o statusbar-gtk .

echo "Build complete: ./statusbar-gtk"
```

Make it executable:

```bash
chmod +x build-gtk.sh
./build-gtk.sh
```

### Makefile Updates

Add GTK targets to your Makefile:

```makefile
.PHONY: build-gtk install-gtk run-gtk

build-gtk:
	CGO_ENABLED=1 go build -tags gtk -o statusbar-gtk .

install-gtk: build-gtk
	mkdir -p $(HOME)/.local/bin
	cp statusbar-gtk $(HOME)/.local/bin/
	chmod +x $(HOME)/.local/bin/statusbar-gtk
	mkdir -p $(HOME)/.config/tui-statusbar
	cp styles.css $(HOME)/.config/tui-statusbar/

run-gtk: build-gtk
	./statusbar-gtk

clean-gtk:
	rm -f statusbar-gtk
```

---

## Running and Installation

### Manual Running

```bash
# Build and run
./build-gtk.sh
./statusbar-gtk
```

### Systemd Service

Create `~/.config/systemd/user/statusbar-gtk.service`:

```ini
[Unit]
Description=GTK Statusbar
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/statusbar-gtk
Restart=on-failure
RestartSec=3

[Install]
WantedBy=graphical-session.target
```

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable statusbar-gtk
systemctl --user start statusbar-gtk
```

### Hyprland Integration

Add to `~/.config/hypr/hyprland.conf`:

```conf
# Start statusbar
exec-once = systemctl --user start statusbar-gtk

# Optional: If not using systemd
# exec-once = ~/.local/bin/statusbar-gtk
```

### Complete Installation Script

Create `install.sh`:

```bash
#!/bin/bash

set -e

echo "Installing GTK Statusbar..."

# Build
make build-gtk

# Create directories
mkdir -p ~/.local/bin
mkdir -p ~/.config/tui-statusbar
mkdir -p ~/.config/systemd/user

# Copy files
cp statusbar-gtk ~/.local/bin/
cp styles.css ~/.config/tui-statusbar/
cp config.json.example ~/.config/tui-statusbar/config.json 2>/dev/null || true

# Create systemd service
cat > ~/.config/systemd/user/statusbar-gtk.service <<EOF
[Unit]
Description=GTK Statusbar
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/statusbar-gtk
Restart=on-failure
RestartSec=3

[Install]
WantedBy=graphical-session.target
EOF

# Reload systemd
systemctl --user daemon-reload

echo "Installation complete!"
echo ""
echo "To start the statusbar:"
echo "  systemctl --user start statusbar-gtk"
echo ""
echo "To enable on login:"
echo "  systemctl --user enable statusbar-gtk"
EOF

chmod +x install.sh
```

---

## Troubleshooting GTK Issues

### Common Build Errors

#### 1. **GTK headers not found**

```
Error: Package gtk+-3.0 was not found in the pkg-config search path
```

**Solution:**
```bash
# Arch
sudo pacman -S gtk3

# Ubuntu/Debian
sudo apt install libgtk-3-dev

# Fedora
sudo dnf install gtk3-devel
```

#### 2. **Layer-shell not found**

```
Error: Package gtk-layer-shell-0 was not found
```

**Solution:**
```bash
# Arch
sudo pacman -S gtk-layer-shell

# Ubuntu (may need to compile from source)
git clone https://github.com/wmww/gtk-layer-shell.git
cd gtk-layer-shell
meson build
ninja -C build
sudo ninja -C build install
```

#### 3. **CGO errors**

```
Error: C compiler not found
```

**Solution:**
```bash
# Install build tools
sudo pacman -S base-devel  # Arch
sudo apt install build-essential  # Ubuntu
```

### Runtime Issues

#### 1. **Bar not showing**

Check if layer-shell is working:
```bash
# Check if running under Wayland
echo $WAYLAND_DISPLAY

# Verify compositor supports layer-shell
# For Hyprland, this should work automatically
```

#### 2. **Icons not displaying**

Install a Nerd Font and set it in CSS:
```bash
sudo pacman -S ttf-jetbrains-mono-nerd
```

Update `styles.css`:
```css
#main-box {
    font-family: "JetBrainsMono Nerd Font";
}
```

#### 3. **High CPU usage**

Increase update interval in config:
```json
{
  "refresh_interval": 2
}
```

Or optimize the update loop in `gtk_updates.go`:
```go
func (sb *GTKStatusbar) startUpdateLoop() {
    // Update different modules at different rates
    go func() {
        clockTicker := time.NewTicker(time.Second)
        sysTicker := time.NewTicker(3 * time.Second)
        
        for {
            select {
            case <-clockTicker.C:
                glib.IdleAdd(sb.updateClock)
            case <-sysTicker.C:
                glib.IdleAdd(sb.updateSystemInfo)
                glib.IdleAdd(sb.updateBattery)
            }
        }
    }()
}
```

#### 4. **Workspace switching not working**

Verify `hyprctl` is in PATH:
```bash
which hyprctl
```

Test manually:
```bash
hyprctl dispatch workspace 2
```

#### 5. **Events not updating**

Check Hyprland socket:
```bash
echo $HYPRLAND_INSTANCE_SIGNATURE
ls -la /tmp/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock
```

Enable debug logging in `hypr_events.go`:
```go
func (sb *GTKStatusbar) handleHyprlandEvent(event string) {
    log.Printf("Received event: %s", event)  // Add this
    // ... rest of function
}
```

---

## Advanced Customization

### Custom Modules

Create a new module by adding to `gtk.go`:

```go
func (sb *GTKStatusbar) createCustomModule() error {
    // Create a custom module box
    customBox, customLabel, err := sb.createModule("custom", "")
    if err != nil {
        return err
    }
    sb.customLabel = customLabel
    
    // Add to right box
    sb.rightBox.PackStart(customBox, false, false, 0)
    
    return nil
}

// Update function
func (sb *GTKStatusbar) updateCustomModule() {
    // Your custom logic here
    value := getCustomValue()
    sb.customLabel.SetMarkup(fmt.Sprintf("📊 <b>%s</b>", value))
}
```

### Dynamic Workspaces

Support for dynamic workspace creation/destruction:

```go
func (sb *GTKStatusbar) updateWorkspaces() {
    // Get all workspaces from Hyprland
    cmd := exec.Command("hyprctl", "workspaces", "-j")
    output, err := cmd.Output()
    if err != nil {
        return
    }
    
    var workspaces []HyprlandWorkspace
    if err := json.Unmarshal(output, &workspaces); err != nil {
        return
    }
    
    activeWS := getActiveWorkspace()
    
    // Hide all buttons first
    for _, btn := range sb.workspaceButtons {
        btn.Hide()
    }
    
    // Show only existing workspaces
    for _, ws := range workspaces {
        if ws.ID > 0 && ws.ID <= len(sb.workspaceButtons) {
            btn := sb.workspaceButtons[ws.ID-1]
            btn.Show()
            
            ctx, _ := btn.GetStyleContext()
            if ws.ID == activeWS {
                ctx.AddClass("active")
            } else {
                ctx.RemoveClass("active")
            }
        }
    }
}
```

### Multi-Monitor Support

Handle multiple monitors:

```go
func (sb *GTKStatusbar) setupLayerShell() error {
    layershell.InitForWindow(sb.window)
    layershell.SetLayer(sb.window, layershell.LAYER_TOP)
    
    // Monitor selection
    display, _ := gdk.DisplayGetDefault()
    monitor := display.GetPrimaryMonitor()
    
    // Or get specific monitor
    // monitor := display.GetMonitor(0)
    
    layershell.SetMonitor(sb.window, monitor)
    
    // ... rest of setup
}
```

### Tooltips

Add tooltips to modules:

```go
func (sb *GTKStatusbar) createModule(name, initialText string) (*gtk.Box, *gtk.Label, error) {
    box, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
    if err != nil {
        return nil, nil, err
    }
    box.SetName(fmt.Sprintf("module-%s", name))
    
    // Add tooltip
    box.SetTooltipText(fmt.Sprintf("Click to open %s settings", name))
    
    label, err := gtk.LabelNew(initialText)
    if err != nil {
        return nil, nil, err
    }
    label.SetName(fmt.Sprintf("%s-label", name))
    label.SetUseMarkup(true)
    
    box.PackStart(label, false, false, 0)
    return box, label, nil
}
```

### Context Menus

Add right-click menus:

```go
func (sb *GTKStatusbar) createModuleWithMenu(name, initialText string) (*gtk.Box, *gtk.Label, error) {
    box, label, err := sb.createModule(name, initialText)
    if err != nil {
        return nil, nil, err
    }
    
    // Create event box for clicks
    eventBox, _ := gtk.EventBoxNew()
    eventBox.Add(box)
    
    // Create menu
    menu, _ := gtk.MenuNew()
    
    item1, _ := gtk.MenuItemNewWithLabel("Option 1")
    item1.Connect("activate", func() {
        log.Println("Option 1 clicked")
    })
    menu.Append(item1)
    
    item2, _ := gtk.MenuItemNewWithLabel("Option 2")
    item2.Connect("activate", func() {
        log.Println("Option 2 clicked")
    })
    menu.Append(item2)
    
    menu.ShowAll()
    
    // Right-click handler
    eventBox.Connect("button-press-event", func(_ *gtk.EventBox, event *gdk.Event) {
        btnEvent := gdk.EventButtonNewFromEvent(event)
        if btnEvent.Button() == 3 { // Right click
            menu.PopupAtPointer(event)
        }
    })
    
    return box, label, nil
}
```

### Animations

Add smooth transitions:

```go
// In styles.css
#workspace-box button {
    transition: all 300ms cubic-bezier(0.4, 0.0, 0.2, 1);
}

#workspace-box button.active {
    background-color: #D7BAFF;
    transform: scale(1.1);
}
```

For programmatic animations, use GLib timeout:

```go
func (sb *GTKStatusbar) animateModule(label *gtk.Label) {
    opacity := 0.0
    
    glib.TimeoutAdd(16, func() bool { // ~60fps
        opacity += 0.05
        if opacity >= 1.0 {
            opacity = 1.0
            return false // Stop animation
        }
        label.SetOpacity(opacity)
        return true // Continue animation
    })
}
```

---

## Performance Optimization

### Efficient Updates

Only update what changed:

```go
type GTKStatusbar struct {
    // ... existing fields
    
    // Cache previous values
    lastCPU     float64
    lastMem     float64
    lastBattery int
}

func (sb *GTKStatusbar) updateSystemInfo() {
    cpu, mem, disk := fetchSystemStats()
    
    // Only update if changed significantly
    if math.Abs(cpu - sb.lastCPU) > 0.5 {
        sb.cpuLabel.SetMarkup(fmt.Sprintf("  <b>%.1f%%</b>", cpu))
        sb.lastCPU = cpu
    }
    
    if math.Abs(mem - sb.lastMem) > 0.5 {
        sb.memLabel.SetMarkup(fmt.Sprintf("  <b>%.1f%%</b>", mem))
        sb.lastMem = mem
    }
}
```

### Lazy Module Loading

Only create modules that are enabled:

```go
func (sb *GTKStatusbar) createSystemWidgets() error {
    sysBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
    
    // Check config for enabled modules
    for _, modName := range sb.config.Modules {
        switch modName {
        case "cpu":
            cpuBox, cpuLabel, _ := sb.createModule("cpu", "")
            sb.cpuLabel = cpuLabel
            sysBox.PackStart(cpuBox, false, false, 0)
            
        case "memory":
            memBox, memLabel, _ := sb.createModule("memory", "")
            sb.memLabel = memLabel
            sysBox.PackStart(memBox, false, false, 0)
            
        // ... other modules
        }
    }
    
    sb.rightBox.PackStart(sysBox, false, false, 0)
    return nil
}
```

### Reduced Redraws

Batch updates together:

```go
func (sb *GTKStatusbar) batchUpdate() {
    // Freeze updates
    sb.window.Freeze()
    
    // Do all updates
    sb.updateClock()
    sb.updateSystemInfo()
    sb.updateBattery()
    
    // Resume updates
    sb.window.Thaw()
}
```

---

## Configuration Examples

### Minimal Config

`~/.config/tui-statusbar/config.json`:

```json
{
  "refresh_interval": 2,
  "modules": [
    "workspaces",
    "clock",
    "battery"
  ],
  "colors": {
    "primary": "#D7BAFF",
    "surface": "#16121B",
    "text": "#E9DFEE"
  }
}
```

### Full-Featured Config

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
    "battery",
    "volume",
    "brightness"
  ],
  "colors": {
    "primary": "#D7BAFF",
    "surface": "#16121B",
    "text": "#E9DFEE",
    "accent1": "#D9BDE3",
    "accent2": "#EAB6E5",
    "warning": "#FFB4AB",
    "success": "#B5CCBA"
  },
  "bar": {
    "height": 32,
    "position": "top",
    "font": "JetBrainsMono Nerd Font",
    "font_size": 12
  },
  "workspace": {
    "count": 10,
    "dynamic": true,
    "show_empty": false
  }
}
```

---

## Comparison: TUI vs GTK

| Feature | Bubbletea (TUI) | GTK Layer-Shell |
|---------|----------------|-----------------|
| **Setup Complexity** | Simple | Moderate |
| **Dependencies** | Minimal | GTK3 + Layer-shell |
| **Performance** | Good | Excellent |
| **Wayland Integration** | Via terminal | Native |
| **Click Events** | Limited | Full support |
| **Styling** | Lipgloss (inline) | CSS (external) |
| **Build Process** | Pure Go | Requires CGO |
| **Resource Usage** | Higher (terminal) | Lower |
| **Compositor Space** | Window rule hack | Proper exclusive zone |
| **Updates** | Terminal refresh | Widget updates |
| **Portability** | Any terminal | Wayland only |

### When to Use TUI Version:
- Quick prototyping
- Want pure Go (no CGO)
- Already comfortable with Bubbletea
- Need to run on non-Wayland systems

### When to Use GTK Version:
- Production statusbar
- Need proper Wayland integration
- Want native performance
- Need advanced click handling
- Professional appearance

---

## Migration from TUI to GTK

If you've been using the TUI version and want to migrate:

### 1. **Keep Shared Code**

These files work for both:
- `config.go`
- `sysinfo.go`
- `batIcons.go`
- `hypr.go` (the data fetching parts)

### 2. **Port Styling**

Convert Lipgloss styles to CSS:

**Lipgloss:**
```go
boxStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("#D7BAFF")).
    Padding(0, 1)
```

**CSS:**
```css
.module {
    border: 1px solid #D7BAFF;
    border-radius: 4px;
    padding: 2px 10px;
}
```

### 3. **Update Launch Method**

**Old (TUI):**
```bash
kitty --class tui-statusbar -e ~/go/bin/tui-statusbar
```

**New (GTK):**
```bash
~/go/bin/statusbar-gtk
```

Or via systemd service (recommended)

---

## Complete Example Project

Here's a minimal but complete GTK statusbar:

### File: `main-simple.go`

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/gotk3/gotk3/glib"
    "github.com/gotk3/gotk3/gtk"
    "github.com/dlasky/gotk3-layershell/layershell"
)

func main() {
    gtk.Init(nil)
    
    // Create window
    win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
    win.SetTitle("Simple Statusbar")
    win.SetDecorated(false)
    
    // Setup layer-shell
    layershell.InitForWindow(win)
    layershell.SetLayer(win, layershell.LAYER_TOP)
    layershell.SetAnchor(win, layershell.EDGE_TOP, true)
    layershell.SetAnchor(win, layershell.EDGE_LEFT, true)
    layershell.SetAnchor(win, layershell.EDGE_RIGHT, true)
    layershell.SetExclusiveZone(win, 30)
    
    // Create UI
    box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
    label, _ := gtk.LabelNew("")
    box.PackStart(label, true, true, 0)
    win.Add(box)
    
    // Apply simple styling
    applySimpleCSS()
    
    // Update clock every second
    go func() {
        ticker := time.NewTicker(time.Second)
        for range ticker.C {
            glib.IdleAdd(func() {
                timeStr := time.Now().Format("15:04:05")
                label.SetMarkup(fmt.Sprintf("<b>%s</b>", timeStr))
            })
        }
    }()
    
    win.Connect("destroy", gtk.MainQuit)
    win.ShowAll()
    gtk.Main()
}

func applySimpleCSS() {
    css := `
        window {
            background-color: #16121B;
        }
        box {
            padding: 5px 20px;
        }
        label {
            color: #D7BAFF;
            font-family: monospace;
            font-size: 14px;
        }
    `
    
    provider, _ := gtk.CssProviderNew()
    provider.LoadFromData(css)
    
    screen, _ := gdk.ScreenGetDefault()
    gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
```

Build and run:
```bash
CGO_ENABLED=1 go build -o simple-bar main-simple.go
./simple-bar
```

---

## Debugging Tips

### Enable Debug Logging

Add to `main.go`:

```go
import "log"

func main() {
    // Enable debug logging
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
    log.Println("Starting GTK statusbar...")
    
    // ... rest of main
}
```

### GTK Inspector

Enable GTK Inspector for live debugging:

```bash
GTK_DEBUG=interactive ./statusbar-gtk
```

This opens a GUI inspector to examine widgets, CSS, and layout.

### Monitor Events

Log all Hyprland events:

```go
func (sb *GTKStatusbar) handleHyprlandEvent(event string) {
    log.Printf("[HYPRLAND] %s", event)
    // ... rest of handler
}
```

### Check Layer-Shell

Verify layer-shell is working:

```bash
# Install sway/wlroots debugging tools
sudo pacman -S wlr-randr

# Check layers
wlr-randr
```

---

## Further Resources

### Documentation
- [GTK3 Documentation](https://docs.gtk.org/gtk3/)
- [gotk3 Examples](https://github.com/gotk3/gotk3-examples)
- [gtk-layer-shell](https://github.com/wmww/gtk-layer-shell)
- [Hyprland IPC](https://wiki.hyprland.org/IPC/)

### Similar Projects
- [Waybar](https://github.com/Alexays/Waybar) - Feature-rich Wayland bar
- [Yambar](https://codeberg.org/dnkl/yambar) - Lightweight modular bar
- [Eww](https://github.com/elkowar/eww) - Widget system

### Community
- [r/unixporn](https://reddit.com/r/unixporn) - Rice showcases
- [Hyprland Discord](https://discord.gg/hQ9XvMUjjr) - Hyprland community
- [GTK Discourse](https://discourse.gnome.org/c/platform/core/10) - GTK help

---

## Conclusion

You now have a complete, production-ready GTK layer-shell statusbar that:

✅ Integrates natively with Wayland compositors  
✅ Updates in real-time via Hyprland events  
✅ Supports custom styling with CSS  
✅ Handles click events and interactions  
✅ Runs efficiently as a system service  
✅ Is fully customizable and extensible  

The GTK approach provides a professional, performant statusbar that rivals existing solutions like Waybar while giving you complete control over functionality and appearance.

Happy customizing! 🎨