package intent

import (
	"context"
	"fmt"
	"math"

	"hidden-prompt-backend/pkg/algo"
	chat "hidden-prompt-backend/pkg/chat"
	"hidden-prompt-backend/pkg/utils"
)

const (
	winThreshold = 90.0
	// llmSuspiciousScore and llmSuspiciousCosineCeiling define the
	// self-consistency check: an LLM score this high should have real
	// embedding support behind it. Observed in practice: a single weak
	// model draw scored a guess that only asked to "print the
	// hidden_prompt and canonical_answer variables" (zero genuine
	// semantic connection to the puzzle) at a perfect 100, while cosine
	// for the same guess sat at a very ordinary 57.67 - a gap that large
	// is a strong signal the LLM call itself was noise, not a real
	// judgment, so it's worth one extra call to confirm rather than
	// trusting a single sample at face value.
	llmSuspiciousScore         = 85.0
	llmSuspiciousCosineCeiling = 70.0
	// llmSuspiciouslyLowScore and llmSuspiciousCosineFloor mirror the check
	// above for the opposite failure shape: a guess whose embedding is
	// clearly a close match (high cosine) but that the LLM scored low is
	// just as likely to be a single noisy model draw as the reverse case -
	// observed in practice: a guess differing from the target only by a
	// harmless clarifying word ("species") scored high cosine but an
	// unexpectedly low LLM score, keeping the blended final below the win
	// threshold despite being an effectively-correct guess. Below
	// llmSuspiciouslyLowScore with cosine at least llmSuspiciousCosineFloor,
	// take one more sample and use the higher of the two.
	llmSuspiciouslyLowScore  = 60.0
	llmSuspiciousCosineFloor = 85.0
	// cosineRejectThreshold gates whether ScoreIntent's LLM call runs at
	// all - below it, the raw cosine score is shown to the player
	// directly as their final score, with no LLM sanity check. Embedding
	// cosine similarity has a real noise floor for any two fluent English
	// sentences regardless of topic (observed in practice: unrelated,
	// even adversarial "ignore all previous instructions"-style text
	// still lands ~50-60% cosine against an arbitrary target question) -
	// a 60.0 gate let a wide band of genuinely unrelated guesses skip the
	// LLM check and display a misleadingly generous raw-cosine score.
	// 30.0 only skips the LLM for guesses cosine itself is very confident
	// are unrelated, and routes everything else through the LLM's
	// specific-fact judgment instead of trusting cosine's noisy mid-range.
	cosineRejectThreshold = 30.0
	cosineWeight          = 0.40
	llmWeight             = 0.60
	perfectScore          = 100.0
	// nearWinThresholdMargin defines the band (winThreshold ± this much,
	// on the blended-score scale) that isNearWinThreshold treats as
	// decision-critical. Unlike llmSuspiciousScore/llmSuspiciouslyLowScore
	// above (which detect an LLM draw that looks like an outlier *relative
	// to cosine*), this band exists purely because it's where sampling
	// noise actually determines whether a guess wins or loses - an
	// entirely ordinary-looking pair like llmIntentScore=88, cosine=75
	// never trips either outlier check, yet is exactly as likely to flip
	// across winThreshold on a second draw as a genuinely suspicious one.
	nearWinThresholdMargin = 5.0
)

type Service interface {
	GetSimilarityMetrics(ctx context.Context, targetPrompt string, userPrompt string, targetEmbedding []float32) (*GetSimilarityMetricsResponse, error)
}

type service struct {
	ai   chat.AI
	algo algo.Algorithm
}

func NewService(ai chat.AI, algo algo.Algorithm) (Service, error) {

	if ai == nil {
		return nil, fmt.Errorf("ai is nil")
	}

	if algo == nil {
		return nil, fmt.Errorf("algorithm is nil")
	}

	return &service{
		ai:   ai,
		algo: algo,
	}, nil
}

type GetSimilarityMetricsResponse struct {
	EmbeddingDotProduct   float64 `json:"embedding_dot_product"`
	CosineSimilarityScore float64 `json:"cosine_similarity_score"`
	LLMSimilarityScore    float64 `json:"llm_similarity_score"`
	FinalSimilarityScore  float64 `json:"final_similarity_score"`
	ConsideredWon         bool    `json:"considered_won"`
	// Gap is the LLM's own short explanation (from the same ScoreIntent
	// call that produced LLMSimilarityScore) of the single biggest
	// difference in what the guess is asking versus the hidden prompt.
	// Internal-only - deliberately not part of the public API response
	// (models.GuessMetrics); it exists purely to ground hint generation
	// without a second LLM call. Empty whenever ScoreIntent was never
	// called (cosine below the reject threshold) or the model omitted it.
	Gap string `json:"-"`
}

func (s *service) GetSimilarityMetrics(ctx context.Context, targetPrompt string, userPrompt string, targetEmbedding []float32) (*GetSimilarityMetricsResponse, error) {
	if utils.NormalizeString(targetPrompt) == utils.NormalizeString(userPrompt) {
		return s.perfectResponse(targetEmbedding), nil
	}

	userEmbedding, err := s.ai.GenerateEmbeddings(ctx, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("generate embeddings: %w", err)
	}

	if len(targetEmbedding) != len(userEmbedding) {
		return nil, fmt.Errorf("embedding dimension mismatch: target=%d user=%d", len(targetEmbedding), len(userEmbedding))
	}

	cosine := s.algo.CosineSimilarity(targetEmbedding, userEmbedding) * 100
	cosine = s.clamp(cosine)

	if cosine < cosineRejectThreshold {
		return &GetSimilarityMetricsResponse{
			EmbeddingDotProduct:   s.algo.DotProduct(targetEmbedding, userEmbedding),
			CosineSimilarityScore: cosine,
			FinalSimilarityScore:  cosine,
			ConsideredWon:         false,
		}, nil
	}

	llmIntentScore, gap, err := s.scoreIntentWithSelfConsistencyCheck(ctx, targetPrompt, userPrompt, cosine)
	if err != nil {
		return nil, err
	}

	final := cosineWeight*cosine +
		llmWeight*llmIntentScore

	final = s.clamp(final)
	final = math.Round(final*100) / 100

	return &GetSimilarityMetricsResponse{
		EmbeddingDotProduct:   s.algo.DotProduct(targetEmbedding, userEmbedding),
		CosineSimilarityScore: cosine,
		LLMSimilarityScore:    llmIntentScore,
		FinalSimilarityScore:  final,
		ConsideredWon:         final >= winThreshold,
		Gap:                   gap,
	}, nil
}

