package geminiClient

/*
{
        "model": "models/gemini-embedding-2",
        "content": {
        "parts": [{
            "text": "What is the meaning of life?"
        }]
        }
*/

type geminiRequestPartsText struct {
	Text string `json:"text"`
}

type geminiRequestContent struct {
	Parts []geminiRequestPartsText `json:"parts"`
}

type geminiRequest struct {
	Model                string               `json:"model"`
	Content              geminiRequestContent `json:"content"`
	OutputDimensionality int                  `json:"outputDimensionality,omitempty"`
}

type geminiResponse struct {
	Embedding     geminiEmbedding     `json:"embedding"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
}

type geminiEmbedding struct {
	Values []float32 `json:"values"`
}

type geminiUsageMetadata struct {
	PromptTokenCount   float64                    `json:"promptTokenCount"`
	PromptTokenDetails []geminiPromptTokenDetails `json:"promptTokenDetails"`
}

type geminiPromptTokenDetails struct {
	Modality   string `json:"modality"`
	TokenCount int64  `json:"tokenCount"`
}
