package hrw

import (
	"math"
	"strconv"
)

// Skeleton is a precomputed clustered tree enabling O(log n) lookups.
// Plain data; build once with Build or BuildW, reuse across calls.
type Skeleton struct {
	clusters      [][]string
	weights       [][]int
	weighted      bool
	branchWeights []map[int]int
	fanout        int
	levels        int
}

const defaultClusterSize = 16

// Build precomputes a Skeleton for O(log n) lookups.
func Build(nodes []string, opts ...Option) (*Skeleton, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyNodes
	}

	c := newConfig(opts)
	nodes = dedup(nodes)

	size := defaultClusterSize
	if c.clusterSize > 0 {
		size = c.clusterSize
	}

	groups := chunkIndexes(len(nodes), size)

	clusters := make([][]string, len(groups))

	for gi, g := range groups {
		clusters[gi] = make([]string, len(g))

		for li, idx := range g {
			clusters[gi][li] = nodes[idx]
		}
	}

	fanout := c.fanout
	if fanout == 0 {
		fanout = optimalFanout(len(clusters))
	}

	return &Skeleton{
		clusters: clusters,
		weighted: false,
		fanout:   fanout,
		levels:   levelsFor(len(clusters), fanout),
	}, nil
}

// chunkIndexes splits n indexes into contiguous groups of size, spreading
// an undersized tail round-robin so no group is left undersized.
func chunkIndexes(n, size int) [][]int {
	if n <= size {
		all := make([]int, n)

		for i := range all {
			all[i] = i
		}

		return [][]int{all}
	}

	fullCount := n / size
	groups := make([][]int, 0, fullCount+1)

	for i := range fullCount {
		g := make([]int, 0, size)

		for j := i * size; j < (i+1)*size; j++ {
			g = append(g, j)
		}

		groups = append(groups, g)
	}

	if n%size == 0 {
		return groups
	}

	// Spread tail indexes over existing groups so none is undersized.
	for j := fullCount * size; j < n; j++ {
		groups[j%len(groups)] = append(groups[j%len(groups)], j)
	}

	return groups
}

// BuildW precomputes a weighted Skeleton. Weights bias both tree routing
// (via per-branch totals) and in-cluster leaf selection.
func BuildW(nodes []WeightedNode, opts ...Option) (*Skeleton, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyNodes
	}

	for _, n := range nodes {
		if n.Weight <= 0 {
			return nil, ErrBadWeights
		}
	}

	c := newConfig(opts)
	sorted := dedupWeighted(nodes)

	size := defaultClusterSize
	if c.clusterSize > 0 {
		size = c.clusterSize
	}

	groups := chunkIndexes(len(sorted), size)

	clusters := make([][]string, len(groups))
	weights := make([][]int, len(groups))

	for gi, g := range groups {
		clusters[gi] = make([]string, len(g))
		weights[gi] = make([]int, len(g))

		for li, idx := range g {
			clusters[gi][li] = sorted[idx].Node
			weights[gi][li] = sorted[idx].Weight
		}
	}

	totals := make([]int, len(clusters))

	for i, ws := range weights {
		for _, w := range ws {
			totals[i] += w
		}
	}

	fanout := c.fanout
	if fanout == 0 {
		fanout = optimalFanout(len(clusters))
	}

	levels := levelsFor(len(clusters), fanout)

	return &Skeleton{
		clusters:      clusters,
		weights:       weights,
		weighted:      true,
		branchWeights: branchWeights(totals, fanout, levels),
		fanout:        fanout,
		levels:        levels,
	}, nil
}

// branchWeights maps each level's branch prefix to the total node weight
// under it. Branches absent from a map carry no nodes and never win.
func branchWeights(clusterTotals []int, fanout, levels int) []map[int]int {
	maps := make([]map[int]int, levels)

	for lvl := range maps {
		maps[lvl] = map[int]int{}
	}

	for index, total := range clusterTotals {
		for lvl := range levels {
			divisor := powInt(fanout, levels-lvl-1)
			maps[lvl][index/divisor] += total
		}
	}

	return maps
}

