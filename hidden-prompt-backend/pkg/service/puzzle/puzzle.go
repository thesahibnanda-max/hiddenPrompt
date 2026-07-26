package puzzle

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	chatAndAIUsage "hidden-prompt-backend/pkg/chat"
	"hidden-prompt-backend/pkg/database"
	puzzlesRepo "hidden-prompt-backend/pkg/database/psql/puzzles"
	customErrors "hidden-prompt-backend/pkg/errors"
	"hidden-prompt-backend/pkg/models"
	"hidden-prompt-backend/pkg/service/intent"
	"hidden-prompt-backend/pkg/service/user"
	"hidden-prompt-backend/pkg/utils"
)

type Service interface {
	CreatePuzzleForUser(ctx context.Context, email string) ([]models.PuzzleElement, error)
	GetAllPuzzlesForUser(ctx context.Context, email string) ([]models.PuzzleElement, error)
	GetPuzzleByID(ctx context.Context, req models.GetPuzzleByIDRequest) (*models.PuzzleDetails, error)
	AddUserPrompt(ctx context.Context, req models.AddUserPromptRequest) (*models.PuzzleDetails, error)
}

type service struct {
	db            *database.DatabaseParams
	ai            chatAndAIUsage.AI
	userService   user.Service
	intentService intent.Service
}

func NewService(
	db *database.DatabaseParams,
	ai chatAndAIUsage.AI,
	userService user.Service,
	intentService intent.Service,
) (Service, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if ai == nil {
		return nil, fmt.Errorf("ai is nil")
	}
	if userService == nil {
		return nil, fmt.Errorf("userService is nil")
	}
	if intentService == nil {
		return nil, fmt.Errorf("intentService is nil")
	}

	return &service{
		db:            db,
		ai:            ai,
		userService:   userService,
		intentService: intentService,
	}, nil
}

func (s *service) GetAllPuzzlesForUser(ctx context.Context, email string) ([]models.PuzzleElement, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return nil, &customErrors.EMailIsInvalid{Err: err}
	}

	isVerified, err := s.userService.IsVerified(ctx, email)
	if err != nil {
		return nil, err
	}
	if !isVerified {
		return nil, &customErrors.UserIsNotVerified{Err: errors.New("user is not verified")}
	}

	puzzles, err := s.db.PuzzlesRepository.GetPuzzlesByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return s.toPuzzleElements(puzzles), nil
}

// toPuzzleElements maps repo puzzles to the public summary shape - shared
// by GetAllPuzzlesForUser and CreatePuzzleForUser so the latter can build
// its response from already-fetched/already-created data instead of
// re-querying for it.
func (s *service) toPuzzleElements(puzzles []puzzlesRepo.Puzzle) []models.PuzzleElement {
	elements := make([]models.PuzzleElement, len(puzzles))
	for i, p := range puzzles {
		elements[i] = models.PuzzleElement{
			PuzzleID:        p.PuzzleID,
			CanonicalAnswer: p.CanonicalAnswer,
		}
	}
	return elements
}

