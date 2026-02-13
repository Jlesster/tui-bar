package main

import (
	"strconv"
	"sync"
)

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

func (h *HyprlandEventHandler) On(eventType string, callback EventCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callbacks[eventType] = append(h.callbacks[eventType], callback)
}

func (h *HyprlandEventHandler) Start() error {
	if err := h.client.StartEventListener(); err != nil {
		return err
	}
	h.events = h.client.Subscribe()
	go h.handleEvents()
	return nil
}

func (h *HyprlandEventHandler) Stop() {
	close(h.stopChan)
	h.client.Unsubscribe(h.events)
}

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

func (h *HyprlandEventHandler) processEvent(event HyprlandEvent) {
	h.mu.RLock()
	callbacks := h.callbacks[event.Type]
	h.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(event)
	}
}

// typed event handlers
type WorkspaceCallback func(workspaceID int, workspaceName string)
type WindowCallback func(windowClass string, windowTitle string)
type MonitorCallback func(monitorName string, workspaceName string)
type WindowOpenCallback func(address string, workspace string, class string, title string)
type WindowCloseCallback func(address string)

func (h *HyprlandEventHandler) OnWorkspaceChange(callback WorkspaceCallback) {
	h.On("workspace", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			if id, err := strconv.Atoi(event.Data[0]); err == nil {
				callback(id, event.Data[0])
			} else {
				callback(0, event.Data[0])
			}
		}
	})
}

func (h *HyprlandEventHandler) OnActiveWindow(callback WindowCallback) {
	h.On("activewindow", func(event HyprlandEvent) {
		if len(event.Data) >= 2 {
			callback(event.Data[0], event.Data[1])
		}
	})
}

func (h *HyprlandEventHandler) OnMonitorFocus(callback MonitorCallback) {
	h.On("focusedmon", func(event HyprlandEvent) {
		if len(event.Data) >= 2 {
			callback(event.Data[0], event.Data[1])
		}
	})
}

func (h *HyprlandEventHandler) OnWindowOpen(callback WindowOpenCallback) {
	h.On("openwindow", func(event HyprlandEvent) {
		if len(event.Data) >= 4 {
			callback(event.Data[0], event.Data[1], event.Data[2], event.Data[3])
		}
	})
}

func (h *HyprlandEventHandler) OnWindowClose(callback WindowCloseCallback) {
	h.On("closewindow", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0])
		}
	})
}

func (h *HyprlandEventHandler) OnFullscreenToggle(callback func(fullscreen bool)) {
	h.On("fullscreen", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0] == "1")
		}
	})
}

func (h *HyprlandEventHandler) OnWorkspaceCreate(callback func(workspaceName string)) {
	h.On("createworkspace", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0])
		}
	})
}

func (h *HyprlandEventHandler) OnWorkspaceDestroy(callback func(workdspaceName string)) {
	h.On("destroyworkspace", func(event HyprlandEvent) {
		if len(event.Data) > 0 {
			callback(event.Data[0])
		}
	})
}
