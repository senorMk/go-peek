package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func receive(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a stream event")
		return Event{}
	}
}

func TestResetStopsOldStreamAndUsesNewIdentity(t *testing.T) {
	events := make(chan Event, 100)
	m := New(func(e Event) { events <- e })
	old, err := m.Start(context.Background(), "old marker")
	if err != nil {
		t.Fatal(err)
	}
	if event := receive(t, events); event.RequestID != old || event.Kind != "text" {
		t.Fatalf("unexpected event: %+v", event)
	}
	m.Reset()
	// Drain everything delivered before Reset returned, including its terminal event.
	for len(events) > 0 {
		<-events
	}
	next, err := m.Start(context.Background(), "new marker")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Reset()
	if next == old {
		t.Fatal("request identity was reused")
	}
	event := receive(t, events)
	if event.RequestID != next {
		t.Fatalf("old request survived reset: %+v", event)
	}
}

func TestCancelAllowsNextRequest(t *testing.T) {
	events := make(chan Event, 100)
	m := New(func(e Event) { events <- e })
	first, err := m.Start(context.Background(), "cancel marker")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(context.Background(), "other"); !errors.Is(err, ErrBusy) {
		t.Fatalf("expected busy error, got %v", err)
	}
	m.Cancel()
	event := receive(t, events)
	if event.RequestID != first || event.Kind != "cancelled" {
		t.Fatalf("unexpected cancellation: %+v", event)
	}
	if _, err := m.Start(context.Background(), "next"); err != nil {
		t.Fatal(err)
	}
	m.Reset()
}

func TestValidation(t *testing.T) {
	m := New(func(Event) {})
	for _, marker := range []string{"", "  ", strings.Repeat("x", MaxMarkerBytes+1), string([]byte{0xff})} {
		if _, err := m.Start(context.Background(), marker); err == nil {
			t.Fatalf("accepted invalid marker of length %d", len(marker))
		}
	}
}

func TestStreamCompletesAndParentCancellationStopsWork(t *testing.T) {
	events := make(chan Event, 100)
	m := New(func(e Event) { events <- e })
	ctx, cancel := context.WithCancel(context.Background())
	id, err := m.Start(ctx, "marker")
	if err != nil {
		t.Fatal(err)
	}
	receive(t, events)
	cancel()
	m.Reset()
	for len(events) > 0 {
		<-events
	}
	next, err := m.Start(context.Background(), "complete")
	if err != nil {
		t.Fatal(err)
	}
	if next == id {
		t.Fatal("request identity reused")
	}
	defer m.Reset()
	var text strings.Builder
	for {
		event := receive(t, events)
		if event.RequestID != next {
			t.Fatalf("stale event: %+v", event)
		}
		if event.Kind == "done" {
			break
		}
		text.WriteString(event.Text)
	}
	if !strings.Contains(text.String(), "DEMO ONLY") || !strings.Contains(text.String(), "complete") {
		t.Fatalf("unexpected demo content: %s", text.String())
	}
}
