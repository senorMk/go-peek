package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/senorMk/go-peek/apps/desktop/internal/prompts"
	"github.com/senorMk/go-peek/apps/desktop/internal/providers"
)

type fakeProvider struct {
	stream func(context.Context, []providers.Message, func(string) error) (*providers.Usage, error)
}

func (p fakeProvider) Stream(ctx context.Context, m []providers.Message, f func(string) error) (*providers.Usage, error) {
	return p.stream(ctx, m, f)
}
func problem() prompts.Problem {
	return prompts.Problem{Statement: "find a pair", Language: "Go", Action: "hint"}
}
func terminal(t *testing.T, events <-chan Event) Event {
	t.Helper()
	for {
		e := receive(t, events)
		if e.Kind != "text" {
			return e
		}
	}
}
func TestAssistantFollowupsAndResetIsolation(t *testing.T) {
	events := make(chan Event, 100)
	requests := make(chan []providers.Message, 10)
	a := NewAssistant(func(e Event) { events <- e })
	p := fakeProvider{func(ctx context.Context, m []providers.Message, emit func(string) error) (*providers.Usage, error) {
		requests <- m
		return nil, emit("answer")
	}}
	if _, err := a.Start(context.Background(), problem(), p); err != nil {
		t.Fatal(err)
	}
	if terminal(t, events).Kind != "done" {
		t.Fatal("initial response did not complete")
	}
	<-requests
	if _, err := a.Followup(context.Background(), "more specific", p); err != nil {
		t.Fatal(err)
	}
	terminal(t, events)
	m := <-requests
	if len(m) != 3 || m[1].Role != "assistant" || m[1].Content != "answer" || m[2].Content != "more specific" {
		t.Fatal("follow-up context missing")
	}
	a.Reset()
	if _, err := a.Followup(context.Background(), "old problem", p); err == nil {
		t.Fatal("old context survived reset")
	}
}
func TestAssistantDropsLateEventsAfterReset(t *testing.T) {
	events := make(chan Event, 100)
	a := NewAssistant(func(e Event) { events <- e })
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	p := fakeProvider{func(ctx context.Context, m []providers.Message, emit func(string) error) (*providers.Usage, error) {
		close(started)
		<-release
		err := emit("late")
		close(finished)
		return nil, err
	}}
	a.Start(context.Background(), problem(), p)
	<-started
	a.Reset()
	terminal(t, events)
	close(release)
	<-finished
	select {
	case e := <-events:
		t.Fatalf("late event survived: %+v", e)
	case <-time.After(20 * time.Millisecond):
	}
}
func TestAssistantBudgetAndPartialOutputLimits(t *testing.T) {
	events := make(chan Event, 100)
	a := NewAssistant(func(e Event) { events <- e })
	p := fakeProvider{func(ctx context.Context, m []providers.Message, emit func(string) error) (*providers.Usage, error) {
		return nil, emit(strings.Repeat("x", MaxResponseBytes+1))
	}}
	a.Start(context.Background(), problem(), p)
	if e := terminal(t, events); e.Kind != "error" {
		t.Fatal("unbounded output accepted")
	}
	a.mu.Lock()
	a.history = []providers.Message{{Role: "user", Content: strings.Repeat("x", MaxContextBytes)}}
	a.mu.Unlock()
	if _, err := a.Followup(context.Background(), "follow-up", p); err == nil {
		t.Fatal("oversize context silently accepted")
	}
	a.Reset()
}

func TestScreenshotsSurviveFollowupsAndAreReleasedOnReset(t *testing.T) {
	var data bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&data, image.NewRGBA(image.Rect(0, 0, 128, 128))); err != nil {
		t.Fatal(err)
	}
	imageURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
	if len(imageURL) <= MaxContextBytes {
		t.Fatal("fixture must exceed text budget")
	}
	events := make(chan Event, 20)
	requests := make(chan []providers.Message, 3)
	a := NewAssistant(func(e Event) { events <- e })
	p := fakeProvider{func(ctx context.Context, m []providers.Message, emit func(string) error) (*providers.Usage, error) {
		requests <- m
		return nil, emit("hint")
	}}
	problem := prompts.Problem{Screenshots: []string{imageURL}, Language: "Go", Action: "hint"}
	if _, err := a.Start(context.Background(), problem, p); err != nil {
		t.Fatal(err)
	}
	problem.Screenshots[0] = "changed after starting"
	if terminal(t, events).Kind != "done" {
		t.Fatal("screenshot request failed")
	}
	m := <-requests
	if len(m[0].Images) != 1 || m[0].Images[0] != imageURL {
		t.Fatal("request image snapshot changed")
	}
	if _, err := a.Followup(context.Background(), "next hint", p); err != nil {
		t.Fatal(err)
	}
	if terminal(t, events).Kind != "done" {
		t.Fatal("followup failed")
	}
	m = <-requests
	if len(m) != 3 || len(m[0].Images) != 1 || m[0].Images[0] != imageURL {
		t.Fatal("images lost in followup")
	}
	a.Reset()
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.history) != 0 {
		t.Fatal("reset retained screenshot history")
	}
}
