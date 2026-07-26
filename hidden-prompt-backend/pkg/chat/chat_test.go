package chatAndAIUsage_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	chatAndAIUsage "hidden-prompt-backend/pkg/chat"
	"hidden-prompt-backend/pkg/chat/templates"
	geminiClient "hidden-prompt-backend/pkg/clients/gemini"
	groqClient "hidden-prompt-backend/pkg/clients/groq"

	"github.com/stretchr/testify/require"
)

// fakeGroqClient is an in-memory stand-in for groqClient.Client so tests
// never spend real (free-tier) API tokens. It records the last request it
// received and returns whatever canned response/error was configured.
type fakeGroqClient struct {
	lastReq groqClient.CallRequest
	resp    *groqClient.CallResponse
	err     error
}

func (f *fakeGroqClient) Call(_ context.Context, req groqClient.CallRequest) (*groqClient.CallResponse, error) {
	f.lastReq = req
	return f.resp, f.err
}

// sequencedGroqClient returns a different canned result on each successive
// call, for exercising ScoreIntent's retry loop. The last result repeats
// once results are exhausted.
type sequencedGroqClient struct {
	calls   int
	lastReq groqClient.CallRequest
	results []groqClientCallResult
}

type groqClientCallResult struct {
	resp *groqClient.CallResponse
	err  error
}

func (f *sequencedGroqClient) Call(_ context.Context, req groqClient.CallRequest) (*groqClient.CallResponse, error) {
	f.lastReq = req
	i := f.calls
	if i >= len(f.results) {
		i = len(f.results) - 1
	}
	f.calls++
	r := f.results[i]
	return r.resp, r.err
}

// fakeGeminiClient is an in-memory stand-in for geminiClient.Client.
type fakeGeminiClient struct {
	lastPrompt string
	resp       []float32
	err        error
}

func (f *fakeGeminiClient) GetEmbeddings(_ context.Context, prompt string) ([]float32, error) {
	f.lastPrompt = prompt
	return f.resp, f.err
}

var (
	_ groqClient.Client   = (*fakeGroqClient)(nil)
	_ groqClient.Client   = (*sequencedGroqClient)(nil)
	_ geminiClient.Client = (*fakeGeminiClient)(nil)
)

var historyLimitPattern = regexp.MustCompile(`History\(last (\d+) only\):`)

func newTestAI(t *testing.T, groq groqClient.Client, gemini geminiClient.Client) chatAndAIUsage.AI {
	t.Helper()
	ai, err := chatAndAIUsage.NewAI(groq, gemini)
	require.NoError(t, err)
	require.NotNil(t, ai)
	return ai
}

func Test_NewAI_NilDependencies(t *testing.T) {
	t.Run("nil groq client", func(t *testing.T) {
		ai, err := chatAndAIUsage.NewAI(nil, &fakeGeminiClient{})
		require.Error(t, err)
		require.Nil(t, ai)
	})

	t.Run("nil gemini client", func(t *testing.T) {
		ai, err := chatAndAIUsage.NewAI(&fakeGroqClient{}, nil)
		require.Error(t, err)
		require.Nil(t, ai)
	})
}

func Test_NewAI_Success(t *testing.T) {
	ai, err := chatAndAIUsage.NewAI(&fakeGroqClient{}, &fakeGeminiClient{})
	require.NoError(t, err)
	require.NotNil(t, ai)
}

func Test_GenerateEmbeddings(t *testing.T) {
	gemini := &fakeGeminiClient{resp: []float32{0.1, 0.2, 0.3}}
	ai := newTestAI(t, &fakeGroqClient{}, gemini)

	got, err := ai.GenerateEmbeddings(context.Background(), "  Hello, World!!  ")
	require.NoError(t, err)
	require.Equal(t, []float32{0.1, 0.2, 0.3}, got)
	require.Equal(t, "hello world", gemini.lastPrompt)
}

