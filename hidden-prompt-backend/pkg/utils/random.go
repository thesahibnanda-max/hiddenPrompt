package utils

import (
	"math/rand/v2"

	"github.com/sethvargo/go-password/password"
)

func GenerateRandomString(len uint) (string, error) {
	if len == 0 {
		return "", nil
	}
	s, err := password.Generate(int(len), rand.IntN(int(len)), 0, false, true)
	if err != nil {
		return "", err
	}

	return s, nil
}

// RangeWeight describes a sub-range of a GenerateRandomFloat64 call and the
// relative weight it should be picked with.
type RangeWeight struct {
	Start  float64
	End    float64
	Weight float64
}

// GenerateRandomFloat64 returns a random float64 in [start, end).
//
// If no weights are provided, the distribution is uniform.
// If weights are provided, one of the weighted ranges is selected
// proportional to its Weight, and then a value is chosen uniformly
// within that range.
func GenerateRandomFloat64(start, end float64, weights ...RangeWeight) float64 {
	if start > end {
		start, end = end, start
	}
	if start == end {
		return start
	}
	if len(weights) == 0 {
		return start + rand.Float64()*(end-start)
	}

	totalWeight := 0.0
	for _, w := range weights {
		if w.Weight <= 0 {
			continue
		}

		// Clamp to overall range.
		if w.Start < start {
			w.Start = start
		}
		if w.End > end {
			w.End = end
		}

		if w.Start >= w.End {
			continue
		}

		totalWeight += w.Weight
	}

	if totalWeight == 0 {
		return start + rand.Float64()*(end-start)
	}

	r := rand.Float64() * totalWeight

	cumulative := 0.0
	for _, w := range weights {
		if w.Weight <= 0 {
			continue
		}

		if w.Start < start {
			w.Start = start
		}
		if w.End > end {
			w.End = end
		}

		if w.Start >= w.End {
			continue
		}

		cumulative += w.Weight
		if r < cumulative {
			return w.Start + rand.Float64()*(w.End-w.Start)
		}
	}

	// Fallback due to floating-point precision.
	last := weights[len(weights)-1]
	if last.Start < start {
		last.Start = start
	}
	if last.End > end {
		last.End = end
	}

	return last.Start + rand.Float64()*(last.End-last.Start)
}

// ChooseOneRandomly returns one of the given items, chosen uniformly at
// random. Returns the zero value of T if items is empty.
func ChooseOneRandomly[T any](items ...T) T {
	if len(items) == 0 {
		var zero T
		return zero
	}

	return items[rand.IntN(len(items))]
}
