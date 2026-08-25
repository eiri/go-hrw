package hrw

import "math"

// WeightedNode pairs a node with a positive weight.
type WeightedNode struct {
	Node   string
	Weight int
}

const hash01Max = float64(math.MaxUint64)

// Weighted implements weighted rendezvous scoring:
// score = pow(hash01, 1/weight). Higher weight biases keys toward the node.
type Weighted struct{}

// score maps a 64-bit hash into [0,1) and exponentiates by 1/weight.
func (Weighted) score(hash uint64, weight int) float64 {
	return math.Pow(float64(hash)/hash01Max, 1.0/float64(weight))
}