func powInt(base, exp int) int {
	r := 1
	for range exp {
		r *= base
	}

	return r
}
func optimalFanout(clusterCount int) int {
	if clusterCount <= 2 {
		return 2
	}

	best, bestCost := 0, math.Inf(1)

	for f := 2; f <= 8; f++ {
		lv := levelsFor(clusterCount, f)
		capacity := math.Pow(float64(f), float64(lv))
		overflow := (capacity - float64(clusterCount)) / capacity
		cost := float64(f*lv) / (1 - overflow)

		if cost < bestCost {
			best, bestCost = f, cost
		}
	}

	return best
}

func levelsFor(clusterCount, fanout int) int {
	if clusterCount <= 1 {
		return 0
	}

	return int(math.Ceil(math.Log(float64(clusterCount)) / math.Log(float64(fanout))))
}

// Owner routes key through the tree to a cluster, then picks the
// best-scoring node in it. O(log n).
func (s *Skeleton) Owner(key string) (string, error) {
	// Fast path: one cluster, direct scan.
	if len(s.clusters) == 1 {
		return s.pickInCluster(key, 0), nil
	}

	return s.ownerSalted(key, 0)
}

func (s *Skeleton) ownerSalted(key string, salt int) (string, error) {
	prefix := 0

	// Descend level by level, greedily picking the best branch digit.
	for level := range s.levels {
		digit := s.bestDigit(key, salt, level, prefix)
		prefix = prefix*s.fanout + digit
	}

	if prefix < len(s.clusters) {
		return s.pickInCluster(key+saltKey(salt)+clusterKey(prefix), prefix), nil
	}

	// Routed past the last cluster; rehash with a fresh salt.
	return s.ownerSalted(key, salt+1)
}

// pickInCluster selects the winning node inside one cluster.
func (s *Skeleton) pickInCluster(key string, ci int) string {
	cluster := s.clusters[ci]

	if !s.weighted {
		return maxBy(cluster, func(node string) uint64 {
			return defaultHash(key, node)
		})
	}

	var scorer Weighted
	bestIdx, bestScore := 0, -1.0

	for i, node := range cluster {
		if sc := scorer.score(defaultHash(key, node), s.weights[ci][i]); sc > bestScore {
			bestIdx, bestScore = i, sc
		}
	}

	return cluster[bestIdx]
}

// bestDigit picks the routing digit at one level; weighted skeletons score
// digits by their total branch weight raised into hash01 space.
func (s *Skeleton) bestDigit(key string, salt, level, prefix int) int {
	if !s.weighted {
		return maxDigit(key, salt, level, prefix, s.fanout)
	}

	var scorer Weighted
	weights := s.branchWeights[level]
	best, bestScore := 0, -1.0

	for d := range s.fanout {
		bw := weights[prefix*s.fanout+d]
		if bw == 0 {
			continue // empty branch never wins
		}

		score := scorer.score(hashRoute(key, salt, level, prefix, d), bw)
		if score > bestScore {
			best, bestScore = d, score
		}
	}

	return best
}

func maxDigit(key string, salt, level, prefix, fanout int) int {
	best := 0
	bestScore := hashRoute(key, salt, level, prefix, best)

	for d := 1; d < fanout; d++ {
		if score := hashRoute(key, salt, level, prefix, d); score > bestScore {
			best, bestScore = d, score
		}
	}

	return best
}

func hashRoute(key string, salt, level, prefix, digit int) uint64 {
	return defaultHash(
		key+"|"+saltKey(salt)+"|"+levelKey(level)+"|"+prefixKey(prefix),
		levelKey(digit),
	)
}

func saltKey(salt int) string      { return intKey("s", salt) }
func levelKey(level int) string    { return intKey("l", level) }
func prefixKey(prefix int) string  { return intKey("p", prefix) }
func clusterKey(prefix int) string { return prefixKey(prefix) }

func intKey(tag string, v int) string {
	return tag + strconv.Itoa(v)
}

func maxBy(nodes []string, score func(string) uint64) string {
	best := nodes[0]
	bestScore := score(best)

	for _, n := range nodes[1:] {
		if s := score(n); s > bestScore {
			best, bestScore = n, s
		}
	}

	return best
}
