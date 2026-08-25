package hrw

import "math"

// Assign maps every key to a node, bounding each node's load at
// ceil(len(keys)/len(nodes) * (1+epsilon)). Keys and nodes are deduped;
// the result is a pure function of the inputs. O(k*n).
func Assign(keys, nodes []string, opts ...Option) (map[string]string, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyNodes
	}

	c := newConfig(opts)

	if c.epsilon < 0 {
		return nil, ErrBadEpsilon
	}

	keys = dedup(keys)
	nodes = dedup(nodes)

	cap := int(math.Ceil(float64(len(keys)) / float64(len(nodes)) * (1 + c.epsilon)))

	out := make(map[string]string, len(keys))
	load := make(map[string]int, len(nodes))

	for _, k := range keys {
		best := ""
		var bestScore uint64
		first := true

		for _, n := range nodes {
			if load[n] >= cap {
				continue
			}

			s := c.hash(k, n)

			// Tie-break on node name keeps assignment deterministic.
			if first || s > bestScore || (s == bestScore && n < best) {
				best, bestScore = n, s
				first = false
			}
		}

		out[k] = best
		load[best]++
	}

	return out, nil
}
