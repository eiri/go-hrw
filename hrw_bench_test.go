package hrw

import (
	"strconv"
	"testing"
)

func benchNodes(n int) []string {
	nodes := make([]string, n)
	for i := range nodes {
		nodes[i] = "node-" + string(rune('a'+i%26)) + strconv.Itoa(i/26%100)
	}

	return nodes
}

func BenchmarkOwner(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		nodes := benchNodes(n)

		b.Run("n"+strconv.Itoa(n), func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				_, _ = Owner(strconv.Itoa(i), nodes)
			}
		})
	}
}

func BenchmarkSkeletonOwner(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		s, err := Build(benchNodes(n))
		if err != nil {
			b.Fatal(err)
		}

		b.Run("n"+strconv.Itoa(n), func(b *testing.B) {
			for i := 0; b.Loop(); i++ {
				_, _ = s.Owner(strconv.Itoa(i))
			}
		})
	}
}
