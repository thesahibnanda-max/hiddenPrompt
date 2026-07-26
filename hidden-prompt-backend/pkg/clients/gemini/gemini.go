package geminiClient

import (
	"context"
	"errors"
	"fmt"
	"hidden-prompt-backend/pkg/config"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thesahibnanda-max/fastclient"
)

const (
	geminiEmbeddingModel = "models/gemini-embedding-2"
	geminiDimension      = 768
)

type Client interface {
	GetEmbeddings(ctx context.Context, prompt string) ([]float32, error)
}

func NewClient() (Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("gemini: load config: %w", err)
	}
	if len(cfg.GeminiAPIKeys) == 0 {
		return nil, errors.New("gemini: no api keys configured")
	}

	fClient := fastclient.New().
		SetBaseURL("https://generativelanguage.googleapis.com").
		SetTimeout(30*time.Second).
		SetRetryCount(2).
		SetRetryOnStatusCodes(http.StatusTooManyRequests, http.StatusServiceUnavailable)

	return &client{
		fClient:     fClient,
		apiKeys:     cfg.GeminiAPIKeys,
		indexAtomic: new(atomic.Uint64),
	}, nil
}

type client struct {
	fClient     *fastclient.Client
	apiKeys     []string
	indexAtomic *atomic.Uint64
}

func (c *client) GetEmbeddings(ctx context.Context, prompt string) ([]float32, error) {
	i := c.indexAtomic.Add(1) - 1
	apiKey := strings.TrimSpace(c.apiKeys[i%uint64(len(c.apiKeys))])
	if apiKey == "" {
		return nil, errors.New("gemini: api key is missing")
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("gemini: empty prompts not allowed")
	}

	reqBody := geminiRequest{
		OutputDimensionality: geminiDimension,
		Model:                geminiEmbeddingModel,
		Content: geminiRequestContent{
			Parts: []geminiRequestPartsText{
				{
					Text: prompt,
				},
			},
		},
	}

	var result = geminiResponse{}
	resp, err := c.fClient.R().
		SetHeader("x-goog-api-key", apiKey).
		SetResult(&result).
		SetTimeout(45*time.Second).SetBody(reqBody).PostCtx(ctx, "/v1beta/models/gemini-embedding-2:embedContent")

	if err != nil {
		return nil, fmt.Errorf("gemini: client call error %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("gemini: status code is %d, body is %s", resp.StatusCode(), resp.String())
	}

	if len(result.Embedding.Values) == 0 {
		return nil, fmt.Errorf("gemini: values is nil ")
	}

	return result.Embedding.Values, nil
}
