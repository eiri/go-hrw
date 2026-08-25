package hrw

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestChunkIndexesExactMultiple(t *testing.T) {
	got := chunkIndexes(4, 2)
	want := [][]int{{0, 1}, {2, 3}}

	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("group %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestChunkIndexesUndersizedTail(t *testing.T) {
	// 7 indexes of size 3 -> groups of 4 and 3; no tiny tail group.
	got := chunkIndexes(7, 3)

	if len(got) != 2 {
		t.Fatalf("groups = %d, want 2", len(got))
	}

	sizes := []int{len(got[0]), len(got[1])}

	total := 0
	for _, g := range got {
		total += len(g)
	}

	if total != 7 {
		t.Errorf("total = %d, want 7", total)
	}

	if sizes[0]-sizes[1] > 1 {
		t.Errorf("unbalanced sizes %v", sizes)
	}
}

func TestChunkIndexesSingleGroup(t *testing.T) {
	got := chunkIndexes(2, 16)

	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Errorf("got %v, want one group [0 1]", got)
	}
}

func TestOptimalFanout(t *testing.T) {
	tests := []struct {
		clusters int
		want     int
	}{
		{1, 2},
		// Cost model: f*levels/(1-overflow); f=2 ties f=4 at C=10, first wins.
		{10, 2},
	}

	for _, tt := range tests {
		got := optimalFanout(tt.clusters)
		if got < 2 || got > 8 {
			t.Fatalf("fanout(%d) = %d, out of range", tt.clusters, got)
		}

		if tt.want != 0 && got != tt.want {
			t.Errorf("fanout(%d) = %d, want %d", tt.clusters, got, tt.want)
		}
	}
}

func TestBuildErrors(t *testing.T) {
	if _, err := Build(nil); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("got %v, want ErrEmptyNodes", err)
	}
}

func TestBuildDefaults(t *testing.T) {
	s, err := Build([]string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}

	if s.fanout == 0 || s.levels < 0 || len(s.clusters) == 0 {
		t.Errorf("bad skeleton: fanout=%d levels=%d clusters=%d",
			s.fanout, s.levels, len(s.clusters))
	}

	count := 0
	for _, cl := range s.clusters {
		count += len(cl)
	}
	if count != 3 {
		t.Errorf("skeleton holds %d nodes, want 3", count)
	}
}

func TestBuildCustomFanout(t *testing.T) {
	s, err := Build(makeNodes(100), WithFanout(5), WithClusterSize(4))
	if err != nil {
		t.Fatal(err)
	}

	if s.fanout != 5 {
		t.Errorf("fanout = %d, want 5", s.fanout)
	}

	if len(s.clusters) != 25 { // ceil(100/4)
		t.Errorf("clusters = %d, want 25", len(s.clusters))
	}
}

func makeNodes(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "node" + string(rune('a'+i%26)) + string(rune('0'+i/26))
	}

	return out
}

func TestSkeletonOwnerSingleClusterMatchesLinear(t *testing.T) {
	nodes := []string{"a", "b", "c", "d", "e"}

	s, err := Build(nodes, WithClusterSize(16))
	if err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(21, 22))

	for range 200 {
		key := "k" + string(rune(rng.IntN(1000)))
		want, _ := Owner(key, nodes)
		got, err := s.Owner(key)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("key %q: skeleton=%s linear=%s", key, got, want)
		}
	}
}

func TestSkeletonOwnerDeterministic(t *testing.T) {
	s, err := Build(makeNodes(50))
	if err != nil {
		t.Fatal(err)
	}

	first, err := s.Owner("stable-key")
	if err != nil {
		t.Fatal(err)
	}

	for range 10 {
		if got, _ := s.Owner("stable-key"); got != first {
			t.Fatalf("got %q after %q", got, first)
		}
	}
}

func TestSkeletonOwnerInNodes(t *testing.T) {
	nodes := makeNodes(100)

	s, err := Build(nodes)
	if err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(31, 32))

	for range 500 {
		key := "key" + string(rune(rng.IntN(9999)))
		got, err := s.Owner(key)
		if err != nil {
			t.Fatal(err)
		}

		found := slices.Contains(nodes, got)

		if !found {
			t.Fatalf("owner %q not in node set", got)
		}
	}
}

func TestSkeletonOwnerSaltEscalation(t *testing.T) {
	// Force index overflow: 5 clusters, fanout 2 -> levels=3, capacity 8.
	// Some salts must escalate; result must still be valid.
	nodes := makeNodes(20)

	s, err := Build(nodes, WithClusterSize(4), WithFanout(2))
	if err != nil {
		t.Fatal(err)
	}

	for i := range 1000 {
		got, err := s.Owner("overflow" + string(rune(i)))
		if err != nil {
			t.Fatal(err)
		}

		if got == "" {
			t.Fatalf("empty owner for key %d", i)
		}
	}
}

func TestSkeletonOwnerDistribution(t *testing.T) {
	nodes := makeNodes(64)

	s, err := Build(nodes)
	if err != nil {
		t.Fatal(err)
	}

	counts := make(map[string]int)

	total := 6400
	for i := range total {
		owner, err := s.Owner("load" + string(rune(i)))
		if err != nil {
			t.Fatal(err)
		}
		counts[owner]++
	}

	// Even distribution: every node gets a share, none dominates.
	expected := total / len(nodes)

	for _, c := range counts {
		if c > expected*3 || c < expected/3 {
			t.Fatalf("uneven load: expected ~%d per node, saw %d", expected, c)
		}
	}

	if len(counts) < len(nodes)*9/10 {
		t.Errorf("only %d of %d nodes ever won a key", len(counts), len(nodes))
	}
}
