package groqClient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"hidden-prompt-backend/pkg/config"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/utils"

	"github.com/devlibx/gox-base/v2"
	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	"github.com/samber/lo"
	"github.com/thesahibnanda-max/fastclient"
)

type CallRequest struct {
	UserPrompt   string `json:"user_prompt"`
	SystemPrompt string `json:"system_prompt"`
	// MaxTemperature, when set, narrows the sampled temperature to a
	// uniform [0.8, *MaxTemperature] instead of the default wide weighted
	// bands (which go up to 2.0) - for call types that need much more
	// consistency than the default range gives. Nil preserves the existing
	// random behavior unchanged.
	MaxTemperature *float64 `json:"max_temperature,omitempty"`
	ModelsToAvoid  []string `json:"models_to_avoid"` // avoid these models if mentioned
	ModelToUse     *string  `json:"model_to_use"`
}

type CallResponse struct {
	Response    []string            `json:"response"`
	RawResponse gox.StringObjectMap `json:"raw_response"`
}

type Client interface {
	Call(ctx context.Context, req CallRequest) (*CallResponse, error)
}

type model struct {
	key            string
	reasoningIfAny string
}

type client struct {
	fClient             *fastclient.Client
	apiKeys             []string
	models              utils.ThreadSafeWeightedMap[model]
	modelKeyToReasoning map[string]model
	indexAtomic         *atomic.Uint64
}

// NewClient builds a Groq Client. API keys are loaded from config.GetConfig,
// requests are round-robined across all configured keys, and the chat model
// used per-call is chosen by weighted random selection.
func NewClient() (Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("groq: load config: %w", err)
	}
	if len(cfg.GroqAPIKeys) == 0 {
		return nil, errors.New("groq: no api keys configured")
	}

	fClient := fastclient.New().
		SetBaseURL("https://api.groq.com").
		SetTimeout(30*time.Second).
		SetRetryCount(2).
		SetRetryOnStatusCodes(http.StatusTooManyRequests, http.StatusServiceUnavailable)

	keyToModelMap := map[string]model{
		constants.Llama318B:    {key: constants.Llama318B},
		constants.GroqCompound: {key: constants.GroqCompound},
		constants.Llama3370B:   {key: constants.Llama3370B},
		constants.GPTOSS20B:    {key: constants.GPTOSS20B, reasoningIfAny: "medium"},
		constants.Qwen3627B:    {key: constants.Qwen3627B, reasoningIfAny: "default"},
		constants.GPTOSS120B:   {key: constants.GPTOSS120B, reasoningIfAny: "medium"},
	}

	models, err := utils.NewThreadSafeWeightedMap(map[model]uint{
		keyToModelMap[constants.Llama318B]:    65,
		keyToModelMap[constants.GroqCompound]: 8,
		keyToModelMap[constants.Llama3370B]:   14,
		keyToModelMap[constants.GPTOSS20B]:    6,
		keyToModelMap[constants.Qwen3627B]:    6,
		keyToModelMap[constants.GPTOSS120B]:   1,
	})
	if err != nil {
		return nil, err
	}

	return &client{
		fClient:             fClient,
		apiKeys:             cfg.GroqAPIKeys,
		models:              models,
		modelKeyToReasoning: keyToModelMap,
		indexAtomic:         new(atomic.Uint64),
	}, nil
}

func (c *client) Call(ctx context.Context, req CallRequest) (*CallResponse, error) {
	i := c.indexAtomic.Add(1) - 1
	apiKey := strings.TrimSpace(c.apiKeys[i%uint64(len(c.apiKeys))])
	if apiKey == "" {
		return nil, errors.New("groq: api key is missing")
	}

	userPrompt := strings.TrimSpace(req.UserPrompt)
	systemPrompt := strings.TrimSpace(req.SystemPrompt)
	if userPrompt == "" {
		return nil, errors.New("groq: user prompt is missing")
	}

	messages := []groqClientRequestMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = append([]groqClientRequestMessage{
			{Role: "system", Content: systemPrompt},
		}, messages...)
	}

	chosenModel := c.chooseModel(req.ModelsToAvoid, req.ModelToUse)
	requestBody := groqClientRequest{
		Messages:            messages,
		Model:               chosenModel.key,
		Temperature:         c.chooseTemperature(req.MaxTemperature),
		MaxCompletionTokens: 512,
		TopP:                utils.ChooseOneRandomly(utils.GenerateRandomFloat64(0.9, 1.0), 1.0),
		Stream:              false,
		Stop:                nil,
		ReasoningEffort:     chosenModel.reasoningIfAny,
	}

	var response groqClientResponse
	resp, err := c.fClient.R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetResult(&response).
		SetTimeout(45*time.Second).
		PostCtx(ctx, "/openai/v1/chat/completions", requestBody)
	if err != nil {
		return nil, fmt.Errorf("groq: call failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("groq: status code is %d, body is %s", resp.StatusCode(), resp.String())
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("groq: no choices returned")
	}

	s := make([]string, 0, len(response.Choices))
	for _, choice := range response.Choices {
		ans := strings.TrimSpace(choice.Message.Content)
		if ans != "" {
			s = append(s, ans)
		}
	}

	return &CallResponse{
		Response:    s,
		RawResponse: goxJsonUtils.StringToStringObjectMapSuppressError(resp.String()),
	}, nil
}

func (c *client) chooseModel(modelsToAvoid []string, modelToUse *string) model {
	if modelToUse != nil {
		model, ok := c.modelKeyToReasoning[lo.FromPtr(modelToUse)]
		if ok {
			return model
		}
	}
	if len(modelsToAvoid) == 0 {
		return c.models.GetRandom()
	}
	return c.models.GetRandomAvoiding(
		lo.FilterMap(
			modelsToAvoid,
			func(key string, _ int) (model, bool) {
				m, ok := c.modelKeyToReasoning[key]
				return m, ok
			},
		)...,
	)
}

// chooseTemperature samples the default wide, weighted-band temperature
// (up to 2.0) unless maxTemperature caps it, in which case it samples
// uniformly in [0.8, *maxTemperature] instead - for call types that need
// far more consistency than the default range gives.
func (c *client) chooseTemperature(maxTemperature *float64) float64 {
	if maxTemperature != nil {
		return utils.GenerateRandomFloat64(0.8, *maxTemperature)
	}
	return utils.GenerateRandomFloat64(
		0.8,
		2.0,
		utils.RangeWeight{Start: 0.8, End: 1.2, Weight: 7},
		utils.RangeWeight{Start: 1.2, End: 1.6, Weight: 2},
		utils.RangeWeight{Start: 1.6, End: 2.0, Weight: 1},
	)
}
