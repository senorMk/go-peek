// Package session owns the in-memory feasibility stream. It has no desktop dependencies.
package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const MaxMarkerBytes = 512

var ErrBusy = errors.New("a demo stream is already running; cancel it first")

type Event struct {
	RequestID string `json:"requestId"`
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
}

// Sink must be nonblocking and must not call Manager methods synchronously.
type Sink func(Event)

type Manager struct {
	mu       sync.Mutex
	sequence uint64
	active   string
	cancel   context.CancelFunc
	sink     Sink
}

func New(sink Sink) *Manager { return &Manager{sink: sink} }

// Start allocates a request before streaming. Every event carries its identity.
func (m *Manager) Start(parent context.Context, marker string) (string, error) {
	if strings.TrimSpace(marker) == "" {
		return "", errors.New("enter a visible test marker")
	}
	if len(marker) > MaxMarkerBytes || !utf8.ValidString(marker) {
		return "", errors.New("test marker must be valid UTF-8 and at most 512 bytes")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		return "", ErrBusy
	}
	m.sequence++
	id := fmt.Sprintf("demo-%d", m.sequence)
	ctx, cancel := context.WithCancel(parent)
	m.active, m.cancel = id, cancel
	go m.stream(ctx, id, marker)
	return id, nil
}

func (m *Manager) stream(ctx context.Context, id, marker string) {
	defer m.finish(id)
	words := strings.Fields("DEMO ONLY — no provider request, no code execution. Visible capture marker: " + marker)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for _, word := range words {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.emit(ctx, id, "text", word+" ") {
				return
			}
		}
	}
	m.emit(ctx, id, "done", "")
}

// Emit under the same lock as Reset so no old event is delivered after Reset returns.
func (m *Manager) emit(ctx context.Context, id, kind, text string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != id || ctx.Err() != nil {
		return false
	}
	if kind == "done" {
		m.cancel()
		m.active, m.cancel = "", nil
	}
	m.sink(Event{RequestID: id, Kind: kind, Text: text})
	return true
}

func (m *Manager) finish(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == id {
		m.cancel()
		m.active, m.cancel = "", nil
	}
}

func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.sink(Event{RequestID: m.active, Kind: "cancelled"})
		m.active, m.cancel = "", nil
	}
}

// Reset also cancels work. The next request gets a different identity.
func (m *Manager) Reset() { m.Cancel() }
