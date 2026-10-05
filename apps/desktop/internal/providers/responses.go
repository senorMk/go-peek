package providers

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/senorMk/go-peek/apps/desktop/internal/prompts"
)

type Message struct {
	Role, Content string
	Images        []string
}
type Usage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}
type Provider interface {
	Stream(context.Context, []Message, func(string) error) (*Usage, error)
}
type Responses struct {
	client    openai.Client
	model     string
	closeIdle func()
}

func New(provider, model, key string) *Responses {
	base := "https://api.openai.com/v1/"
	if provider == "zen" {
		base = "https://opencode.ai/zen/v1/"
	}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second, IdleConnTimeout: 30 * time.Second}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return newClient(base, model, key, client)
}
func newClient(base, model, key string, client *http.Client) *Responses {
	return &Responses{closeIdle: client.CloseIdleConnections, model: model, client: openai.NewClient(option.WithAPIKey(key), option.WithBaseURL(base), option.WithHTTPClient(client), option.WithMaxRetries(0))}
}
func (p *Responses) Stream(ctx context.Context, messages []Message, emit func(string) error) (*Usage, error) {
	defer p.closeIdle()
	input := make([]responses.ResponseInputItemUnionParam, 0, len(messages))
	for _, message := range messages {
		if len(message.Images) == 0 {
			input = append(input, responses.ResponseInputItemParamOfMessage(message.Content, responses.EasyInputMessageRole(message.Role)))
			continue
		}
		if !SupportsImages(p.model) {
			return nil, errors.New("image input is not enabled for this model; choose gpt-6-luna, gpt-6-sol, or gpt-6.1-sol, or explicitly send typed text only")
		}
		if err := prompts.ValidateScreenshots(message.Images); err != nil {
			return nil, err
		}
		content := responses.ResponseInputMessageContentListParam{responses.ResponseInputContentParamOfInputText(message.Content)}
		for _, image := range message.Images {
			content = append(content, responses.ResponseInputContentUnionParam{OfInputImage: &responses.ResponseInputImageParam{ImageURL: openai.String(image), Detail: responses.ResponseInputImageDetailHigh}})
		}
		input = append(input, responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRole(message.Role)))
	}
	stream := p.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Model:        shared.ResponsesModel(p.model),
		Instructions: openai.String(prompts.Instructions),
		Input:        responses.ResponseNewParamsInputUnion{OfInputItemList: input},
		Store:        openai.Bool(false), MaxOutputTokens: openai.Int(4096),
		Truncation: responses.ResponseNewParamsTruncationDisabled,
	})
	defer stream.Close()
	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case "response.output_text.delta", "response.refusal.delta":
			if err := emit(event.Delta); err != nil {
				return nil, err
			}
		case "response.completed":
			usage := event.Response.Usage
			if usage.InputTokens != 0 || usage.OutputTokens != 0 {
				return &Usage{usage.InputTokens, usage.OutputTokens}, nil
			}
			return nil, nil
		case "response.incomplete":
			return nil, errors.New("response reached a provider limit; partial text is shown. Ask for a shorter answer or retry explicitly")
		case "response.failed", "error":
			return nil, errors.New("provider generation failed; check model access and retry explicitly")
		case "":
			return nil, errors.New("provider sent a malformed stream event")
		}
	}
	if err := stream.Err(); err != nil {
		return nil, normalize(err)
	}
	return nil, errors.New("response stream ended before completion; partial text is shown, and nothing was retried automatically")
}
func normalize(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("provider request timed out; retry explicitly")
	}
	var api *openai.Error
	if errors.As(err, &api) {
		switch api.StatusCode {
		case 401, 403:
			return errors.New("provider rejected authentication or model access; check your saved key and selected model")
		case 429:
			return errors.New("provider rate or credit limit reached; check your account and retry later")
		case 400, 404, 422:
			return errors.New("provider rejected the request or model; verify this model supports Responses streaming and PNG image input, or explicitly send typed text only")
		}
	}
	return errors.New("provider connection or stream failed; check connectivity and retry explicitly")
}

// Initial image-enabled subset; provider access is still checked by its API.
func SupportsImages(model string) bool {
	return model == "gpt-6-luna" || model == "gpt-6-sol" || model == "gpt-6.1-sol"
}
