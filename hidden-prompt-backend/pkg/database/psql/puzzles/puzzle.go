package puzzlesRepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	puzzlesInternalPSQL "hidden-prompt-backend/pkg/database/psql/puzzles/internal"
	"hidden-prompt-backend/pkg/utils"
	"time"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

var (
	ErrNoRows                            = errors.New("no rows in db")
	ErrSimilarityPercentageNotIncreasing = errors.New("max intent similarity percentage must be greater than current value")
	ErrInvalidSimilarityPercentage       = errors.New("max intent similarity percentage must be between 0 and 100")
	// ErrConcurrentModification is returned when an update's optimistic
	// concurrency check fails - the row's updated_at no longer matches
	// currentUpdatedAt, meaning another write landed first. Callers should
	// re-fetch the puzzle and retry against the fresh data.
	ErrConcurrentModification = errors.New("puzzle was concurrently modified")
)

type Repository interface {
	CreatePuzzle(ctx context.Context, puzzle Puzzle) (*Puzzle, error)
	GetPuzzleByID(ctx context.Context, id string) (*Puzzle, error)
	// GetPuzzlesByEmail returns a summary projection, not full rows:
	// HiddenPrompt, HiddenPromptEmbedding, Messages, and UserEmail are left
	// zero-valued on every returned Puzzle. Use GetPuzzleByID when full
	// puzzle data is needed.
	GetPuzzlesByEmail(ctx context.Context, email string) ([]Puzzle, error)
	// AppendPuzzleMessage appends to currentMessages - the caller's
	// already-loaded copy - rather than re-fetching the puzzle internally
	// just to read its message list. currentUpdatedAt is the row's
	// updated_at at the time currentMessages was read; if another write
	// has landed since, this returns ErrConcurrentModification and the
	// caller must re-fetch and retry.
	AppendPuzzleMessage(ctx context.Context, id string, currentMessages PuzzleMessages, currentUpdatedAt time.Time, role MessageRole, message string) (*Puzzle, error)
	// UpdateMaxIntentSimilarityPercentage checks the "must increase" rule
	// against currentMetadata - the caller's already-loaded copy - rather
	// than re-fetching internally. currentUpdatedAt behaves as in
	// AppendPuzzleMessage.
	UpdateMaxIntentSimilarityPercentage(ctx context.Context, id string, currentMetadata PuzzleMetadata, currentUpdatedAt time.Time, percentage int) (*Puzzle, error)
	// MarkUserWon takes the caller's already-loaded current puzzle
	// instead of re-fetching internally - returns it unchanged (not an
	// error) if it was already won. Otherwise behaves as in
	// AppendPuzzleMessage: a concurrent write since current was read
	// yields ErrConcurrentModification.
	MarkUserWon(ctx context.Context, id string, current Puzzle) (*Puzzle, error)
	DeletePuzzlesByEmail(ctx context.Context, email string) error
}

type repo struct {
	q puzzlesInternalPSQL.Querier
}

