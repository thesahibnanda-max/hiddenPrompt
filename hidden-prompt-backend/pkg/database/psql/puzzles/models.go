package puzzlesRepo

import (
	"fmt"
	puzzlesInternalPSQL "hidden-prompt-backend/pkg/database/psql/puzzles/internal"
	"hidden-prompt-backend/pkg/utils"
	"strings"

	"time"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
)

type MessageRole string

const (
	AssistantRole MessageRole = "assistant"
	UserRole      MessageRole = "user"
)

type Puzzle struct {
	PuzzleID              string         `json:"puzzle_id"`
	UserEmail             string         `json:"user_email"`
	HiddenPrompt          string         `json:"hidden_prompt"`
	CanonicalAnswer       string         `json:"canonical_answer"`
	HiddenPromptEmbedding []float32      `json:"hidden_prompt_embedding"`
	Messages              PuzzleMessages `json:"messages"`
	Metadata              PuzzleMetadata `json:"metadata"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type PuzzleMessages struct {
	MessageList []MessageElement `json:"message_list"`
}

type MessageElement struct {
	Role      MessageRole `json:"role"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

type PuzzleMetadata struct {
	MaxIntentSimilarityPercentage int  `json:"max_intent_similarity_percentage"`
	UserWinStatus                bool `json:"user_win_status"`
}

func (p *Puzzle) validate() error {
	if p == nil {
		return fmt.Errorf("p is nil")
	}

	// Puzzle ID
	p.PuzzleID = strings.TrimSpace(p.PuzzleID)
	if p.PuzzleID == "" {
		return fmt.Errorf("empty puzzle id")
	}

	// Email
	userEmail, err := utils.ValidateEmail(p.UserEmail)
	if err != nil {
		return err
	}
	p.UserEmail = userEmail

	// Hidden Prompt
	p.HiddenPrompt = strings.TrimSpace(p.HiddenPrompt)
	if p.HiddenPrompt == "" {
		return fmt.Errorf("hidden prompt is null")
	}

	// Canonical Answer
	p.CanonicalAnswer = strings.TrimSpace(p.CanonicalAnswer)
	if p.CanonicalAnswer == "" {
		return fmt.Errorf("canonical answer is null")
	}

	// Hidden Prompt Embedding
	if len(p.HiddenPromptEmbedding) == 0 {
		return fmt.Errorf("hidden prompt embedding is nil")
	}

	if p.Messages.MessageList == nil {
		p.Messages.MessageList = make([]MessageElement, 0)
	}

	return nil
}

func (r *repo) toModel(p puzzlesInternalPSQL.Puzzle) (*Puzzle, error) {

	messages, err := goxJsonUtils.StringObjectMapToObject[PuzzleMessages](p.Messages)
	if err != nil {
		return nil, err
	}
	if messages.MessageList == nil {
		messages.MessageList = make([]MessageElement, 0)
	}

	metadata, err := goxJsonUtils.StringObjectMapToObject[PuzzleMetadata](p.Metadata)
	if err != nil {
		return nil, err
	}

	return &Puzzle{
		PuzzleID:              p.PuzzleID,
		UserEmail:             p.UserEmail,
		HiddenPrompt:          p.HiddenPrompt,
		CanonicalAnswer:       p.CanonicalAnswer,
		HiddenPromptEmbedding: p.HiddenPromptEmbedding.Slice(),
		Messages:              messages,
		Metadata:              metadata,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}, nil
}

// toSummaryModel converts GetPuzzlesByUser's narrow projection row into a
// Puzzle. UserEmail, HiddenPrompt, HiddenPromptEmbedding, and Messages are
// left zero-valued - this query never selects those columns, since no
// caller of GetPuzzlesByEmail touches them (see the query's doc comment).
func (r *repo) toSummaryModel(p puzzlesInternalPSQL.GetPuzzlesByUserRow) (*Puzzle, error) {
	metadata, err := goxJsonUtils.StringObjectMapToObject[PuzzleMetadata](p.Metadata)
	if err != nil {
		return nil, err
	}

	return &Puzzle{
		PuzzleID:        p.PuzzleID,
		CanonicalAnswer: p.CanonicalAnswer,
		HiddenPrompt:    p.HiddenPrompt,
		Metadata:        metadata,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}, nil
}
