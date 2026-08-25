package hrw

import (
	"cmp"
	"errors"
	"slices"
	"strings"
)

var (
	ErrEmptyNodes = errors.New("hrw: empty node list")
	ErrBadCount   = errors.New("hrw: negative count")
	ErrBadWeights = errors.New("hrw: weights must be positive")
	ErrBadEpsilon = errors.New("hrw: epsilon must be >= 0")
)

// Option configures scoring and skeleton construction.
type Option func(*config)

type config struct {
	hash        func(key, node string) uint64
	fanout      int
	clusterSize int
	epsilon     float64
}

func newConfig(opts []Option) config {
	c := config{
		hash:    defaultHash,
		epsilon: 0.25,
	}

	for _, opt := range opts {
		opt(&c)
	}

	return c
}

// Owner returns the node owning key. O(n) over nodes.
func Owner(key string, nodes []string, opts ...Option) (string, error) {
	if len(nodes) == 0 {
		return "", ErrEmptyNodes
	}

	c := newConfig(opts)

	nodes = dedup(nodes)

	best := nodes[0]
	bestScore := c.hash(key, best)

	for _, n := range nodes[1:] {
		if s := c.hash(key, n); s > bestScore {
			best, bestScore = n, s
		}
	}

	return best, nil
}

// dedup returns sorted copy of nodes without duplicates.
func dedup(nodes []string) []string {
	out := slices.Clone(nodes)
	slices.Sort(out)

	return slices.Compact(out)
}

// OwnerW returns the weighted owner of key. O(n) over nodes.
func OwnerW(key string, nodes []WeightedNode, opts ...Option) (string, error) {
	if len(nodes) == 0 {
		return "", ErrEmptyNodes
	}

	c := newConfig(opts)

	for _, n := range nodes {
		if n.Weight <= 0 {
			return "", ErrBadWeights
		}
	}

	nodes = dedupWeighted(nodes)
	var scorer Weighted
	best := nodes[0]
	bestScore := scorer.score(c.hash(key, best.Node), best.Weight)

	for _, n := range nodes[1:] {
		if s := scorer.score(c.hash(key, n.Node), n.Weight); s > bestScore {
			best, bestScore = n, s
		}
	}

	return best.Node, nil
}

// dedupWeighted returns a copy sorted by node name without duplicates.
func dedupWeighted(nodes []WeightedNode) []WeightedNode {
	out := slices.Clone(nodes)

	slices.SortFunc(out, func(a, b WeightedNode) int {
		return strings.Compare(a.Node, b.Node)
	})

	return slices.CompactFunc(out, func(a, b WeightedNode) bool {
		return a.Node == b.Node
	})
}

// Owners returns the top-n nodes for key, best first. O(n log n).
func Owners(key string, nodes []string, n int, opts ...Option) ([]string, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyNodes
	}

	if n < 0 {
		return nil, ErrBadCount
	}

	c := newConfig(opts)

	nodes = dedup(nodes)

	type ranked struct {
		node  string
		score uint64
	}

	rs := make([]ranked, len(nodes))

	for i, nd := range nodes {
		rs[i] = ranked{nd, c.hash(key, nd)}
	}

	slices.SortFunc(rs, func(a, b ranked) int {
		switch {
		case a.score != b.score:
			return cmp.Compare(b.score, a.score)
		default:
			return strings.Compare(a.node, b.node)
		}
	})

	if n > len(rs) {
		n = len(rs)
	}

	out := make([]string, n)

	for i, r := range rs[:n] {
		out[i] = r.node
	}

	return out, nil
}

// WithFanout sets the skeleton branching factor (default: auto).
func WithFanout(n int) Option {
	return func(c *config) { c.fanout = n }
}

// WithClusterSize sets the skeleton cluster size (default 16).
func WithClusterSize(n int) Option {
	return func(c *config) { c.clusterSize = n }
}

// WithEpsilon sets the load slack factor for Assign (default 0.25).
func WithEpsilon(e float64) Option {
	return func(c *config) { c.epsilon = e }
}