func Test_GenerateEmbeddings_PropagatesError(t *testing.T) {
	gemini := &fakeGeminiClient{err: fmt.Errorf("boom")}
	ai := newTestAI(t, &fakeGroqClient{}, gemini)

	_, err := ai.GenerateEmbeddings(context.Background(), "hello")
	require.Error(t, err)
}

func Test_GeneratePuzzle(t *testing.T) {
	tests := []struct {
		name      string
		userCtx   chatAndAIUsage.UserContext
		wantTier  string
		wantGames uint
	}{
		{
			name:      "new player",
			userCtx:   chatAndAIUsage.UserContext{},
			wantTier:  "easy",
			wantGames: 0,
		},
		{
			name: "high win rate and high similarity is hard",
			userCtx: chatAndAIUsage.UserContext{
				TotalGames:     10,
				GamesWon:       8,
				RecentGamesWon: 2, // both of the 2 recent games won
				PreviousPuzzlesMaxIntentSimilarityPercentage: []uint{80, 75},
			},
			wantTier:  "hard",
			wantGames: 10,
		},
		{
			name: "low win rate is easy",
			userCtx: chatAndAIUsage.UserContext{
				TotalGames: 10,
				GamesWon:   1,
				PreviousPuzzlesMaxIntentSimilarityPercentage: []uint{10, 20},
			},
			wantTier:  "easy",
			wantGames: 10,
		},
		{
			name: "mixed performance is medium",
			userCtx: chatAndAIUsage.UserContext{
				TotalGames:     10,
				GamesWon:       5,
				RecentGamesWon: 1, // 1 of the 2 recent games won
				PreviousPuzzlesMaxIntentSimilarityPercentage: []uint{50, 60},
			},
			wantTier:  "medium",
			wantGames: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{`{"question": "What is forty two in numeric?", "answer": "42"}`}}}
			ai := newTestAI(t, groq, &fakeGeminiClient{})

			resp, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{UserContext: tt.userCtx})
			require.NoError(t, err)
			require.Equal(t, "What is forty two in numeric?", resp.HiddenPrompt)
			require.Equal(t, "42", resp.CanonicalAnswer)

			require.Equal(t, templates.GeneratePuzzleSystemPrompt(), groq.lastReq.SystemPrompt)
			require.Contains(t, groq.lastReq.UserPrompt, fmt.Sprintf("Difficulty:%s", tt.wantTier))
			require.Contains(t, groq.lastReq.UserPrompt, fmt.Sprintf("Games:%d", tt.wantGames))

			require.NotNil(t, groq.lastReq.MaxTemperature)
			require.Equal(t, 1.0, *groq.lastReq.MaxTemperature)
		})
	}
}

func Test_GeneratePuzzle_StripsTrailingPunctuationFromAnswer(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{`{"question": "What color is the sky?", "answer": "Blue."}`}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{})
	require.NoError(t, err)
	require.Equal(t, "Blue", resp.CanonicalAnswer)
}

func Test_GeneratePuzzle_PassesRecentPuzzlesThrough(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{`{"question": "q", "answer": "a"}`}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{
		RecentPuzzles: []chatAndAIUsage.RecentPuzzle{
			{Question: "What is the only bird that can fly backwards?", Answer: "Hummingbird"},
		},
	})
	require.NoError(t, err)
	require.Contains(t, groq.lastReq.UserPrompt, "What is the only bird that can fly backwards?")
	require.Contains(t, groq.lastReq.UserPrompt, "Hummingbird")
}

// Test_GeneratePuzzle_PassesUsedAnswersThrough mirrors
// Test_GeneratePuzzle_PassesRecentPuzzlesThrough for the separate
// all-time UsedAnswers list (Fix #8) - a bare answer with no paired
// question should still reach the prompt.
func Test_GeneratePuzzle_PassesUsedAnswersThrough(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{`{"question": "q", "answer": "a"}`}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{
		UsedAnswers: []string{"Jupiter", "Hummingbird"},
	})
	require.NoError(t, err)
	require.Contains(t, groq.lastReq.UserPrompt, "Jupiter")
	require.Contains(t, groq.lastReq.UserPrompt, "Hummingbird")
}

