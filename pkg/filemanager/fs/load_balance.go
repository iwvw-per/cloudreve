package fs

import "sort"

// MaxLoadBalanceDepth is the maximum nesting depth allowed when resolving a
// load_balance storage policy to a concrete child policy.
const MaxLoadBalanceDepth = 10

// ResolveLoadBalancePolicy selects a child storage policy ID from the given
// weights. Each key is a child storage policy ID, and its value is the
// relative weight of that child. The total weight is the sum of all positive
// weights; a random value in [0, total) decides which child is selected. If
// no positive weight is configured (all weights are zero or negative),
// children are selected uniformly at random. An empty weight map returns 0.
//
// The rand function must return a non-negative integer and is the only source
// of randomness, which keeps the function deterministic and testable.
func ResolveLoadBalancePolicy(weights map[int]int, rand func() int) int {
	if len(weights) == 0 {
		return 0
	}

	ids := make([]int, 0, len(weights))
	total := 0
	for id, w := range weights {
		ids = append(ids, id)
		if w > 0 {
			total += w
		}
	}
	sort.Ints(ids)

	if total <= 0 {
		return ids[rand()%len(ids)]
	}

	roll := rand() % total
	for _, id := range ids {
		w := weights[id]
		if w <= 0 {
			continue
		}
		if roll < w {
			return id
		}
		roll -= w
	}

	// Unreachable when total > 0, kept as a safe fallback.
	return ids[0]
}