func (s *service) CreatePuzzleForUser(ctx context.Context, email string) ([]models.PuzzleElement, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return nil, &customErrors.EMailIsInvalid{Err: err}
	}

	isVerified, err := s.userService.IsVerified(ctx, email)
	if err != nil {
		return nil, err
	}
	if !isVerified {
		return nil, &customErrors.UserIsNotVerified{Err: errors.New("user is not verified")}
	}

	puzzles, err := s.db.PuzzlesRepository.GetPuzzlesByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	puzzleResp, err := s.ai.GeneratePuzzle(ctx, chatAndAIUsage.GeneratePuzzleRequest{
		UserContext:   s.buildUserContext(puzzles),
		RecentPuzzles: s.recentPuzzles(puzzles),
		UsedAnswers:   s.allTimeCanonicalAnswers(puzzles),
	})
	if err != nil {
		return nil, err
	}

	hiddenPromptEmbedding, err := s.ai.GenerateEmbeddings(ctx, puzzleResp.HiddenPrompt)
	if err != nil {
		return nil, err
	}

	created, err := s.db.PuzzlesRepository.CreatePuzzle(ctx, puzzlesRepo.Puzzle{
		PuzzleID:              utils.GenerateValidULID(),
		UserEmail:             email,
		HiddenPrompt:          puzzleResp.HiddenPrompt,
		CanonicalAnswer:       puzzleResp.CanonicalAnswer,
		HiddenPromptEmbedding: hiddenPromptEmbedding,
	})
	if err != nil {
		return nil, err
	}

	// Builds the response from the already-fetched puzzles plus the
	// just-created one, instead of the redundant IsVerified+
	// GetPuzzlesByEmail round-trip a call to GetAllPuzzlesForUser here
	// would repeat. Prepended, not appended - puzzles is newest-first
	// (see recentPuzzles' sort), and this new puzzle is now the newest.
	return s.toPuzzleElements(append([]puzzlesRepo.Puzzle{*created}, puzzles...)), nil
}

func (s *service) GetPuzzleByID(ctx context.Context, req models.GetPuzzleByIDRequest) (*models.PuzzleDetails, error) {
	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		return nil, &customErrors.EMailIsInvalid{Err: err}
	}

	if !utils.IsValidULID(req.PuzzleID) {
		return nil, &customErrors.PuzzleIDIsInvalid{Err: fmt.Errorf("invalid puzzle id: %q", req.PuzzleID)}
	}

	puzzle, err := s.getOwnedPuzzle(ctx, req.PuzzleID, email)
	if err != nil {
		return nil, err
	}

	s.sortMessagesByTimestamp(puzzle.Messages.MessageList)

	return s.toPuzzleDetails(puzzle, nil), nil
}

func (s *service) AddUserPrompt(ctx context.Context, req models.AddUserPromptRequest) (*models.PuzzleDetails, error) {
	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		return nil, &customErrors.EMailIsInvalid{Err: err}
	}

	if !utils.IsValidULID(req.PuzzleID) {
		return nil, &customErrors.PuzzleIDIsInvalid{Err: fmt.Errorf("invalid puzzle id: %q", req.PuzzleID)}
	}

	userPrompt := strings.TrimSpace(req.UserPrompt)
	if userPrompt == "" {
		return nil, &customErrors.UserPromptIsInvalid{Err: errors.New("user prompt is empty")}
	}

	puzzle, err := s.getOwnedPuzzle(ctx, req.PuzzleID, email)
	if err != nil {
		return nil, err
	}

	if puzzle.Metadata.UserWinStatus {
		return nil, &customErrors.PuzzleAlreadyWon{Err: errors.New("puzzle already won")}
	}

	puzzle, err = s.appendMessageWithRetry(ctx, puzzle, puzzlesRepo.UserRole, userPrompt)
	if err != nil {
		return nil, err
	}

	metrics, err := s.intentService.GetSimilarityMetrics(ctx, puzzle.HiddenPrompt, userPrompt, puzzle.HiddenPromptEmbedding)
	if err != nil {
		return nil, err
	}

	puzzle, err = s.applyGuessOutcome(ctx, puzzle, userPrompt, metrics)
	if err != nil {
		return nil, err
	}

	s.sortMessagesByTimestamp(puzzle.Messages.MessageList)

	return s.toPuzzleDetails(puzzle, &models.GuessMetrics{
		EmbeddingDotProduct:   metrics.EmbeddingDotProduct,
		CosineSimilarityScore: metrics.CosineSimilarityScore,
		LLMSimilarityScore:    metrics.LLMSimilarityScore,
		FinalSimilarityScore:  metrics.FinalSimilarityScore,
		ConsideredWon:         metrics.ConsideredWon,
	}), nil
}

