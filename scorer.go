package hrw

import "hash/fnv"

// defaultHash combines key and node into a 64-bit score.
// NUL byte separates the parts so (a, bc) != (ab, c).
// A splitmix64 finalizer decorrelates outputs: FNV alone differs only
// in low bits for messages sharing a long prefix, which biases max-by.
func defaultHash(key, node string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	h.Write([]byte{0})
	h.Write([]byte(node))

	z := h.Sum64()

	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31

	return z
}
