package chatAndAIUsage

import (
	"context"
	"errors"
	"fmt"
	"hidden-prompt-backend/pkg/chat/internal/prompts"
	"hidden-prompt-backend/pkg/chat/templates"
	geminiClient "hidden-prompt-backend/pkg/clients/gemini"
	groqClient "hidden-prompt-backend/pkg/clients/groq"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/serialization"
	"hidden-prompt-backend/pkg/utils"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/samber/lo"
)

var ErrEmptyLLMResponse = errors.New("llm returned empty response")

const (
	// maxHintHistoryMessages caps the number of most-recent chat turns ever
	// considered for the hint prompt, regardless of their length.
	maxHintHistoryMessages = 12
	// maxHintHistoryChars bounds the serialized history text length. ~4
	// characters per token is a standard heuristic for English text, so this
	// keeps history comfortably under ~500 tokens, well inside free-tier
	// model context limits alongside the rest of the hint prompt.
	maxHintHistoryChars = 2000

	// scoreIntentJSONRetryAttempts bounds how many times ScoreIntent
	// re-samples the model when it returns something ExtractJSON can't
	// parse - weak/high-temperature model picks occasionally ignore "JSON
	// only" instructions, and a fresh re-sample often comes back clean.
	scoreIntentJSONRetryAttempts = 3
	// scoreIntentRetryBaseDelay is the base for exponential backoff between
	// retry attempts, so a run of bad responses doesn't hammer the model
	// back-to-back.
	scoreIntentRetryBaseDelay = 200 * time.Millisecond

	// hintMaxTemperature caps hint-generation calls to the low end of the
	// shared client's temperature range (which otherwise randomizes up to
	// 2.0 for every call type). A one-sentence, tightly-constrained hint
	// task needs much more consistency than "generate a fun trivia
	// question" - a high-temperature roll was a direct contributor to
	// garbled/overly-baroque hint phrasing.
	hintMaxTemperature = 1.1

	// scoreIntentMaxTemperature caps guess-scoring calls the same way
	// hintMaxTemperature caps hints. A JSON scoring/classification task
	// needs consistency, not creative range - the default wide temperature
	// band (up to 2.0) was a direct contributor to a guess scoring
	// noticeably lower than an effectively-identical one just because it
	// added a harmless clarifying word. Narrowed further to 0.85 (from an
	// earlier 1.0): scoring has even less use for creative range than
	// generation does, and the tighter the sampling band, the less two
	// near-identical guesses can drift apart in score purely from
	// temperature noise - directly relevant right around winThreshold,
	// where that drift can flip a win into a loss.
	scoreIntentMaxTemperature = 0.85

	// hintRetryAttempts bounds how many times GenerateHint re-samples when
	// a response looks like a refusal/non-hint rather than a real error -
	// observed in practice: some models in the shared weighted pool
	// occasionally produce a boilerplate "I can't help with that"-style
	// reply despite the system prompt's instructions, and a fresh re-sample
	// (often landing on a different model) reliably produces a real hint.
	hintRetryAttempts       = 3
	hintRetryBaseDelay      = 200 * time.Millisecond
	hintMaxWordsForRealHint = 40

	// generatePuzzleRetryAttempts guards against a model in the shared pool
	// ignoring the "JSON only" instruction, or against either JSON field
	// looking malformed (observed in practice for the old, separate calls
	// this replaced: a multi-paragraph "**Answer:** ... **Reasoning:**
	// ..." block instead of a short question, or a multi-sentence
	// explanation instead of a terse answer) - a malformed puzzle poisons
	// its entire lifetime, so this fails loudly after retries exhaust
	// rather than silently accepting whatever came back.
	generatePuzzleRetryAttempts    = 3
	generatePuzzleRetryBaseDelay   = 200 * time.Millisecond
	generatePuzzleQuestionMaxWords = 40
	generatePuzzleAnswerMaxWords   = 12

	// generatePuzzleMaxTemperature caps puzzle-generation calls the same
	// way hintMaxTemperature caps hints. The question half benefits from
	// some creative range, but the answer half must be factually correct
	// for the question in the same response - pairing a wide-open
	// temperature (up to 2.0) with a from-scratch factual answer was a
	// direct contributor to a wrong answer for an otherwise-unambiguous
	// question (e.g. "Parrot" generated for "What is the only bird that
	// can fly backwards?", where the correct answer is "Hummingbird").
	generatePuzzleMaxTemperature = 1.0

	// hardTierSkillThreshold and easyTierSkillThreshold gate
	// deriveDifficultyTier's blended skillScore (0-100). Symmetric around
	// the 50 midpoint by design - see that method's doc comment for why.
	hardTierSkillThreshold = 65.0
	easyTierSkillThreshold = 35.0
)

