package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/senorMk/go-peek/apps/desktop/internal/prompts"
	"github.com/senorMk/go-peek/apps/desktop/internal/providers"
)

const MaxContextBytes = 64 * 1024
const MaxResponseBytes = 32 * 1024

type Assistant struct {
	mu       sync.Mutex
	sequence uint64
	active   string
	cancel   context.CancelFunc
	history  []providers.Message
	sink     Sink
}

func NewAssistant(sink Sink) *Assistant { return &Assistant{sink: sink} }
func (a *Assistant) Start(ctx context.Context, problem prompts.Problem, provider providers.Provider) (string, error) {
	prompt, err := prompts.Build(problem)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.begin(ctx, []providers.Message{{Role: "user", Content: prompt, Images: append([]string(nil), problem.Screenshots...)}}, provider)
}
func (a *Assistant) Followup(ctx context.Context, text string, provider providers.Provider) (string, error) {
	if strings.TrimSpace(text) == "" || len(text) > 4096 || !utf8.ValidString(text) {
		return "", errors.New("follow-up must be nonempty, valid UTF-8, and at most 4 KiB")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.history) == 0 {
		return "", errors.New("complete a response before asking a follow-up")
	}
	messages := append([]providers.Message(nil), a.history...)
	messages = append(messages, providers.Message{Role: "user", Content: text})
	return a.begin(ctx, messages, provider)
}
func (a *Assistant) begin(parent context.Context, messages []providers.Message, provider providers.Provider) (string, error) {
	if a.cancel != nil {
		return "", errors.New("a response is already streaming; cancel it first")
	}
	size := len(prompts.Instructions)
	imageBytes := 0
	for _, message := range messages {
		size += len(message.Content)
		for _, image := range message.Images {
			imageBytes += len(image)
		}
	}
	if imageBytes > prompts.MaxImagesEncodedBytes {
		return "", errors.New("conversation screenshots exceed the image limit; start a new problem with fewer or smaller images")
	}
	if size > MaxContextBytes {
		return "", errors.New("conversation exceeds the 64 KiB context limit; reset or shorten the problem explicitly")
	}
	a.sequence++
	id := fmt.Sprintf("response-%d", a.sequence)
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	a.active, a.cancel = id, cancel
	if len(messages) == 1 {
		a.history = nil
	}
	go a.generate(ctx, id, messages, provider)
	return id, nil
}
func (a *Assistant) generate(ctx context.Context, id string, messages []providers.Message, provider providers.Provider) {
	var output strings.Builder
	usage, err := provider.Stream(ctx, messages, func(text string) error {
		if output.Len()+len(text) > MaxResponseBytes {
			return errors.New("response exceeds the 32 KiB display limit; request a shorter answer")
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.active != id {
			return context.Canceled
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		output.WriteString(text)
		a.sink(Event{RequestID: id, Kind: "text", Text: text})
		return nil
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	if err == nil {
		err = ctx.Err()
	}
	if a.active != id {
		return
	}
	a.cancel()
	a.active, a.cancel = "", nil
	if err != nil {
		message := err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			message = "provider request timed out; retry explicitly"
		}
		if errors.Is(err, context.Canceled) {
			a.sink(Event{RequestID: id, Kind: "cancelled"})
			return
		}
		a.sink(Event{RequestID: id, Kind: "error", Error: message})
		return
	}
	if output.Len() == 0 {
		a.sink(Event{RequestID: id, Kind: "error", Error: "provider completed without text; verify model compatibility"})
		return
	}
	a.history = append(append([]providers.Message(nil), messages...), providers.Message{Role: "assistant", Content: output.String()})
	a.sink(Event{RequestID: id, Kind: "done", Usage: usage})
}
func (a *Assistant) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.sink(Event{RequestID: a.active, Kind: "cancelled"})
		a.active, a.cancel = "", nil
	}
}
func (a *Assistant) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.sink(Event{RequestID: a.active, Kind: "cancelled"})
		a.active, a.cancel = "", nil
	}
	a.history = nil
}