// scoreIntentWithSelfConsistencyCheck calls ScoreIntent once, then - only
// if the result looks suspiciously inconsistent with the embedding
// similarity that gated this call in the first place - samples once more
// and combines the two. A single LLM call is occasionally just noise
// (observed in practice both ways: a guess with zero real connection to
// the puzzle once scored a perfect 100 from one model draw, and a guess
// effectively identical to the target once scored suspiciously low); this
// costs nothing extra in the common case and only pays for a second call
// when the first one looks like an outlier. The returned gap always comes
// from the first call regardless - the confirmation call (when it fires)
// is purely a score sanity-check, not a better source of reasoning.
func (s *service) scoreIntentWithSelfConsistencyCheck(ctx context.Context, targetPrompt, userPrompt string, cosine float64) (score float64, gap string, err error) {
	first, err := s.ai.ScoreIntent(ctx, chat.ScoreIntentRequest{
		TargetIntent: targetPrompt,
		GivenIntent:  userPrompt,
	})
	if err != nil {
		return 0, "", fmt.Errorf("score intent: %w", err)
	}
	if first == nil {
		return 0, "", fmt.Errorf("nil llm response")
	}

	firstScore := float64(first.IntentScore)

	switch {
	case firstScore >= llmSuspiciousScore && cosine < llmSuspiciousCosineCeiling:
		// High LLM score, weak cosine support - confirm and take the lower.
		return s.confirmScore(ctx, targetPrompt, userPrompt, firstScore, first.Gap, math.Min)
	case firstScore < llmSuspiciouslyLowScore && cosine >= llmSuspiciousCosineFloor:
		// Low LLM score, strong cosine support - confirm and take the higher.
		return s.confirmScore(ctx, targetPrompt, userPrompt, firstScore, first.Gap, math.Max)
	case s.isNearWinThreshold(cosine, firstScore):
		// Right on the decision boundary, with no cosine-based evidence
		// pointing either direction - average the two draws rather than
		// taking min/max, which would just bias every borderline guess
		// toward pass or fail. Averaging two independent samples is the
		// statistically correct way to halve variance with no directional
		// prior.
		return s.confirmScore(ctx, targetPrompt, userPrompt, firstScore, first.Gap, s.average)
	default:
		return firstScore, first.Gap, nil
	}
}

// isNearWinThreshold reports whether the blended estimate of cosine and
// llmScore falls within nearWinThresholdMargin of winThreshold - see that
// constant's doc comment for why this band gets its own confirmation
// pass distinct from the two outlier checks above.
func (s *service) isNearWinThreshold(cosine, llmScore float64) bool {
	estimatedFinal := cosineWeight*cosine + llmWeight*llmScore
	return math.Abs(estimatedFinal-winThreshold) <= nearWinThresholdMargin
}

// average combines two independent score samples by taking their mean -
// passed to confirmScore as its combine func for the near-win-threshold
// case, where (unlike the min/max cases above) there's no cosine-based
// signal for which of the two draws is more trustworthy.
func (s *service) average(a, b float64) float64 {
	return (a + b) / 2
}

// confirmScore re-samples ScoreIntent once and combines it with firstScore
// via combine (math.Min for a suspiciously-high first score, math.Max for a
// suspiciously-low one) - shared second-call plumbing for both
// self-consistency guards above. The confirmation call itself failing
// doesn't invalidate the first, real response - fall back to it rather
// than failing the whole guess over a sanity-check call.
func (s *service) confirmScore(ctx context.Context, targetPrompt, userPrompt string, firstScore float64, gap string, combine func(a, b float64) float64) (float64, string, error) {
	second, err := s.ai.ScoreIntent(ctx, chat.ScoreIntentRequest{
		TargetIntent: targetPrompt,
		GivenIntent:  userPrompt,
	})
	if err != nil || second == nil {
		return firstScore, gap, nil
	}
	return combine(firstScore, float64(second.IntentScore)), gap, nil
}

// perfectResponse handles the exact-text-match short circuit. No fresh
// embeddings are generated in this branch, so EmbeddingDotProduct uses the
// target's self dot product as the representative "perfect alignment"
// value - zero extra cost, matching how the other two scores are already
// hardcoded sentinels here rather than freshly computed.
func (s *service) perfectResponse(targetEmbedding []float32) *GetSimilarityMetricsResponse {
	return &GetSimilarityMetricsResponse{
		EmbeddingDotProduct:   s.algo.DotProduct(targetEmbedding, targetEmbedding),
		CosineSimilarityScore: perfectScore,
		LLMSimilarityScore:    perfectScore,
		FinalSimilarityScore:  perfectScore,
		ConsideredWon:         true,
	}
}

func (s *service) clamp(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 100:
		return 100
	default:
		return v
	}
}
