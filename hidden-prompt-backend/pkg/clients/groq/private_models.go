package groqClient

type groqClientRequest struct {
	Messages            []groqClientRequestMessage `json:"messages"`
	Model               string                     `json:"model"`
	Temperature         float64                    `json:"temperature"`
	MaxCompletionTokens int                        `json:"max_completion_tokens"`
	TopP                float64                    `json:"top_p"`
	Stream              bool                       `json:"stream"`
	Stop                []string                   `json:"stop"`
	ReasoningEffort     string                     `json:"reasoning_effort,omitempty"`
}

type groqClientRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqClientUsage struct {
	QueueTime        float64 `json:"queue_time"`
	PromptTokens     int     `json:"prompt_tokens"`
	PromptTime       float64 `json:"prompt_time"`
	CompletionTokens int     `json:"completion_tokens"`
	CompletionTime   float64 `json:"completion_time"`
	TotalTokens      int     `json:"total_tokens"`
	TotalTime        float64 `json:"total_time"`
}

type groqClientXGroq struct {
	ID   string `json:"id"`
	Seed int    `json:"seed"`
}

type groqClientChoice struct {
	Index        int                      `json:"index"`
	Message      groqClientRequestMessage `json:"message"`
	FinishReason string                   `json:"finish_reason"`
}

type groqClientResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []groqClientChoice `json:"choices"`
	Usage             groqClientUsage    `json:"usage"`
	SystemFingerprint string             `json:"system_fingerprint"`
	XGroq             groqClientXGroq    `json:"x_groq"`
	ServiceTier       string             `json:"service_tier"`
}