var (
	ErrAllAttemptsMalformed = errors.New("chat: all retry attempts returned a malformed response")
	modelsToAvoid           = []string{constants.Llama318B, constants.GroqCompound}
)

type AI interface {
	GenerateEmbeddings(ctx context.Context, dataString string) ([]float32, error)
	GeneratePuzzle(ctx context.Context, req GeneratePuzzleRequest) (*GeneratePuzzleResponse, error)
	GenerateHint(ctx context.Context, req GenerateHintRequest) (*GenerateHintResponse, error)
	ScoreIntent(ctx context.Context, req ScoreIntentRequest) (*ScoreIntentResponse, error)
}

type ai struct {
	groqClient   groqClient.Client
	geminiClient geminiClient.Client
	puzzleStyles utils.ThreadSafeWeightedMap[puzzleStyle]
}

func NewAI(groqCli groqClient.Client, geminiCli geminiClient.Client) (AI, error) {
	if groqCli == nil {
		return nil, fmt.Errorf("groqClient is nil")
	}
	if geminiCli == nil {
		return nil, fmt.Errorf("geminiClient is nil")
	}
	puzzleStyles, err := newPuzzleStyleMap()
	if err != nil {
		return nil, fmt.Errorf("build puzzle style map: %w", err)
	}
	return &ai{groqClient: groqCli, geminiClient: geminiCli, puzzleStyles: puzzleStyles}, nil
}

func (a *ai) GenerateEmbeddings(ctx context.Context, dataString string) ([]float32, error) {
	dataString = utils.NormalizeString(dataString)
	return a.geminiClient.GetEmbeddings(ctx, dataString)
}

// generatePuzzleLLMResponse is the raw JSON shape the model is asked to
// return - kept separate from the exported GeneratePuzzleResponse (which
// uses this package's own hidden_prompt/canonical_answer naming) so the
// prompt's wire contract can evolve independently of the public API shape.
type generatePuzzleLLMResponse struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// formatRecentPuzzles renders each recent puzzle as one "Q -> A" line for
// the generation prompt - both halves matter here: the model needs to
// avoid repeating a past question AND avoid reusing a past answer under a
// new question, so both are shown together rather than just the question.
func formatRecentPuzzles(recent []RecentPuzzle) []string {
	lines := make([]string, len(recent))
	for i, p := range recent {
		lines[i] = fmt.Sprintf("Q: %s -> A: %s", p.Question, p.Answer)
	}
	return lines
}

// choosePuzzleStyle assigns a content domain for this generation call
// instead of leaving format/topic entirely to the model's own judgment -
// see puzzle_style.go's doc comment for why that was unreliable in
// practice. A plain stateless weighted pick, deliberately not tracking
// "the last domain assigned" - that would be per-process state, which
// doesn't hold as a real guarantee across multiple server instances in a
// distributed deployment, and an equal-weight pick already makes any
// particular domain repeating several times in a row vanishingly
// unlikely on its own.
func (a *ai) choosePuzzleStyle() (style puzzleStyle, angles string) {
	style = a.puzzleStyles.GetRandom()
	return style, puzzleStyleAngles[style]
}

