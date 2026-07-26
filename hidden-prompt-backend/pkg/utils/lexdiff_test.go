package utils_test

import (
	"testing"

	"hidden-prompt-backend/pkg/utils"

	"github.com/stretchr/testify/require"
)

func Test_MissingContentWords_RealOatmealExample(t *testing.T) {
	// The actual report that motivated this helper: the guess covers
	// "breakfast" but never engages with "usually" (habitual framing) or
	// "eat" - those are exactly the words that should survive as missing.
	target := "What type of food do you usually eat for breakfast?"
	given := "Best meal for breakfast"

	missing := utils.MissingContentWords(target, given, 4)

	require.Contains(t, missing, "usually", "the one word that actually distinguishes habitual framing from a superlative must survive")
	require.Contains(t, missing, "eat")
	require.NotContains(t, missing, "breakfast", "shared between target and given, must not be reported as missing")
}

func Test_MissingContentWords_FullOverlapIsEmpty(t *testing.T) {
	missing := utils.MissingContentWords("What is the largest planet?", "What is the largest planet", 4)
	require.Empty(t, missing)
}

func Test_MissingContentWords_PreservesFrequencyAndSuperlativeWords(t *testing.T) {
	// These are exactly the words a generic/aggressive stopword list would
	// wrongly strip, even though they carry the crucial distinguishing
	// framing in this game's questions.
	for _, word := range []string{"usually", "often", "always", "best", "most", "largest", "specific", "main", "typical"} {
		target := "What is the " + word + " example here?"
		missing := utils.MissingContentWords(target, "completely unrelated guess text", 10)
		require.Containsf(t, missing, word, "%q must survive stopword filtering", word)
	}
}

func Test_MissingContentWords_RespectsMax(t *testing.T) {
	target := "alpha bravo charlie delta echo foxtrot"
	missing := utils.MissingContentWords(target, "", 3)
	require.Len(t, missing, 3)
	require.Equal(t, []string{"alpha", "bravo", "charlie"}, missing)
}

func Test_MissingContentWords_StripsFunctionWords(t *testing.T) {
	missing := utils.MissingContentWords("What is the largest planet in our solar system?", "irrelevant", 10)
	for _, fn := range []string{"what", "is", "the", "in", "our"} {
		require.NotContainsf(t, missing, fn, "%q is a function word and should be filtered", fn)
	}
	require.Contains(t, missing, "largest")
	require.Contains(t, missing, "planet")
}
