package algo

import (
	"math"
)

type Algorithm interface {
	CosineSimilarity(a, b []float32) float64
	// DotProduct returns the raw (non-normalized) dot product of a and b -
	// unlike CosineSimilarity, this is sensitive to vector magnitude, not
	// just direction, so it's a genuinely distinct signal rather than a
	// re-derivation of the same number.
	DotProduct(a, b []float32) float64
}

func NewAlgorithm() Algorithm {
	return &algorithm{}
}

type algorithm struct{}

func (al *algorithm) DotProduct(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

func (al *algorithm) CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64

	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])

		dot += af * bf
		normA += af * af
		normB += bf * bf
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