// applyGuessOutcome persists the consequences of a freshly-scored guess: a
// higher max-similarity score, a win, or (if not won) an appended hint.
func (s *service) applyGuessOutcome(ctx context.Context, puzzle *puzzlesRepo.Puzzle, guess string, metrics *intent.GetSimilarityMetricsResponse) (*puzzlesRepo.Puzzle, error) {
	// Captured before applyMaxSimilarityIncrease overwrites it, so the hint
	// prompt can compare this guess's score against the player's prior best
	// (warmer/colder framing) rather than judging it in isolation.
	previousBest := float64(puzzle.Metadata.MaxIntentSimilarityPercentage)

	puzzle, err := s.applyMaxSimilarityIncrease(ctx, puzzle, metrics.FinalSimilarityScore)
	if err != nil {
		return nil, err
	}

	if metrics.ConsideredWon {
		return s.markUserWonWithRetry(ctx, puzzle)
	}

	hint, err := s.generateHint(ctx, puzzle, guess, metrics.FinalSimilarityScore, previousBest, metrics.Gap)
	if err != nil {
		return nil, err
	}

	return s.appendMessageWithRetry(ctx, puzzle, puzzlesRepo.AssistantRole, hint)
}

// maxConcurrentModificationRetries bounds how many times a write is
// retried against a freshly re-fetched puzzle after losing an optimistic
// concurrency check (puzzlesRepo.ErrConcurrentModification) - matches the
// existing bounded-retry convention used elsewhere in this codebase (e.g.
// generatePuzzleRetryAttempts). The common, uncontended case never retries
// at all; only genuine write contention on the same puzzle pays this cost.
const maxConcurrentModificationRetries = 3

// applyMaxSimilarityIncrease persists finalScore as the puzzle's new
// max-similarity if it's higher than what's already stored. A concurrent
// guess raising the score first is treated as benign, not an error. If a
// concurrent write lands between this method's read and its write, it
// re-fetches the puzzle and re-evaluates against the fresh data rather
// than blindly overwriting - see puzzlesRepo.ErrConcurrentModification.
func (s *service) applyMaxSimilarityIncrease(ctx context.Context, puzzle *puzzlesRepo.Puzzle, finalScore float64) (*puzzlesRepo.Puzzle, error) {
	newScore := int(math.Round(finalScore))

	for range maxConcurrentModificationRetries {
		if newScore <= puzzle.Metadata.MaxIntentSimilarityPercentage {
			return puzzle, nil
		}

		updated, err := s.db.PuzzlesRepository.UpdateMaxIntentSimilarityPercentage(ctx, puzzle.PuzzleID, puzzle.Metadata, puzzle.UpdatedAt, newScore)
		switch {
		case err == nil:
			return updated, nil
		case errors.Is(err, puzzlesRepo.ErrSimilarityPercentageNotIncreasing):
			return puzzle, nil
		case errors.Is(err, puzzlesRepo.ErrConcurrentModification):
			puzzle, err = s.db.PuzzlesRepository.GetPuzzleByID(ctx, puzzle.PuzzleID)
			if err != nil {
				return nil, err
			}
		default:
			return nil, err
		}
	}

	return nil, fmt.Errorf("puzzle %q: %w after %d attempts", puzzle.PuzzleID, puzzlesRepo.ErrConcurrentModification, maxConcurrentModificationRetries)
}