func (a *ai) GeneratePuzzle(ctx context.Context, req GeneratePuzzleRequest) (*GeneratePuzzleResponse, error) {
	tier := a.deriveDifficultyTier(req.UserContext)
	style, styleAngles := a.choosePuzzleStyle()
	userPrompt := prompts.GeneratePuzzleUser(
		tier, req.UserContext.TotalGames, string(style), styleAngles,
		formatRecentPuzzles(req.RecentPuzzles), req.UsedAnswers,
	)

	var lastErr error
	for attempt := range generatePuzzleRetryAttempts {
		if attempt > 0 {
			if err := a.sleepBackoff(ctx, generatePuzzleRetryBaseDelay, attempt); err != nil {
				return nil, err
			}
		}

		resp, err := a.groqClient.Call(ctx, groqClient.CallRequest{
			SystemPrompt:   templates.GeneratePuzzleSystemPrompt(),
			UserPrompt:     userPrompt,
			MaxTemperature: lo.ToPtr(generatePuzzleMaxTemperature),
			ModelsToAvoid:  modelsToAvoid,
			ModelToUse:     utils.ChooseOneRandomly(lo.ToPtr(constants.GPTOSS20B), lo.ToPtr(constants.Llama3370B)),
		})
		if err != nil {
			lastErr = err
			continue
		}
		if len(resp.Response) == 0 {
			lastErr = ErrEmptyLLMResponse
			continue
		}

		parsed, err := utils.ExtractJSON[generatePuzzleLLMResponse](resp.Response[0])
		if err != nil {
			lastErr = err
			continue
		}

		question := strings.TrimSpace(parsed.Question)
		answer := a.normalizeCanonicalAnswer(parsed.Answer)
		if a.isMalformedGeneratedQuestion(question) || a.isMalformedGeneratedAnswer(answer) {
			lastErr = fmt.Errorf("generated puzzle looked malformed: question=%q answer=%q", question, answer)
			continue
		}

		return &GeneratePuzzleResponse{HiddenPrompt: question, CanonicalAnswer: answer}, nil
	}

	slog.Error("GeneratePuzzle failed after all retry attempts", slog.Int("attempts", generatePuzzleRetryAttempts), slog.Any("error", lastErr))
	return nil, fmt.Errorf("generate puzzle: after %d attempt(s): %w: %w", generatePuzzleRetryAttempts, ErrAllAttemptsMalformed, lastErr)
}

// normalizeCanonicalAnswer strips trailing sentence punctuation as a
// deterministic backstop regardless of whether the model followed the
// system prompt's "no trailing punctuation" rule - observed in practice
// across the two calls this replaced: some model draws answered "Blue."
// while others answered "Jupiter" for the identically-styled instruction,
// an inconsistency worth normalizing in code rather than trusting every
// model in the shared pool to comply on every draw.
func (a *ai) normalizeCanonicalAnswer(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ".!?")
}

// isMalformedGeneratedQuestion catches the observed failure shape from the
// call this replaced: a full reasoning/answer dump ("**Answer:** ...
// **Reasoning:** ...") instead of the single short question/task asked for.
func (a *ai) isMalformedGeneratedQuestion(text string) bool {
	if text == "" {
		return true
	}
	if len(strings.Fields(text)) > generatePuzzleQuestionMaxWords {
		return true
	}
	if strings.Contains(text, "\n") {
		return true
	}
	if strings.Contains(text, "**") {
		return true
	}
	return false
}

// isMalformedGeneratedAnswer catches the same failure shape: a real
// "fewest words possible" answer is a handful of words on one line, never
// a multi-line explanation.
func (a *ai) isMalformedGeneratedAnswer(text string) bool {
	if text == "" {
		return true
	}
	if len(strings.Fields(text)) > generatePuzzleAnswerMaxWords {
		return true
	}
	if strings.Contains(text, "\n") {
		return true
	}
	return false
}