func Test_GeneratePuzzle_EmptyResponse(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{})
	require.ErrorIs(t, err, chatAndAIUsage.ErrEmptyLLMResponse)
}

func Test_GeneratePuzzle_PropagatesGroqError(t *testing.T) {
	groq := &fakeGroqClient{err: fmt.Errorf("groq is down")}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{})
	require.Error(t, err)
}

func Test_GeneratePuzzle_RetriesPastMalformedResponse(t *testing.T) {
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"not json at all"}}},
		{resp: &groqClient.CallResponse{Response: []string{`{"question": "What is the largest planet in our solar system?", "answer": "Jupiter"}`}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{})
	require.NoError(t, err)
	require.Equal(t, "What is the largest planet in our solar system?", resp.HiddenPrompt)
	require.Equal(t, "Jupiter", resp.CanonicalAnswer)
	require.Equal(t, 2, groq.calls)
}

func Test_GeneratePuzzle_RetriesPastOverlyLongAnswer(t *testing.T) {
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{`{"question": "q", "answer": "The answer to your question is that it is forty two, which is a well-known number."}`}}},
		{resp: &groqClient.CallResponse{Response: []string{`{"question": "q", "answer": "42"}`}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{})
	require.NoError(t, err)
	require.Equal(t, "42", resp.CanonicalAnswer)
	require.Equal(t, 2, groq.calls)
}

func Test_GeneratePuzzle_FailsAfterAllAttemptsMalformed(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"not json at all"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GeneratePuzzle(context.Background(), chatAndAIUsage.GeneratePuzzleRequest{})
	require.ErrorIs(t, err, chatAndAIUsage.ErrAllAttemptsMalformed)
}

func Test_GenerateHint_SmallHistoryIsNotTruncated(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"  You're warm.  "}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	history := []chatAndAIUsage.ChatMessage{
		{Role: chatAndAIUsage.UserRole, Message: "msg-0"},
		{Role: chatAndAIUsage.AssistantRole, Message: "msg-1"},
	}

	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt:      "hidden",
		CanonicalAnswer:   "answer",
		History:           history,
		Guess:             "guess",
		CurrentScore:      62.4,
		PreviousBestScore: 45,
	})
	require.NoError(t, err)
	require.Equal(t, "You're warm.", resp.Hint)

	require.Equal(t, templates.HintSystemPrompt(), groq.lastReq.SystemPrompt)
	require.Contains(t, groq.lastReq.UserPrompt, "Hidden:hidden")
	require.Contains(t, groq.lastReq.UserPrompt, "Answer:answer")
	require.Contains(t, groq.lastReq.UserPrompt, "Guess:guess")
	require.Contains(t, groq.lastReq.UserPrompt, "msg-0")
	require.Contains(t, groq.lastReq.UserPrompt, "msg-1")
	require.False(t, historyLimitPattern.MatchString(groq.lastReq.UserPrompt), "untruncated history must not carry a truncation note")

	// The whole point of wiring the score through: the hint LLM must see
	// both the current guess's score and the player's prior best, rounded
	// to plain integers, so it can acknowledge warmer/colder.
	require.Contains(t, groq.lastReq.UserPrompt, "GuessScore:62/100")
	require.Contains(t, groq.lastReq.UserPrompt, "PreviousBestScore:45/100")

	// Hint calls must use the narrowed, low-variance temperature range, not
	// the default wide/high band shared by every other call type.
	require.NotNil(t, groq.lastReq.MaxTemperature)
	require.Equal(t, 1.1, *groq.lastReq.MaxTemperature)
}

