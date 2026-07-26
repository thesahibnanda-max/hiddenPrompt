package algo_test

import (
	"hidden-prompt-backend/pkg/algo"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCosineSimilarity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 3},
			expected: 1,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0},
			b:        []float32{0, 1},
			expected: 0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 2},
			b:        []float32{-1, -2},
			expected: -1,
		},
		{
			name:     "zero lhs vector",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 2, 3},
			expected: 0,
		},
		{
			name:     "zero rhs vector",
			a:        []float32{1, 2, 3},
			b:        []float32{0, 0, 0},
			expected: 0,
		},
		{
			name:     "both zero vectors",
			a:        []float32{0, 0},
			b:        []float32{0, 0},
			expected: 0,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0,
		},
		{
			name: "known example",
			a:    []float32{1, 2, 3},
			b:    []float32{4, 5, 6},
			// 32 / (sqrt(14) * sqrt(77))
			expected: 0.974631846,
		},
	}

	al := algo.NewAlgorithm()

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := al.CosineSimilarity(tc.a, tc.b)
			require.InDelta(t, tc.expected, got, 1e-9)
		})
	}
}

func TestDotProduct(t *testing.T) {
	t.Parallel()

	al := algo.NewAlgorithm()

	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 3},
			expected: 14,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0},
			b:        []float32{0, 1},
			expected: 0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 2},
			b:        []float32{-1, -2},
			expected: -5,
		},
		{
			name:     "zero vector",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 2, 3},
			expected: 0,
		},
		{
			name:     "mismatched length",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2},
			expected: 0,
		},
		{
			name:     "known example",
			a:        []float32{1, 2, 3},
			b:        []float32{4, 5, 6},
			expected: 32,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := al.DotProduct(tc.a, tc.b)
			require.InDelta(t, tc.expected, got, 1e-9)
		})
	}
}

func TestCosineSimilarity_IsSymmetric(t *testing.T) {
	t.Parallel()

	al := algo.NewAlgorithm()

	a := []float32{1, 5, -2, 9}
	b := []float32{-3, 4, 1, 7}

	ab := al.CosineSimilarity(a, b)
	ba := al.CosineSimilarity(b, a)

	require.InDelta(t, ab, ba, 1e-12)
}

func TestCosineSimilarity_Range(t *testing.T) {
	t.Parallel()

	al := algo.NewAlgorithm()

	a := []float32{3, -2, 5}
	b := []float32{1, 4, -6}

	got := al.CosineSimilarity(a, b)

	require.LessOrEqual(t, got, 1.0+1e-12)
	require.GreaterOrEqual(t, got, -1.0-1e-12)
	require.False(t, math.IsNaN(got))
	require.False(t, math.IsInf(got, 0))
}
