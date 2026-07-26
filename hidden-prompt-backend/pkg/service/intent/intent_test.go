package intent_test

import (
	"context"
	"fmt"
	"testing"

	"hidden-prompt-backend/pkg/algo"
	chat "hidden-prompt-backend/pkg/chat"
	"hidden-prompt-backend/pkg/service/intent"

	"github.com/stretchr/testify/require"
)

// fakeAI implements chat.AI with fully scripted responses, so intent
// scoring behavior can be tested deterministically without a real Groq
// call - GetEmbeddings returns fixed vectors chosen so CosineSimilarity
// lands at a specific, known percentage, and ScoreIntent returns each
// entry in scoreIntentResponses in order (one per call).
type fakeAI struct {
	embeddings           map[string][]float32
	scoreIntentResponses []int
	// scoreIntentGaps is optional - when shorter than scoreIntentResponses,
	// missing entries default to "" (mirrors the model omitting the field).
	scoreIntentGaps  []string
	scoreIntentCalls int
}

func (f *fakeAI) GenerateEmbeddings(_ context.Context, dataString string) ([]float32, error) {
	v, ok := f.embeddings[dataString]
	if !ok {
		return nil, fmt.Errorf("fakeAI: no embedding scripted for %q", dataString)
	}
	return v, nil
}

func (f *fakeAI) GeneratePuzzle(context.Context, chat.GeneratePuzzleRequest) (*chat.GeneratePuzzleResponse, error) {
	return nil, fmt.Errorf("not implemented in fake")
}

func (f *fakeAI) GenerateHint(context.Context, chat.GenerateHintRequest) (*chat.GenerateHintResponse, error) {
	return nil, fmt.Errorf("not implemented in fake")
}

func (f *fakeAI) ScoreIntent(context.Context, chat.ScoreIntentRequest) (*chat.ScoreIntentResponse, error) {
	if f.scoreIntentCalls >= len(f.scoreIntentResponses) {
		return nil, fmt.Errorf("fakeAI: no more scripted ScoreIntent responses")
	}
	score := f.scoreIntentResponses[f.scoreIntentCalls]
	var gap string
	if f.scoreIntentCalls < len(f.scoreIntentGaps) {
		gap = f.scoreIntentGaps[f.scoreIntentCalls]
	}
	f.scoreIntentCalls++
	return &chat.ScoreIntentResponse{IntentScore: score, Gap: gap}, nil
}

var (
	_ chat.AI = (*fakeAI)(nil)
)

// Two vectors whose cosine similarity is a known, moderate value (well
// above the 30.0 cosineRejectThreshold so ScoreIntent runs, but below the
// 70.0 self-consistency ceiling) - orthogonal-ish but not identical.
var (
	targetEmbedding = []float32{1, 0, 0, 0}
	givenEmbedding  = []float32{0.6, 0.8, 0, 0} // cosine = 0.6 -> 60%
)

func newTestService(t *testing.T, ai chat.AI) intent.Service {
	t.Helper()
	svc, err := intent.NewService(ai, algo.NewAlgorithm())
	require.NoError(t, err)
	return svc
}

func Test_GetSimilarityMetrics_HighLLMScoreWithWeakCosineTriggersRecheck(t *testing.T) {
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": givenEmbedding,
		},
		// First draw is the suspicious 100 with only 60% cosine backing it
		// (mirrors the real observed failure); the confirmation call comes
		// back much more measured - the lower of the two must win.
		scoreIntentResponses: []int{100, 20},
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, 2, ai.scoreIntentCalls, "a suspicious first score must trigger exactly one confirmation call")
	require.Equal(t, 20.0, resp.LLMSimilarityScore, "the lower of the two scores must be used")
	require.False(t, resp.ConsideredWon)
}

func Test_GetSimilarityMetrics_LowLLMScoreWithStrongCosineTriggersRecheck(t *testing.T) {
	// Mirror of the high-score guard above, for the opposite real-world
	// failure: a guess whose embedding is a strong match (95% cosine) but
	// whose first LLM draw scored suspiciously low - the higher of the two
	// scores must win, not the lower, since here the embedding is the
	// signal to trust and the low LLM draw is the outlier.
	strongCosineEmbedding := []float32{0.95, 0.3122499, 0, 0} // cosine with {1,0,0,0} = 95%
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": strongCosineEmbedding,
		},
		scoreIntentResponses: []int{50, 92},
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, 2, ai.scoreIntentCalls, "a suspiciously low first score with strong cosine support must trigger exactly one confirmation call")
	require.Equal(t, 92.0, resp.LLMSimilarityScore, "the higher of the two scores must be used")
}

func Test_GetSimilarityMetrics_LowLLMScoreNeverTriggersRecheck(t *testing.T) {
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": givenEmbedding,
		},
		scoreIntentResponses: []int{20},
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, 1, ai.scoreIntentCalls, "a score below the suspicious threshold must not trigger a second call")
	require.Equal(t, 20.0, resp.LLMSimilarityScore)
}

func Test_GetSimilarityMetrics_HighLLMScoreWithStrongCosineNeverTriggersRecheck(t *testing.T) {
	// cosine=100% (identical vectors) plus a high LLM score is not
	// suspicious - both signals agree, so no confirmation call should fire.
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": {1, 0, 0, 0},
		},
		scoreIntentResponses: []int{95},
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, 1, ai.scoreIntentCalls)
	require.Equal(t, 95.0, resp.LLMSimilarityScore)
	require.True(t, resp.ConsideredWon)
}

func Test_GetSimilarityMetrics_BelowCosineRejectThresholdSkipsLLMEntirely(t *testing.T) {
	// Deliberately orthogonal vectors -> cosine = 0%, well under the
	// reject threshold - ScoreIntent must never be called at all.
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": {0, 1, 0, 0},
		},
		scoreIntentResponses: nil,
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, 0, ai.scoreIntentCalls)
	require.Equal(t, 0.0, resp.LLMSimilarityScore)
	require.Equal(t, resp.CosineSimilarityScore, resp.FinalSimilarityScore)
}

func Test_GetSimilarityMetrics_GapComesFromFirstCallEvenWhenRecheckFires(t *testing.T) {
	// The confirmation call is purely a score sanity-check, not a better
	// source of reasoning - its gap (if any) must be discarded in favor of
	// the first call's.
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": givenEmbedding,
		},
		scoreIntentResponses: []int{100, 20},
		scoreIntentGaps:      []string{"usually, eat", "confirmation call gap - must not be used"},
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, "usually, eat", resp.Gap)
}

func Test_GetSimilarityMetrics_GapEmptyWhenBelowCosineRejectThreshold(t *testing.T) {
	ai := &fakeAI{
		embeddings: map[string][]float32{
			"given": {0, 1, 0, 0},
		},
	}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "target", "given", targetEmbedding)
	require.NoError(t, err)

	require.Empty(t, resp.Gap)
}

func Test_GetSimilarityMetrics_ExactTextMatchIsAlwaysPerfectAndSkipsLLM(t *testing.T) {
	ai := &fakeAI{scoreIntentResponses: nil}
	svc := newTestService(t, ai)

	resp, err := svc.GetSimilarityMetrics(context.Background(), "same text", "same text", targetEmbedding)
	require.NoError(t, err)

	require.Equal(t, 0, ai.scoreIntentCalls)
	require.Equal(t, 100.0, resp.FinalSimilarityScore)
	require.True(t, resp.ConsideredWon)
}