// sleepBackoff waits the exponential-backoff delay for the given attempt
// number, or returns ctx's error if it's cancelled first - shared by every
// retry loop in this file so the backoff shape stays consistent.
func (a *ai) sleepBackoff(ctx context.Context, base time.Duration, attempt int) error {
	delay := base * time.Duration(1<<(attempt-1))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func (a *ai) GenerateHint(ctx context.Context, req GenerateHintRequest) (*GenerateHintResponse, error) {
	tier := a.deriveDifficultyTier(req.UserContext)

	history, historyLimit, err := a.trimHintHistory(req.History)
	if err != nil {
		return nil, err
	}

	currentScore := a.clampIntentScore(int(math.Round(req.CurrentScore)))
	previousBest := a.clampIntentScore(int(math.Round(req.PreviousBestScore)))

	userPrompt := prompts.HintUser(
		req.HiddenPrompt, req.CanonicalAnswer, history, req.Guess, tier,
		currentScore, previousBest, historyLimit, req.GapHint,
	)

	var lastHint string
	var lastErr error
	for attempt := range hintRetryAttempts {
		if attempt > 0 {
			if err := a.sleepBackoff(ctx, hintRetryBaseDelay, attempt); err != nil {
				return nil, err
			}
		}

		resp, err := a.groqClient.Call(ctx, groqClient.CallRequest{
			SystemPrompt:   templates.HintSystemPrompt(),
			UserPrompt:     userPrompt,
			MaxTemperature: lo.ToPtr(hintMaxTemperature),
		})
		if err != nil {
			lastErr = err
			continue
		}
		if len(resp.Response) == 0 {
			lastErr = ErrEmptyLLMResponse
			continue
		}

		hint := strings.TrimSpace(resp.Response[0])
		if a.revealsSecret(hint, req.CanonicalAnswer, req.Guess) {
			lastHint = hint
			lastErr = fmt.Errorf("hint revealed the canonical answer: %q", hint)
			continue
		}
		if !a.isUnusableHint(hint) {
			return &GenerateHintResponse{Hint: hint}, nil
		}
		lastHint = hint
		lastErr = fmt.Errorf("hint looked like a refusal: %q", hint)
	}

	// A leaked answer must never reach the player, even as a last resort -
	// unlike a refusal (supplementary, safe to surface stale), returning a
	// leaked hint would break the game outright. Fall back to a generic,
	// always-safe nudge instead.
	if lastHint != "" && a.revealsSecret(lastHint, req.CanonicalAnswer, req.Guess) {
		slog.Error("GenerateHint: every retry attempt leaked the canonical answer, falling back to a generic hint", slog.Int("attempts", hintRetryAttempts))
		return &GenerateHintResponse{Hint: "Getting warmer - think about what makes this one specific."}, nil
	}

	// Every attempt looked like a refusal (or errored) - a hint is
	// supplementary, not core to the guess itself, so this must never fail
	// the whole guess submission. But surfacing the raw refusal text
	// itself ("I'm sorry, but I can't provide that.") to the player breaks
	// immersion mid-game - a generic nudge is a better fallback than the
	// model's literal refusal, same reasoning as the leak fallback above.
	if lastHint != "" {
		slog.Warn("GenerateHint: all retry attempts looked like refusals, using a generic hint instead", slog.Int("attempts", hintRetryAttempts))
		return &GenerateHintResponse{Hint: "About the same as last time - think it through and try a different angle."}, nil
	}

	slog.Error("GenerateHint failed after all retry attempts", slog.Int("attempts", hintRetryAttempts), slog.Any("error", lastErr))
	return nil, fmt.Errorf("generate hint: after %d attempt(s): %w", hintRetryAttempts, lastErr)
}

// isUnusableHint is a defensive heuristic, not a content filter: it only
// catches the boilerplate refusal phrasing observed in practice from
// certain models in the shared pool, plus responses too long to plausibly
// be the required "one short sentence" hint. Answer-leak detection is a
// separate, stricter check - see revealsSecret.
func (a *ai) isUnusableHint(hint string) bool {
	if hint == "" {
		return true
	}
	if len(strings.Fields(hint)) > hintMaxWordsForRealHint {
		return true
	}

	// Normalize typographic/smart apostrophes ('/'/‘) to a plain ASCII
	// quote before matching - models in the shared pool frequently render
	// refusals with a curly apostrophe ("I'm sorry"), which a
	// straight-quote-only phrase list silently fails to catch, letting the
	// raw refusal text through as if it were a real hint.
	lower := strings.NewReplacer("’", "'", "‘", "'").Replace(strings.ToLower(hint))
	refusalPhrases := []string{
		"i'm sorry", "i am sorry", "i cannot", "i can't", "i can not",
		"as an ai", "i'm not able", "i am not able", "i won't", "i will not",
	}
	for _, phrase := range refusalPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// revealsSecret catches the one failure mode isUnusableHint can't: a hint
// that plainly states the canonical answer instead of nudging toward it.
// The system prompt already instructs the model never to do this, but that
// instruction alone let a real hint of just "Apple" through once (the
// canonical answer, verbatim) - this is the output-side backstop.
//
// Matching is whole-word, not raw substring containment: a canonical
// answer as short as "a" would otherwise match inside ordinary words like
// "warmer", flagging harmless hints as leaks. Very short answers (<=2
// normalized characters) are skipped entirely - too common as substrings
// of unrelated words to check reliably, and real canonical answers in this
// game are always full words.
//
// guess is the player's own current guess: if they already typed the
// answer word themselves (e.g. guessing "Tell me about Jupiter" for a
// puzzle whose answer is "Jupiter"), a hint that merely references that
// same word back isn't disclosing anything new. But explicit confirmation
// phrasing ("The answer is Jupiter.") is a leak regardless of what the
// guess said - it hands over certainty the player hadn't earned, even if
// they'd already named the right word themselves. Caught via the SLT
// suite: a guess containing "Piano" got exempted by word-presence alone,
// letting an actual "The answer is piano." hint straight through.
func (a *ai) revealsSecret(hint string, canonicalAnswer string, guess string) bool {
	normalizedHint := utils.NormalizeString(hint)

	confirmationPhrases := []string{"the answer is", "answer is", "is the answer"}
	for _, phrase := range confirmationPhrases {
		if strings.Contains(normalizedHint, phrase) {
			return true
		}
	}

	normalizedAnswer := utils.NormalizeString(canonicalAnswer)
	if len(normalizedAnswer) <= 2 {
		return false
	}

	paddedGuess := " " + utils.NormalizeString(guess) + " "
	if strings.Contains(paddedGuess, " "+normalizedAnswer+" ") {
		return false
	}

	paddedHint := " " + normalizedHint + " "
	return strings.Contains(paddedHint, " "+normalizedAnswer+" ")
}

func (a *ai) ScoreIntent(ctx context.Context, req ScoreIntentRequest) (*ScoreIntentResponse, error) {
	userPrompt := prompts.ScoreIntentUser(req.TargetIntent, req.GivenIntent)

	var lastErr error
	for attempt := range scoreIntentJSONRetryAttempts {
		if attempt > 0 {
			delay := scoreIntentRetryBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := a.groqClient.Call(ctx, groqClient.CallRequest{
			SystemPrompt:   templates.ScoreIntentSystemPrompt(),
			UserPrompt:     userPrompt,
			MaxTemperature: lo.ToPtr(scoreIntentMaxTemperature),
			// Without this, chooseModel falls through to the full shared
			// pool, where Llama318B alone carries 65% weight - the exact
			// model GeneratePuzzle already avoids via this same var, for
			// the same "needs more consistency than the shared pool gives"
			// reason. A guess's win/loss depends on this score, so it
			// deserves at least the same model-quality floor generation
			// already gets.
			ModelsToAvoid: modelsToAvoid,
		})
		if err != nil {
			lastErr = err
			continue
		}
		if len(resp.Response) == 0 {
			lastErr = ErrEmptyLLMResponse
			continue
		}

		parsed, err := utils.ExtractJSON[ScoreIntentResponse](resp.Response[0])
		if err != nil {
			lastErr = err
			continue
		}

		parsed.IntentScore = a.clampIntentScore(parsed.IntentScore)
		return &parsed, nil
	}

	slog.Error("ScoreIntent failed after all retry attempts", slog.Int("attempts", scoreIntentJSONRetryAttempts), slog.Any("error", lastErr))
	return nil, fmt.Errorf("score intent: after %d attempt(s): %w", scoreIntentJSONRetryAttempts, lastErr)
}

func (a *ai) clampIntentScore(score int) int {
	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}

// trimHintHistory returns the serialized form of the most recent slice of
// history that fits within the token-overflow guards, plus how many
// messages were included. includedCount is 0 when the full history was sent
// untruncated (nothing to caveat in the prompt), and the actual kept count
// when truncation happened, so the caller can tell the model it's only
// seeing the last N turns instead of silently truncating.
func (a *ai) trimHintHistory(history []ChatMessage) (serializedHistory string, includedCount uint, err error) {
	trimmed := history
	if len(trimmed) > maxHintHistoryMessages {
		trimmed = trimmed[len(trimmed)-maxHintHistoryMessages:]
	}

	serialized, err := serialization.Serialize(trimmed)
	if err != nil {
		return "", 0, err
	}

	for len(trimmed) > 0 && len(serialized) > maxHintHistoryChars {
		trimmed = trimmed[1:]
		serialized, err = serialization.Serialize(trimmed)
		if err != nil {
			return "", 0, err
		}
	}

	if len(trimmed) == len(history) {
		return serialized, 0, nil
	}
	return serialized, uint(len(trimmed)), nil
}

// deriveDifficultyTier picks a difficulty tier from userCtx's
// recency-scoped signals (PreviousPuzzlesMaxIntentSimilarityPercentage /
// RecentGamesWon - see UserContext's doc comment for why these, not the
// lifetime totals, drive this decision). The two signals are blended into
// a single 0-100 skillScore rather than gated independently: an earlier
// version required winRate>=0.7 AND avgSimilarity>=70 for "hard" but only
// winRate<=0.3 OR avgSimilarity<=30 for "easy" - asymmetric on purpose in
// neither direction, just an accident of how the conditions were written,
// and the result was a tier that fell to "easy" easily but climbed back
// to "hard" only when both signals recovered simultaneously. A single
// blended score computed identically in both directions removes that
// asymmetry: hardTierSkillThreshold and easyTierSkillThreshold sit
// equidistant from the 50 midpoint, so climbing and falling take equally
// little.
func (a *ai) deriveDifficultyTier(userCtx UserContext) string {
	if userCtx.TotalGames == 0 {
		return "easy"
	}

	recentGamesPlayed := len(userCtx.PreviousPuzzlesMaxIntentSimilarityPercentage)
	if recentGamesPlayed == 0 {
		return "easy"
	}

	recentWinRate := float64(userCtx.RecentGamesWon) / float64(recentGamesPlayed)

	var similaritySum uint
	for _, v := range userCtx.PreviousPuzzlesMaxIntentSimilarityPercentage {
		similaritySum += v
	}
	avgSimilarity := float64(similaritySum) / float64(recentGamesPlayed)

	skillScore := 0.5*recentWinRate*100 + 0.5*avgSimilarity

	switch {
	case skillScore >= hardTierSkillThreshold:
		return "hard"
	case skillScore <= easyTierSkillThreshold:
		return "easy"
	default:
		return "medium"
	}
}