func Test_GenerateHint_PassesGapHintThrough(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"You're warm."}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt:    "hidden",
		CanonicalAnswer: "answer",
		Guess:           "guess",
		GapHint:         "usually, eat",
	})
	require.NoError(t, err)
	require.Contains(t, groq.lastReq.UserPrompt, "GapHint:usually, eat")
}

func Test_GenerateHint_ClampsOutOfRangeScores(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"hint"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt:      "h",
		CanonicalAnswer:   "a",
		Guess:             "g",
		CurrentScore:      140,
		PreviousBestScore: -20,
	})
	require.NoError(t, err)

	require.Contains(t, groq.lastReq.UserPrompt, "GuessScore:100/100")
	require.Contains(t, groq.lastReq.UserPrompt, "PreviousBestScore:0/100")
}

func Test_GenerateHint_TruncatesByMessageCount(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"hint"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	const totalMessages = 20
	history := make([]chatAndAIUsage.ChatMessage, 0, totalMessages)
	for i := range totalMessages {
		role := chatAndAIUsage.UserRole
		if i%2 == 1 {
			role = chatAndAIUsage.AssistantRole
		}
		history = append(history, chatAndAIUsage.ChatMessage{Role: role, Message: fmt.Sprintf("msg-%d", i)})
	}

	_, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt:    "hidden",
		CanonicalAnswer: "answer",
		History:         history,
		Guess:           "guess",
	})
	require.NoError(t, err)

	matches := historyLimitPattern.FindStringSubmatch(groq.lastReq.UserPrompt)
	require.NotNil(t, matches, "expected a truncation note in the user prompt")
	require.Equal(t, "12", matches[1])

	// Oldest messages must be dropped, most recent ones kept.
	require.NotContains(t, groq.lastReq.UserPrompt, "msg-0")
	require.Contains(t, groq.lastReq.UserPrompt, fmt.Sprintf("msg-%d", totalMessages-1))
}

func Test_GenerateHint_TruncatesByCharacterBudget(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"hint"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	// Well under the message-count cap, but each message is large enough
	// that the serialized history alone blows past the character budget.
	longMessage := func(marker string) string {
		return marker + ":" + repeatString("x", 900)
	}
	history := []chatAndAIUsage.ChatMessage{
		{Role: chatAndAIUsage.UserRole, Message: longMessage("first")},
		{Role: chatAndAIUsage.AssistantRole, Message: longMessage("second")},
		{Role: chatAndAIUsage.UserRole, Message: longMessage("third")},
	}

	_, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt:    "hidden",
		CanonicalAnswer: "answer",
		History:         history,
		Guess:           "guess",
	})
	require.NoError(t, err)

	matches := historyLimitPattern.FindStringSubmatch(groq.lastReq.UserPrompt)
	require.NotNil(t, matches, "expected a truncation note when history exceeds the character budget")
	keptCount, err := strconv.Atoi(matches[1])
	require.NoError(t, err)
	require.Less(t, keptCount, len(history), "fewer than the original messages must have been kept")
	require.Contains(t, groq.lastReq.UserPrompt, "third", "the most recent message must always survive truncation")
	require.NotContains(t, groq.lastReq.UserPrompt, "first", "the oldest message must be dropped first")
}

func Test_GenerateHint_RetriesPastRefusalStyleResponse(t *testing.T) {
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"I'm sorry, but I can't provide that."}}},
		{resp: &groqClient.CallResponse{Response: []string{"Getting warmer, think about the sky."}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{HiddenPrompt: "h", CanonicalAnswer: "a", Guess: "g"})
	require.NoError(t, err)
	require.Equal(t, "Getting warmer, think about the sky.", resp.Hint)
	require.Equal(t, 2, groq.calls)
}

