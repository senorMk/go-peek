package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func mockClient(handler http.HandlerFunc) *http.Client {
	return &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		response := recorder.Result()
		response.Request = r
		return response, nil
	})}
}

type blockedBody struct{ ctx context.Context }

func (b blockedBody) Read([]byte) (int, error) { <-b.ctx.Done(); return 0, b.ctx.Err() }
func (b blockedBody) Close() error             { return nil }

var _ io.ReadCloser = blockedBody{}

func TestResponsesStreamsAndSendsPrivacyOptions(t *testing.T) {
	client := mockClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("wrong route or explicit key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["store"] != false || body["stream"] != true || body["truncation"] != "disabled" {
			t.Error("privacy/stream options missing")
		}
		if body["model"] != "configured-model" || body["max_output_tokens"] != float64(4096) {
			t.Error("model or output cap missing")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":2,\"output_tokens\":3}}}\n\n")
	}))
	p := newClient("https://example.test/", "configured-model", "test-key", client)
	var text strings.Builder
	usage, err := p.Stream(context.Background(), []Message{{Role: "user", Content: "problem"}}, func(s string) error { text.WriteString(s); return nil })
	if err != nil || text.String() != "hello" || usage == nil || usage.InputTokens != 2 || usage.OutputTokens != 3 {
		t.Fatalf("unexpected result: %q %+v %v", text.String(), usage, err)
	}
}
func TestStreamFailuresNeverLeakProviderTextOrReplay(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"auth", 401, `{"error":{"message":"private-key private-source","type":"authentication_error"}}`},
		{"rate", 429, `{"error":{"message":"private-key private-source"}}`},
		{"malformed", 200, "data: {broken}\n\n"},
		{"failed", 200, "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"private-source\"}}}\n\n"},
		{"partial", 200, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"},
		{"incomplete", 200, "data: {\"type\":\"response.incomplete\"}\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			client := mockClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			_, err := newClient("https://example.test/", "model", "test-key", client).Stream(context.Background(), nil, func(string) error { return nil })
			if err == nil || strings.Contains(err.Error(), "private-") {
				t.Fatalf("missing or unsafe error: %v", err)
			}
			if requests.Load() != 1 {
				t.Fatal("request was replayed")
			}
		})
	}
}
func TestStreamCancellationAndReadDeadline(t *testing.T) {
	client := &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: blockedBody{r.Context()}, Request: r}, nil
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := newClient("https://example.test/", "model", "test-key", client).Stream(ctx, nil, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected timeout: %v", err)
	}
}

func TestScreenshotRequestContainsOrderedImagesAndText(t *testing.T) {
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	first := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes.Bytes())
	pngBytes.Reset()
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 4, 5))); err != nil {
		t.Fatal(err)
	}
	second := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes.Bytes())
	for _, base := range []string{"https://api.openai.com/v1/", "https://opencode.ai/zen/v1/"} {
		t.Run(base, func(t *testing.T) {
			client := mockClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Input []struct {
						Role    string `json:"role"`
						Content []struct {
							Type     string `json:"type"`
							Text     string `json:"text"`
							ImageURL string `json:"image_url"`
							Detail   string `json:"detail"`
						} `json:"content"`
					} `json:"input"`
					Store bool `json:"store"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if len(body.Input) != 1 || body.Input[0].Role != "user" || len(body.Input[0].Content) != 3 {
					t.Fatal("image message missing")
				}
				parts := body.Input[0].Content
				if parts[0].Type != "input_text" || parts[0].Text != "hint from screenshots" {
					t.Fatal("task instructions lost")
				}
				for i, want := range []string{first, second} {
					if parts[i+1].Type != "input_image" || parts[i+1].ImageURL != want || parts[i+1].Detail != "high" {
						t.Fatal("image order, content, or detail changed")
					}
				}
				if body.Store {
					t.Fatal("image request opted into storage")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"type\":\"response.completed\"}\n\n")
			}))
			_, err := newClient(base, "gpt-6-luna", "test-key", client).Stream(context.Background(), []Message{{Role: "user", Content: "hint from screenshots", Images: []string{first, second}}}, func(string) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUnknownImageModelFailsBeforeSendingAnything(t *testing.T) {
	client := mockClient(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unsupported image request sent") }))
	_, err := newClient("https://example.test/", "unknown-model", "test-key", client).Stream(context.Background(), []Message{{Role: "user", Content: "problem", Images: []string{"private image data"}}}, func(string) error { return nil })
	if err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("missing or unsafe model capability error")
	}
}
