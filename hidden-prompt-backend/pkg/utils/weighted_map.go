package utils

import (
	"fmt"
	"math/rand/v2"
)

// ThreadSafeWeightedMap picks a random key from a fixed set, weighted by
// each key's configured weight. Safe for concurrent use because the
// underlying tree is never mutated after construction - GetRandomAvoiding
// builds new nodes for any excluded subtree rather than modifying shared
// ones.
type ThreadSafeWeightedMap[K comparable] interface {
	// GetRandom is O(log n) time, O(1) extra space.
	GetRandom() K
	// GetRandomAvoiding excludes every given key (any not present in the
	// map are ignored) and picks randomly among what's left. It's
	// O(k log n) time and space for k avoided keys - only the nodes on an
	// avoided leaf's path to the root are ever copied, everything else is
	// shared unchanged. Falls back to unrestricted GetRandom if avoiding
	// every given key would leave nothing to pick from.
	GetRandomAvoiding(avoiding ...K) K
}

// NewThreadSafeWeightedMap builds a ThreadSafeWeightedMap from a map of
// key -> weight. Keys with a weight of 0 are excluded. Returns an error if
// no key has a positive weight, so a malformed map is caught explicitly by
// the caller at construction time rather than surfacing later as a
// meaningless zero-value result from GetRandom.
func NewThreadSafeWeightedMap[K comparable](weights map[K]uint) (ThreadSafeWeightedMap[K], error) {
	root, leaves := buildTree(weights)
	if root == nil {
		return nil, fmt.Errorf("weightedmap: at least one key with a positive weight is required")
	}
	return &weightedMap[K]{rootNode: root, leaves: leaves}, nil
}

// node is a balanced binary tree node: an internal node always has both
// children set and its weight is their sum; a leaf always has neither
// child set and carries a real key. This invariant is what both
// pickRandom and rebuildExcluding rely on - it's maintained by
// construction and preserved by every exclusion rebuild.
type node[K comparable] struct {
	weight uint
	parent *node[K]
	left   *node[K]
	right  *node[K]
	key    K
}

func (n *node[K]) isLeaf() bool {
	return n.left == nil && n.right == nil
}

// pickRandom descends from n choosing left/right based on r, which must
// satisfy 0 <= r < n.weight. O(log n) time, O(1) extra space.
func (n *node[K]) pickRandom(r uint64) K {
	cur := n
	for !cur.isLeaf() {
		if r < uint64(cur.left.weight) {
			cur = cur.left
			continue
		}
		r -= uint64(cur.left.weight)
		cur = cur.right
	}
	return cur.key
}

// buildTree builds a balanced binary tree over every positive-weight key
// in weights, bottom-up: pair adjacent nodes into parents, carry an odd
// one out forward unpaired, repeat until one root remains. This gives
// every leaf O(log n) depth regardless of the weight distribution -
// O(n) time and space. Returns a nil root for an empty/all-zero-weight
// input.
func buildTree[K comparable](weights map[K]uint) (*node[K], map[K]*node[K]) {
	leaves := make(map[K]*node[K], len(weights))
	level := make([]*node[K], 0, len(weights))

	for key, weight := range weights {
		if weight == 0 {
			continue
		}
		n := &node[K]{key: key, weight: weight}
		level = append(level, n)
		leaves[key] = n
	}

	if len(level) == 0 {
		return nil, leaves
	}

	for len(level) > 1 {
		next := make([]*node[K], 0, (len(level)+1)/2)

		for i := 0; i < len(level); i += 2 {
			if i+1 >= len(level) {
				next = append(next, level[i])
				continue
			}

			left, right := level[i], level[i+1]
			parent := &node[K]{
				weight: left.weight + right.weight,
				left:   left,
				right:  right,
			}
			left.parent = parent
			right.parent = parent

			next = append(next, parent)
		}

		level = next
	}

	return level[0], leaves
}

// markDirty walks each leaf up to the root via its parent link, marking
// every ancestor whose weight changes as a result of excluding these
// leaves. Stops early the moment it reaches an already-marked ancestor -
// it and everything above it were already added by an earlier leaf, so
// there's nothing more to gain from continuing up this path.
// O(k log n) time and space for k leaves.
func markDirty[K comparable](leaves []*node[K]) map[*node[K]]struct{} {
	dirty := make(map[*node[K]]struct{})
	for _, leaf := range leaves {
		for cur := leaf; cur != nil; cur = cur.parent {
			if _, already := dirty[cur]; already {
				break
			}
			dirty[cur] = struct{}{}
		}
	}
	return dirty
}

// rebuildExcluding returns a tree equal to the one rooted at n but with
// every dirty leaf removed, sharing any subtree untouched by the
// exclusion (i.e. only copying nodes on the path from a dirty leaf to the
// root) rather than rebuilding the whole tree. Returns nil if n's entire
// subtree is excluded. Bounded by len(dirty) - O(k log n) time and space
// for k avoided keys, since dirty was built by markDirty above.
func rebuildExcluding[K comparable](n *node[K], dirty map[*node[K]]struct{}) *node[K] {
	if n == nil {
		return nil
	}
	if _, isDirty := dirty[n]; !isDirty {
		return n
	}
	if n.isLeaf() {
		return nil
	}

	left := rebuildExcluding(n.left, dirty)
	right := rebuildExcluding(n.right, dirty)

	switch {
	case left == nil && right == nil:
		return nil
	case left == nil:
		return right
	case right == nil:
		return left
	default:
		return &node[K]{weight: left.weight + right.weight, left: left, right: right}
	}
}

type weightedMap[K comparable] struct {
	rootNode *node[K]
	leaves   map[K]*node[K]
}

func (w *weightedMap[K]) GetRandom() K {
	return w.rootNode.pickRandom(rand.Uint64N(uint64(w.rootNode.weight)))
}

func (w *weightedMap[K]) GetRandomAvoiding(avoiding ...K) K {
	if len(avoiding) == 0 {
		return w.GetRandom()
	}

	leaves := make([]*node[K], 0, len(avoiding))
	for _, avoid := range avoiding {
		if leaf, ok := w.leaves[avoid]; ok {
			leaves = append(leaves, leaf)
		}
	}
	if len(leaves) == 0 {
		return w.GetRandom()
	}

	newRoot := rebuildExcluding(w.rootNode, markDirty(leaves))
	if newRoot == nil {
		// Avoiding every given key left nothing to pick from - fall back
		// to unrestricted selection rather than a meaningless result.
		return w.GetRandom()
	}
	return newRoot.pickRandom(rand.Uint64N(uint64(newRoot.weight)))
}
