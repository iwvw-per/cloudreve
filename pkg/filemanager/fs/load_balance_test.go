package fs

import "testing"

func TestResolveLoadBalancePolicy(t *testing.T) {
	tests := []struct {
		name     string
		weights  map[int]int
		randSeq  []int
		expected []int
	}{
		{
			name:     "empty weights returns 0",
			weights:  map[int]int{},
			randSeq:  []int{0},
			expected: []int{0},
		},
		{
			name:     "weighted selection picks first child on low roll",
			weights:  map[int]int{1: 1, 2: 3, 3: 6},
			randSeq:  []int{0},
			expected: []int{1}, // roll in [0, 1)
		},
		{
			name:     "weighted selection lands on second child",
			weights:  map[int]int{1: 1, 2: 3, 3: 6},
			randSeq:  []int{1, 2, 3},
			expected: []int{2, 2, 2}, // roll in [1, 4)
		},
		{
			name:     "weighted selection lands on third child",
			weights:  map[int]int{1: 1, 2: 3, 3: 6},
			randSeq:  []int{4, 9},
			expected: []int{3, 3}, // roll in [4, 10)
		},
		{
			name:     "all zero weights fall back to uniform random",
			weights:  map[int]int{1: 0, 2: 0, 3: 0},
			randSeq:  []int{0, 1, 2, 3},
			expected: []int{1, 2, 3, 1}, // ids sorted [1,2,3], index = rand % 3
		},
		{
			name:     "negative weights are ignored",
			weights:  map[int]int{1: -1, 2: 2},
			randSeq:  []int{1},
			expected: []int{2},
		},
		{
			name:     "single child always selected",
			weights:  map[int]int{7: 100},
			randSeq:  []int{0, 99},
			expected: []int{7, 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, r := range tt.randSeq {
				got := ResolveLoadBalancePolicy(tt.weights, func() int { return r })
				if got != tt.expected[i] {
					t.Fatalf("rand=%d: got %d, want %d", r, got, tt.expected[i])
				}
			}
		})
	}
}

func TestResolveLoadBalancePolicyDistribution(t *testing.T) {
	// With equal weights, a uniform random source over the total weight range
	// must produce each child roughly the same number of times.
	weights := map[int]int{1: 1, 2: 1, 3: 1}
	counts := map[int]int{}
	const samples = 30000
	for i := 0; i < samples; i++ {
		counts[ResolveLoadBalancePolicy(weights, func() int { return i })]++
	}

	for id, w := range weights {
		if w <= 0 {
			t.Fatalf("unexpected child %d", id)
		}
		share := float64(counts[id]) / float64(samples)
		if share < 0.3 || share > 0.37 {
			t.Fatalf("child %d got %v of samples, want ~0.333", id, share)
		}
	}
}