func (r *repo) CreatePuzzle(ctx context.Context, puzzle Puzzle) (*Puzzle, error) {
	if err := puzzle.validate(); err != nil {
		return nil, err
	}

	messages, err := goxJsonUtils.ObjectToStringObjectMap(puzzle.Messages)
	if err != nil {
		return nil, err
	}

	// New puzzles always start with zero-value metadata (0 similarity, not won),
	// regardless of what the caller set on puzzle.Metadata.
	metadata, err := goxJsonUtils.ObjectToStringObjectMap(PuzzleMetadata{})
	if err != nil {
		return nil, err
	}

	p, err := r.q.CreatePuzzle(ctx, puzzlesInternalPSQL.CreatePuzzleParams{
		PuzzleID:              puzzle.PuzzleID,
		UserEmail:             puzzle.UserEmail,
		HiddenPrompt:          puzzle.HiddenPrompt,
		CanonicalAnswer:       puzzle.CanonicalAnswer,
		HiddenPromptEmbedding: pgvector.NewVector(puzzle.HiddenPromptEmbedding),
		Messages:              messages,
		Metadata:              metadata,
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(p)
}

func (r *repo) GetPuzzleByID(ctx context.Context, id string) (*Puzzle, error) {
	return r.getPuzzleByID(ctx, id)
}

func (r *repo) GetPuzzlesByEmail(ctx context.Context, email string) ([]Puzzle, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return nil, err
	}
	ps, err := r.q.GetPuzzlesByUser(ctx, email)
	if err != nil {
		return nil, err
	}

	if len(ps) == 0 {
		return make([]Puzzle, 0), nil
	}

	psModel := make([]Puzzle, 0)
	for _, p := range ps {
		pModel, err := r.toSummaryModel(p)
		if err != nil {
			return nil, err
		}
		if pModel == nil {
			return nil, fmt.Errorf("nil model conversion")
		}
		psModel = append(psModel, *pModel)
	}
	return psModel, nil
}

func (r *repo) AppendPuzzleMessage(ctx context.Context, id string, currentMessages PuzzleMessages, currentUpdatedAt time.Time, role MessageRole, message string) (*Puzzle, error) {
	currentMessages.MessageList = append(currentMessages.MessageList, MessageElement{
		Role:      role,
		Message:   message,
		Timestamp: time.Now(),
	})

	messageForInternal, err := goxJsonUtils.ObjectToStringObjectMap(currentMessages)
	if err != nil {
		return nil, err
	}

	pInternal, err := r.q.UpdatePuzzleMessages(ctx, puzzlesInternalPSQL.UpdatePuzzleMessagesParams{
		PuzzleID:  id,
		Messages:  messageForInternal,
		UpdatedAt: currentUpdatedAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConcurrentModification
		}
		return nil, err
	}

	return r.toModel(pInternal)
}

func (r *repo) UpdateMaxIntentSimilarityPercentage(ctx context.Context, id string, currentMetadata PuzzleMetadata, currentUpdatedAt time.Time, percentage int) (*Puzzle, error) {
	if percentage < 0 || percentage > 100 {
		return nil, ErrInvalidSimilarityPercentage
	}

	if percentage <= currentMetadata.MaxIntentSimilarityPercentage {
		return nil, ErrSimilarityPercentageNotIncreasing
	}

	currentMetadata.MaxIntentSimilarityPercentage = percentage
	return r.updateMetadata(ctx, id, currentMetadata, currentUpdatedAt)
}

func (r *repo) MarkUserWon(ctx context.Context, id string, current Puzzle) (*Puzzle, error) {
	if current.Metadata.UserWinStatus {
		return &current, nil
	}

	current.Metadata.UserWinStatus = true
	return r.updateMetadata(ctx, id, current.Metadata, current.UpdatedAt)
}

func (r *repo) DeletePuzzlesByEmail(ctx context.Context, email string) error {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return err
	}

	return r.q.DeletePuzzlesByUser(ctx, email)
}

func (r *repo) updateMetadata(ctx context.Context, id string, metadata PuzzleMetadata, currentUpdatedAt time.Time) (*Puzzle, error) {
	metadataMap, err := goxJsonUtils.ObjectToStringObjectMap(metadata)
	if err != nil {
		return nil, err
	}

	pInternal, err := r.q.UpdatePuzzleMetadata(ctx, puzzlesInternalPSQL.UpdatePuzzleMetadataParams{
		PuzzleID:  id,
		Metadata:  metadataMap,
		UpdatedAt: currentUpdatedAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConcurrentModification
		}
		return nil, err
	}

	return r.toModel(pInternal)
}

func (r *repo) getPuzzleByID(ctx context.Context, id string) (*Puzzle, error) {
	p, err := r.q.GetPuzzleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, err
	}

	return r.toModel(p)
}