// appendMessageWithRetry appends message to puzzle's message list, retrying
// against a freshly re-fetched puzzle if a concurrent write lands first -
// see puzzlesRepo.ErrConcurrentModification. Shared by AddUserPrompt's own
// guess append and applyGuessOutcome's hint append.
func (s *service) appendMessageWithRetry(ctx context.Context, puzzle *puzzlesRepo.Puzzle, role puzzlesRepo.MessageRole, message string) (*puzzlesRepo.Puzzle, error) {
	for range maxConcurrentModificationRetries {
		updated, err := s.db.PuzzlesRepository.AppendPuzzleMessage(ctx, puzzle.PuzzleID, puzzle.Messages, puzzle.UpdatedAt, role, message)
		if err == nil {
			return updated, nil
		}
		if !errors.Is(err, puzzlesRepo.ErrConcurrentModification) {
			return nil, err
		}

		puzzle, err = s.db.PuzzlesRepository.GetPuzzleByID(ctx, puzzle.PuzzleID)
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("puzzle %q: %w after %d attempts", puzzle.PuzzleID, puzzlesRepo.ErrConcurrentModification, maxConcurrentModificationRetries)
}

// markUserWonWithRetry marks puzzle won, retrying against a freshly
// re-fetched puzzle if a concurrent write lands first - see
// puzzlesRepo.ErrConcurrentModification. A concurrent write that already
// marked the puzzle won is handled by MarkUserWon's own "already won"
// no-op check once retried against the fresh data, not treated as an error.
func (s *service) markUserWonWithRetry(ctx context.Context, puzzle *puzzlesRepo.Puzzle) (*puzzlesRepo.Puzzle, error) {
	for range maxConcurrentModificationRetries {
		updated, err := s.db.PuzzlesRepository.MarkUserWon(ctx, puzzle.PuzzleID, *puzzle)
		if err == nil {
			return updated, nil
		}
		if !errors.Is(err, puzzlesRepo.ErrConcurrentModification) {
			return nil, err
		}

		puzzle, err = s.db.PuzzlesRepository.GetPuzzleByID(ctx, puzzle.PuzzleID)
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("puzzle %q: %w after %d attempts", puzzle.PuzzleID, puzzlesRepo.ErrConcurrentModification, maxConcurrentModificationRetries)
}

// generateHint asks the AI layer for a hint toward the given puzzle,
// calibrated against the requesting user's overall play history and this
// guess's actual closeness (currentScore) relative to their prior best
// (previousBest) on this puzzle, so the hint can acknowledge whether the
// player is getting warmer or colder instead of judging the guess blind.
// llmGap is ScoreIntent's own short explanation of what this guess's
// wording is missing (from the same LLM call that already scored this
// guess) - reused here so the hint model doesn't have to re-derive the
// gap from scratch. When empty (ScoreIntent wasn't called at all, or the
// model omitted it), a deterministic lexical fallback stands in instead -
// either way the hint model gets a concrete concept to ground its nudge
// in rather than reasoning about two raw text blocks unassisted.
func (s *service) generateHint(ctx context.Context, puzzle *puzzlesRepo.Puzzle, guess string, currentScore float64, previousBest float64, llmGap string) (string, error) {
	puzzles, err := s.db.PuzzlesRepository.GetPuzzlesByEmail(ctx, puzzle.UserEmail)
	if err != nil {
		return "", err
	}

	gapHint := llmGap
	if gapHint == "" {
		gapHint = strings.Join(utils.MissingContentWords(puzzle.HiddenPrompt, guess, 4), ", ")
	}

	resp, err := s.ai.GenerateHint(ctx, chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt:      puzzle.HiddenPrompt,
		CanonicalAnswer:   puzzle.CanonicalAnswer,
		History:           s.toChatHistory(puzzle.Messages.MessageList),
		Guess:             guess,
		CurrentScore:      currentScore,
		PreviousBestScore: previousBest,
		UserContext:       s.buildUserContext(puzzles),
		GapHint:           gapHint,
	})
	if err != nil {
		return "", err
	}

	return resp.Hint, nil
}

// getOwnedPuzzle fetches a puzzle by ID and verifies it belongs to email.
// A puzzle that doesn't exist and a puzzle that exists but belongs to a
// different user report the identical error, so a caller can't use this
// API to enumerate other users' valid puzzle IDs.
func (s *service) getOwnedPuzzle(ctx context.Context, puzzleID string, email string) (*puzzlesRepo.Puzzle, error) {
	puzzle, err := s.db.PuzzlesRepository.GetPuzzleByID(ctx, puzzleID)
	if err != nil {
		if errors.Is(err, puzzlesRepo.ErrNoRows) {
			return nil, &customErrors.PuzzleNotFound{Err: err}
		}
		return nil, err
	}

	if puzzle.UserEmail != email {
		return nil, &customErrors.PuzzleNotFound{Err: fmt.Errorf("puzzle %q not found for this user", puzzleID)}
	}

	return puzzle, nil
}

// buildUserContext derives difficulty-calibration signals from a user's
// past puzzles for the AI layer's UserContext. TotalGames/GamesWon are
// computed from the full lifetime list (general player-experience
// context). PreviousPuzzlesMaxIntentSimilarityPercentage and
// RecentGamesWon are deliberately scoped to only the most recent
// maxRecentPuzzlesForContext puzzles instead - deriveDifficultyTier uses
// that recency-scoped pair to pick a tier, so a player's difficulty
// tracks their current form rather than staying anchored by a permanent
// lifetime average that a rough start (or a long lucky streak) could keep
// dragging in one direction indefinitely.
func (s *service) buildUserContext(puzzles []puzzlesRepo.Puzzle) chatAndAIUsage.UserContext {
	var gamesWon uint
	for _, p := range puzzles {
		if p.Metadata.UserWinStatus {
			gamesWon++
		}
	}

	recent := s.mostRecentPuzzles(puzzles, maxRecentPuzzlesForContext)
	similarities := make([]uint, len(recent))
	var recentGamesWon uint
	for i, p := range recent {
		similarities[i] = uint(p.Metadata.MaxIntentSimilarityPercentage)
		if p.Metadata.UserWinStatus {
			recentGamesWon++
		}
	}

	return chatAndAIUsage.UserContext{
		PreviousPuzzlesMaxIntentSimilarityPercentage: similarities,
		TotalGames:     uint(len(puzzles)),
		GamesWon:       gamesWon,
		RecentGamesWon: recentGamesWon,
	}
}

// maxRecentPuzzlesForContext bounds how many of this user's most recent
// puzzles feed two related concerns: what GeneratePuzzle sees as "don't
// repeat these" context (see recentPuzzles), and what buildUserContext
// uses to calibrate the current difficulty tier. Bounded rather than the
// user's whole history (which could grow unbounded over a long-playing
// account) while still giving enough recent signal for both purposes.
const maxRecentPuzzlesForContext = 15

// mostRecentPuzzles returns up to the most recent limit puzzles from
// puzzles, newest first - the shared sort-and-bound step behind both
// recentPuzzles and buildUserContext, so the two concerns stay looking at
// the same window by construction instead of two independently-bounded
// slices that could drift apart.
func (s *service) mostRecentPuzzles(puzzles []puzzlesRepo.Puzzle, limit int) []puzzlesRepo.Puzzle {
	sorted := make([]puzzlesRepo.Puzzle, len(puzzles))
	copy(sorted, puzzles)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

// recentPuzzles returns the most recent maxRecentPuzzlesForContext
// puzzles (question + answer), newest first - reuses the puzzle list
// CreatePuzzleForUser already fetched for buildUserContext, no extra DB
// query needed. Carries the answer alongside the question so generation
// can avoid reusing a past answer even under a brand-new question - a
// repeated answer is guessable from memory alone regardless of how the
// question is worded.
func (s *service) recentPuzzles(puzzles []puzzlesRepo.Puzzle) []chatAndAIUsage.RecentPuzzle {
	sorted := s.mostRecentPuzzles(puzzles, maxRecentPuzzlesForContext)

	recent := make([]chatAndAIUsage.RecentPuzzle, len(sorted))
	for i, p := range sorted {
		recent[i] = chatAndAIUsage.RecentPuzzle{Question: p.HiddenPrompt, Answer: p.CanonicalAnswer}
	}
	return recent
}

// maxUsedAnswersForGeneration safety-caps allTimeCanonicalAnswers' output
// against a pathologically long-playing account - not a real design
// target (bare answers are only a few words each, so this stays
// token-light well past what any realistic account will ever reach), just
// a bound against unbounded growth.
const maxUsedAnswersForGeneration = 300

// allTimeCanonicalAnswers returns every distinct canonical answer this
// player has ever been given, newest first and deduplicated
// case-insensitively - deliberately not bounded to the same recent
// window as recentPuzzles/buildUserContext. A repeated answer is
// guessable from memory alone regardless of how long ago or under what
// question it last appeared, so this list exists to guard against reuse
// across the player's entire history, not just their recent one. No
// extra DB query needed - reuses the same already-fetched puzzles slice
// CreatePuzzleForUser already has.
func (s *service) allTimeCanonicalAnswers(puzzles []puzzlesRepo.Puzzle) []string {
	sorted := s.mostRecentPuzzles(puzzles, len(puzzles))

	seen := make(map[string]struct{}, len(sorted))
	answers := make([]string, 0, len(sorted))
	for _, p := range sorted {
		key := strings.ToLower(strings.TrimSpace(p.CanonicalAnswer))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		answers = append(answers, p.CanonicalAnswer)
		if len(answers) >= maxUsedAnswersForGeneration {
			break
		}
	}
	return answers
}

func (s *service) toChatHistory(messages []puzzlesRepo.MessageElement) []chatAndAIUsage.ChatMessage {
	history := make([]chatAndAIUsage.ChatMessage, len(messages))
	for i, m := range messages {
		role := chatAndAIUsage.UserRole
		if m.Role == puzzlesRepo.AssistantRole {
			role = chatAndAIUsage.AssistantRole
		}
		history[i] = chatAndAIUsage.ChatMessage{
			Role:    role,
			Message: m.Message,
		}
	}
	return history
}

func (s *service) toMessageElements(messages []puzzlesRepo.MessageElement) []models.MessageElement {
	elements := make([]models.MessageElement, len(messages))
	for i, m := range messages {
		role := models.UserRole
		if m.Role == puzzlesRepo.AssistantRole {
			role = models.AssistantRole
		}
		elements[i] = models.MessageElement{
			Role:      role,
			Message:   m.Message,
			Timestamp: m.Timestamp,
		}
	}
	return elements
}

// toPuzzleDetails maps the repo's Puzzle into the public PuzzleDetails
// model. HiddenPrompt is only populated once the puzzle is won - it's the
// secret the player is guessing, unlike CanonicalAnswer (the game's clue,
// always visible). latest is nil for a plain read and populated with the
// just-scored guess's breakdown when called from AddUserPrompt.
func (s *service) toPuzzleDetails(puzzle *puzzlesRepo.Puzzle, latest *models.GuessMetrics) *models.PuzzleDetails {
	details := &models.PuzzleDetails{
		PuzzleID:                      puzzle.PuzzleID,
		CanonicalAnswer:               puzzle.CanonicalAnswer,
		Messages:                      s.toMessageElements(puzzle.Messages.MessageList),
		MaxIntentSimilarityPercentage: puzzle.Metadata.MaxIntentSimilarityPercentage,
		UserWinStatus:                 puzzle.Metadata.UserWinStatus,
		LatestGuessMetrics:            latest,
		CreatedAt:                     puzzle.CreatedAt,
		UpdatedAt:                     puzzle.UpdatedAt,
	}

	if puzzle.Metadata.UserWinStatus {
		details.HiddenPrompt = puzzle.HiddenPrompt
	}

	return details
}

func (s *service) sortMessagesByTimestamp(messages []puzzlesRepo.MessageElement) {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
}