func Test_GenerateHint_RetriesPastRefusalWithCurlyApostrophe(t *testing.T) {
	// Real production case: some models render refusals with a
	// typographic apostrophe ("I'm sorry" with U+2019), which a
	// straight-quote-only phrase match would silently miss, letting the
	// raw refusal text through as if it were a legitimate hint.
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"I’m sorry, but I can’t provide that."}}},
		{resp: &groqClient.CallResponse{Response: []string{"Getting warmer, think about the sky."}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{HiddenPrompt: "h", CanonicalAnswer: "a", Guess: "g"})
	require.NoError(t, err)
	require.Equal(t, "Getting warmer, think about the sky.", resp.Hint)
	require.Equal(t, 2, groq.calls)
}

func Test_GenerateHint_FallsBackToGenericHintIfAllAttemptsLookLikeRefusals(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"I cannot help with that request."}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	// A hint is supplementary, not core to the guess - even a persistent
	// refusal-looking response must not fail the whole guess submission.
	// The raw refusal text itself must never reach the player though (it
	// breaks immersion mid-game) - a generic nudge is the fallback instead.
	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{HiddenPrompt: "h", CanonicalAnswer: "a", Guess: "g"})
	require.NoError(t, err)
	require.NotEqual(t, "I cannot help with that request.", resp.Hint)
	require.NotEmpty(t, resp.Hint)
}

func Test_GenerateHint_RetriesPastLeakedAnswer(t *testing.T) {
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"Apple"}}},
		{resp: &groqClient.CallResponse{Response: []string{"Getting warmer, think about the color."}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt: "h", CanonicalAnswer: "Apple", Guess: "g",
	})
	require.NoError(t, err)
	require.Equal(t, "Getting warmer, think about the color.", resp.Hint)
	require.Equal(t, 2, groq.calls)
}

func Test_GenerateHint_AllowsHintToEchoAnswerWordThePlayerAlreadyGuessed(t *testing.T) {
	// Real false-positive caught via the SLT suite: a hint that references
	// "Jupiter" is not a leak when the player's own guess already said
	// "Jupiter" ("Tell me something about Jupiter in general.") - they
	// aren't learning anything new. Only flag hints that state the answer
	// the player hasn't already said themselves.
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"About the same as last time, think about what's special about Jupiter's size."}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt: "h", CanonicalAnswer: "Jupiter", Guess: "Tell me something about Jupiter in general.",
	})
	require.NoError(t, err)
	require.Equal(t, "About the same as last time, think about what's special about Jupiter's size.", resp.Hint)
	require.Equal(t, 1, groq.calls, "no retry should fire - the player already said the word themselves")
}

func Test_GenerateHint_FlagsExplicitConfirmationEvenIfGuessAlreadySaidTheWord(t *testing.T) {
	// Real false negative caught via the SLT suite: guess already said
	// "Piano", so the word-presence exemption let a hint through even
	// though it explicitly confirmed the answer ("The answer is piano.")
	// rather than just referencing the word - confirmation must always be
	// caught, regardless of what the guess said.
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"The answer is piano."}}},
		{resp: &groqClient.CallResponse{Response: []string{"Closer now - think about where you'd usually find this."}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt: "h", CanonicalAnswer: "Piano", Guess: "Tell me something about Piano in general.",
	})
	require.NoError(t, err)
	require.Equal(t, "Closer now - think about where you'd usually find this.", resp.Hint)
	require.Equal(t, 2, groq.calls)
}

func Test_GenerateHint_FallsBackToGenericHintIfEveryAttemptLeaks(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"Apple"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	// A leaked answer must never reach the player, even as a last resort -
	// unlike a persistent refusal (safe to surface stale), so every retry
	// leaking must fall back to a generic, always-safe hint instead.
	resp, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{
		HiddenPrompt: "h", CanonicalAnswer: "Apple", Guess: "g",
	})
	require.NoError(t, err)
	require.NotContains(t, resp.Hint, "Apple")
}

func Test_GenerateHint_EmptyResponse(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{HiddenPrompt: "h", CanonicalAnswer: "a", Guess: "g"})
	require.ErrorIs(t, err, chatAndAIUsage.ErrEmptyLLMResponse)
}

