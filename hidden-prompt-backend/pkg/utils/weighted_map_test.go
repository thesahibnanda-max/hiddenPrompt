package utils_test

import (
	"testing"

	"hidden-prompt-backend/pkg/utils"

	"github.com/stretchr/testify/require"
)

func mustNewWeightedMap(t *testing.T, weights map[string]uint) utils.ThreadSafeWeightedMap[string] {
	t.Helper()
	wm, err := utils.NewThreadSafeWeightedMap(weights)
	require.NoError(t, err)
	return wm
}

func Test_NewThreadSafeWeightedMap_ErrorsOnEmptyMap(t *testing.T) {
	_, err := utils.NewThreadSafeWeightedMap(map[string]uint{})
	require.Error(t, err)
}

func Test_NewThreadSafeWeightedMap_ErrorsOnNoPositiveWeight(t *testing.T) {
	_, err := utils.NewThreadSafeWeightedMap(map[string]uint{"a": 0, "b": 0})
	require.Error(t, err)
}

func Test_NewThreadSafeWeightedMap_IgnoresZeroWeights(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"a": 5, "b": 0, "c": 0})

	for range 200 {
		require.Equal(t, "a", wm.GetRandom(), "only the single positive-weight key should ever be returned")
	}
}

func Test_GetRandom_DistributionRoughlyMatchesWeights(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"heavy": 90, "light": 10})

	const draws = 20000
	counts := map[string]int{}
	for range draws {
		counts[wm.GetRandom()]++
	}

	// Not an exact check (it's random) - just confirms the heavy key
	// dominates roughly proportionally rather than e.g. a 50/50 split.
	require.Greater(t, counts["heavy"], counts["light"]*5)
	require.Equal(t, draws, counts["heavy"]+counts["light"])
}

func Test_GetRandomAvoiding_NoKeysToAvoid(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1})
	require.Equal(t, "a", wm.GetRandomAvoiding())
}

func Test_GetRandomAvoiding_SingleKeyNeverReturnsIt(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1, "b": 1, "c": 1})

	for range 300 {
		require.NotEqual(t, "a", wm.GetRandomAvoiding("a"))
	}
}

func Test_GetRandomAvoiding_MultipleKeysNeverReturnsAnyOfThem(t *testing.T) {
	// This is the exact bug being fixed: the previous implementation's
	// loop overwrote its own result every iteration, so only the last
	// avoided key that happened to have a submap/leaf was ever actually
	// excluded - every earlier key in the list silently slipped through.
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1})

	for range 500 {
		got := wm.GetRandomAvoiding("a", "b", "c")
		require.NotEqual(t, "a", got)
		require.NotEqual(t, "b", got)
		require.NotEqual(t, "c", got)
		require.Contains(t, []string{"d", "e"}, got)
	}
}

func Test_GetRandomAvoiding_DuplicateKeysInAvoidList(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1, "b": 1, "c": 1})

	for range 200 {
		got := wm.GetRandomAvoiding("a", "a", "a")
		require.NotEqual(t, "a", got)
		require.Contains(t, []string{"b", "c"}, got)
	}
}

func Test_GetRandomAvoiding_IgnoresKeysNotInMap(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1, "b": 1})

	for range 200 {
		got := wm.GetRandomAvoiding("not-a-real-key", "also-not-real")
		require.Contains(t, []string{"a", "b"}, got)
	}
}

func Test_GetRandomAvoiding_FallsBackWhenAvoidingTheOnlyKey(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"only": 1})

	for range 50 {
		require.Equal(t, "only", wm.GetRandomAvoiding("only"))
	}
}

func Test_GetRandomAvoiding_FallsBackWhenAvoidingEveryKey(t *testing.T) {
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1, "b": 1, "c": 1})

	for range 100 {
		got := wm.GetRandomAvoiding("a", "b", "c")
		require.Contains(t, []string{"a", "b", "c"}, got, "falling back to unrestricted selection must still return a valid key")
	}
}

func Test_GetRandomAvoiding_SiblingLeavesUnderSameParent(t *testing.T) {
	// Exercises the dirty-set's shared-ancestor early-stop and the
	// child-collapse logic in rebuildExcluding together: with 4 keys built
	// two-by-two, "a" and "b" are very likely paired under the same
	// immediate parent - avoiding both at once must still correctly
	// collapse that subtree away rather than leaving a stale node behind.
	wm := mustNewWeightedMap(t, map[string]uint{"a": 1, "b": 1, "c": 1, "d": 1})

	for range 300 {
		got := wm.GetRandomAvoiding("a", "b")
		require.NotEqual(t, "a", got)
		require.NotEqual(t, "b", got)
		require.Contains(t, []string{"c", "d"}, got)
	}
}
