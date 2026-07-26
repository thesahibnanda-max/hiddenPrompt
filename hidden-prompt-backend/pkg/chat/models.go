package chatAndAIUsage

type ChatMessageRole string

const (
	AssistantRole ChatMessageRole = "assistant"
	UserRole      ChatMessageRole = "user"
)

type ChatMessage struct {
	Role    ChatMessageRole `json:"role"`
	Message string          `json:"message"`
}

// UserContext carries difficulty-calibration signals for a player.
// TotalGames/GamesWon are true lifetime counts (shown to the model as
// general "how experienced is this player" context). By contrast,
// PreviousPuzzlesMaxIntentSimilarityPercentage and RecentGamesWon are
// both scoped to the same bounded recent window (see puzzle.go's
// buildUserContext) - deriveDifficultyTier uses only that recency-scoped
// pair, not the lifetime totals, so a player's tier tracks their current
// form rather than being permanently anchored by how they played months
// ago.
type UserContext struct {
	PreviousPuzzlesMaxIntentSimilarityPercentage []uint `json:"previous_puzzles_max_intent_similarity_percentage"`
	TotalGames                                   uint   `json:"total_games"`
	GamesWon                                     uint   `json:"games_won"`
	// RecentGamesWon is the number of wins within the same bounded recent
	// window as PreviousPuzzlesMaxIntentSimilarityPercentage - paired with
	// that field's length (recent games played) to give
	// deriveDifficultyTier a recency-scoped win rate.
	RecentGamesWon uint `json:"recent_games_won"`
}

type RecentPuzzle struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type GeneratePuzzleRequest struct {
	UserContext UserContext `json:"user_context"`
	// RecentPuzzles is a bounded slice of this user's most recent puzzles
	// (question + answer), passed so the model can avoid both repeating a
	// question it's already asked this player AND reusing an answer it's
	// already given them under a different question - a repeated answer
	// is guessable from memory alone regardless of how the question is
	// worded. May be empty for a brand-new player.
	RecentPuzzles []RecentPuzzle `json:"recent_puzzles"`
	// UsedAnswers is every distinct canonical answer this player has ever
	// been given, deduplicated - deliberately not bounded to the same
	// recent window as RecentPuzzles. A repeated answer is guessable from
	// memory alone regardless of how long ago or under what question it
	// last appeared, so this guards against reuse for the player's entire
	// history, not just their recent one. Kept separate from
	// RecentPuzzles since bare answers are far cheaper per entry than full
	// question+answer pairs, so a much larger (or effectively unbounded)
	// list stays affordable here in a way it wouldn't for full detail. May
	// be empty for a brand-new player.
	UsedAnswers []string `json:"used_answers"`
}

type GeneratePuzzleResponse struct {
	HiddenPrompt    string `json:"hidden_prompt"`
	CanonicalAnswer string `json:"canonical_answer"`
}

type GenerateHintRequest struct {
	HiddenPrompt    string        `json:"hidden_prompt"`
	CanonicalAnswer string        `json:"canonical_answer"`
	History         []ChatMessage `json:"history"`
	Guess           string        `json:"guess"`
	// CurrentScore is this guess's blended similarity score (0-100).
	// PreviousBestScore is the player's best score on this puzzle before
	// this guess. Together they let the hint acknowledge whether the
	// player is getting warmer or colder instead of judging the guess in
	// isolation.
	CurrentScore      float64     `json:"current_score"`
	PreviousBestScore float64     `json:"previous_best_score"`
	UserContext       UserContext `json:"user_context"`
	// GapHint names the specific concept the guess's own wording doesn't
	// address - either the LLM's own Gap from ScoreIntent (this exact
	// guess was already judged once; reuse that reasoning instead of
	// asking the model to re-derive it from scratch) or, when that's
	// unavailable, a deterministic stopword-filtered word-diff fallback
	// (pkg/utils.MissingContentWords). May be empty.
	GapHint string `json:"gap_hint"`
}

type GenerateHintResponse struct {
	Hint string `json:"hint"`
}

type ScoreIntentRequest struct {
	TargetIntent string `json:"target_intent"`
	GivenIntent  string `json:"given_intent"`
}

type ScoreIntentResponse struct {
	IntentScore int `json:"intent_score"`
	// Gap is a short (3-6 word) phrase naming the single biggest
	// difference in what's being asked between GivenIntent and
	// TargetIntent - never the answer itself. Reused downstream to ground
	// hint generation without spending a second LLM call on it. May be
	// empty if the model omits it; callers must treat that as "no signal
	// available," not an error.
	Gap string `json:"gap"`
}
