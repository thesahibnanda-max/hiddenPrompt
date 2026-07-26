package models

import "time"

type PuzzleElement struct {
	PuzzleID        string `json:"puzzle_id"`
	CanonicalAnswer string `json:"canonical_answer"`
}

type MessageRole string

const (
	AssistantRole MessageRole = "assistant"
	UserRole      MessageRole = "user"
)

type MessageElement struct {
	Role      MessageRole `json:"role"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

type GuessMetrics struct {
	EmbeddingDotProduct   float64 `json:"embedding_dot_product"`
	CosineSimilarityScore float64 `json:"cosine_similarity_score"`
	LLMSimilarityScore    float64 `json:"llm_similarity_score"`
	FinalSimilarityScore  float64 `json:"final_similarity_score"`
	ConsideredWon         bool    `json:"considered_won"`
}

type PuzzleDetails struct {
	PuzzleID                      string           `json:"puzzle_id"`
	CanonicalAnswer               string           `json:"canonical_answer"`
	HiddenPrompt                  string           `json:"hidden_prompt,omitempty"`
	Messages                      []MessageElement `json:"messages"`
	MaxIntentSimilarityPercentage int              `json:"max_intent_similarity_percentage"`
	UserWinStatus                 bool             `json:"user_win_status"`
	LatestGuessMetrics            *GuessMetrics    `json:"latest_guess_metrics,omitempty"`
	CreatedAt                     time.Time        `json:"created_at"`
	UpdatedAt                     time.Time        `json:"updated_at"`
}

type GetPuzzleByIDRequest struct {
	Email    string `json:"email"`
	PuzzleID string `json:"puzzle_id"`
}

type AddUserPromptRequest struct {
	PuzzleID   string `json:"puzzle_id"`
	UserPrompt string `json:"user_prompt"`
	Email      string `json:"email"`
}