func Test_GenerateHint_PropagatesGroqError(t *testing.T) {
	groq := &fakeGroqClient{err: fmt.Errorf("groq is down")}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.GenerateHint(context.Background(), chatAndAIUsage.GenerateHintRequest{HiddenPrompt: "h", CanonicalAnswer: "a", Guess: "g"})
	require.Error(t, err)
}

func Test_ScoreIntent_PlainJSON(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{`{"intent_score": 73}`}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.ScoreIntent(context.Background(), chatAndAIUsage.ScoreIntentRequest{TargetIntent: "target", GivenIntent: "given"})
	require.NoError(t, err)
	require.Equal(t, 73, resp.IntentScore)

	require.Equal(t, templates.ScoreIntentSystemPrompt(), groq.lastReq.SystemPrompt)
	require.Contains(t, groq.lastReq.UserPrompt, "Target:target")
	require.Contains(t, groq.lastReq.UserPrompt, "Given:given")
}

func Test_ScoreIntent_BacktickWrapped(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"```json\n{\"intent_score\": 55}\n```"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.ScoreIntent(context.Background(), chatAndAIUsage.ScoreIntentRequest{TargetIntent: "t", GivenIntent: "g"})
	require.NoError(t, err)
	require.Equal(t, 55, resp.IntentScore)
}

func Test_ScoreIntent_RetriesUntilParseable(t *testing.T) {
	groq := &sequencedGroqClient{results: []groqClientCallResult{
		{resp: &groqClient.CallResponse{Response: []string{"not json at all"}}},
		{resp: &groqClient.CallResponse{Response: []string{"still not json"}}},
		{resp: &groqClient.CallResponse{Response: []string{`{"intent_score": 90}`}}},
	}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	resp, err := ai.ScoreIntent(context.Background(), chatAndAIUsage.ScoreIntentRequest{TargetIntent: "t", GivenIntent: "g"})
	require.NoError(t, err)
	require.Equal(t, 90, resp.IntentScore)
	require.Equal(t, 3, groq.calls)
}

func Test_ScoreIntent_AllAttemptsExhausted(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"never valid json"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.ScoreIntent(context.Background(), chatAndAIUsage.ScoreIntentRequest{TargetIntent: "t", GivenIntent: "g"})
	require.Error(t, err)
}

func Test_ScoreIntent_EmptyResponse(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	_, err := ai.ScoreIntent(context.Background(), chatAndAIUsage.ScoreIntentRequest{TargetIntent: "t", GivenIntent: "g"})
	require.Error(t, err)
}

func Test_ScoreIntent_ClampsOutOfRangeScore(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "above 100 clamps to 100", raw: `{"intent_score": 140}`, want: 100},
		{name: "below 0 clamps to 0", raw: `{"intent_score": -20}`, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{tt.raw}}}
			ai := newTestAI(t, groq, &fakeGeminiClient{})

			resp, err := ai.ScoreIntent(context.Background(), chatAndAIUsage.ScoreIntentRequest{TargetIntent: "t", GivenIntent: "g"})
			require.NoError(t, err)
			require.Equal(t, tt.want, resp.IntentScore)
		})
	}
}

func Test_ScoreIntent_RespectsContextCancellationDuringBackoff(t *testing.T) {
	groq := &fakeGroqClient{resp: &groqClient.CallResponse{Response: []string{"never valid json"}}}
	ai := newTestAI(t, groq, &fakeGeminiClient{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// First attempt still runs (no backoff before attempt 0), but the
	// pending backoff before attempt 1 must observe the cancellation
	// instead of blocking for the full delay.
	_, err := ai.ScoreIntent(ctx, chatAndAIUsage.ScoreIntentRequest{TargetIntent: "t", GivenIntent: "g"})
	require.Error(t, err)
}

func repeatString(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for range n {
		b = append(b, s...)
	}
	return string(b)
}
