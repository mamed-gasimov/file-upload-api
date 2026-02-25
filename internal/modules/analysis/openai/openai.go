package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/mamed-gasimov/file-service/internal/modules/analysis"
)

type Provider struct {
	client openai.Client
}

var _ analysis.Provider = (*Provider)(nil)

func NewProvider(apiKey string, baseURL string) *Provider {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	return &Provider{client: client}
}

func (p *Provider) FileResume(ctx context.Context, input string) (string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant. Generate a brief, concise overview (2-3 sentences) of the provided file content. Focus on the purpose and key elements of the file."),
			openai.UserMessage(input),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from openai")
	}

	return resp.Choices[0].Message.Content, nil
}
