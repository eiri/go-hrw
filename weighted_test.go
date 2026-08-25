package hrw

import (
	"errors"
	"math/rand/v2"
	"testing"
)

func TestOwnerWGolden(t *testing.T) {
	nodes := []WeightedNode{{Node: "a", Weight: 1}, {Node: "b", Weight: 1}, {Node: "c", Weight: 10}}

	got, err := OwnerW("192.168.0.1", nodes)
	if err != nil {
		t.Fatal(err)
	}

	// Winner for this key per independent oracle; weight-10 "c" scores 0.815
	// but "a" scores 0.876.
	if got != "a" {
		t.Errorf("OwnerW = %q, want a", got)
	}
}

func TestOwnerWErrors(t *testing.T) {
	if _, err := OwnerW("k", nil); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("empty: got %v", err)
	}

	bad := []struct {
		name  string
		nodes []WeightedNode
	}{
		{"zero weight", []WeightedNode{{Node: "a", Weight: 0}}},
		{"negative weight", []WeightedNode{{Node: "a", Weight: -3}}},
	}

	for _, tt := range bad {
		if _, err := OwnerW("k", tt.nodes); !errors.Is(err, ErrBadWeights) {
			t.Errorf("%s: got %v, want ErrBadWeights", tt.name, err)
		}
	}
}

func TestOwnerWEqualWeightsMatchUnweighted(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 12))

	for range 100 {
		key := "k" + string(rune(rng.IntN(500)))
		plain := []string{"n1", "n2", "n3"}
		weighted := []WeightedNode{
			{Node: "n1", Weight: 1}, {Node: "n2", Weight: 1}, {Node: "n3", Weight: 1},
		}

		a, _ := Owner(key, plain)
		b, _ := OwnerW(key, weighted)

		if a != b {
			t.Fatalf("key %q: plain=%s weighted=%s", key, a, b)
		}
	}
}

func TestOwnerWProportionalShare(t *testing.T) {
	nodes := []WeightedNode{
		{Node: "small", Weight: 1},
		{Node: "big", Weight: 9},
	}

	counts := map[string]int{}
	total := 20000

	for i := range total {
		owner, err := OwnerW(string(rune(i)), nodes)
		if err != nil {
			t.Fatal(err)
		}
		counts[owner]++
	}

	// big should get roughly 90%.
	bigShare := float64(counts["big"]) / float64(total)
	if bigShare < 0.85 || bigShare > 0.95 {
		t.Errorf("big share = %.2f, want ~0.90", bigShare)
	}
}

func TestScoreWeighted(t *testing.T) {
	w := Weighted{}

	const h = 1000
	score := w.score(h, 5)
	if score <= 0 || score >= 1 {
		t.Fatalf("score = %v, want in (0,1)", score)
	}

	if score <= w.score(h, 4) {
		t.Error("higher weight must raise the score")
	}

	if score >= w.score(h, 6) {
		t.Error("lower weight must lower the score")
	}
}

func TestBuildWErrors(t *testing.T) {
	if _, err := BuildW(nil); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("empty: got %v", err)
	}

	bad := []WeightedNode{{Node: "a", Weight: 0}}
	if _, err := BuildW(bad); !errors.Is(err, ErrBadWeights) {
		t.Errorf("bad weight: got %v", err)
	}
}

func TestBuildWBranchWeights(t *testing.T) {
	// Clusters of one: [a(1)] [b(3)] [c(1)] [d(1)]; fanout 2, levels 2.
	nodes := []WeightedNode{
		{Node: "c", Weight: 1}, {Node: "a", Weight: 1},
		{Node: "d", Weight: 1}, {Node: "b", Weight: 3},
	}

	s, err := BuildW(nodes, WithClusterSize(1))
	if err != nil {
		t.Fatal(err)
	}

	if s.fanout != 2 || s.levels != 2 {
		t.Fatalf("fanout=%d levels=%d, want 2/2", s.fanout, s.levels)
	}

	want := []map[string]int{
		{"0": 4, "1": 2}, // level 0: (a+b)=4, (c+d)=2
		{"0": 1, "1": 3, "2": 1, "3": 1},
	}

	for lvl, m := range want {
		got := s.branchWeights[lvl]

		if got == nil {
			t.Fatalf("level %d: nil branch weights", lvl)
		}

		for k, w := range m {
			key := atoi(k)
			if got[key] != w {
				t.Errorf("level %d branch %d = %d, want %d", lvl, key, got[key], w)
			}
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}

	return n
}

func TestBuildWSingleCluster(t *testing.T) {
	// Few nodes, large clusters: plain leaf weighting decides.
	nodes := []WeightedNode{{Node: "small", Weight: 1}, {Node: "big", Weight: 9}}

	s, err := BuildW(nodes)
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}

	total := 4000
	for i := range total {
		o, err := s.Owner(string(rune(i)))
		if err != nil {
			t.Fatal(err)
		}
		counts[o]++
	}

	share := float64(counts["big"]) / float64(total)
	if share < 0.85 || share > 0.95 {
		t.Errorf("big share = %.2f, want ~0.90", share)
	}
}

func TestBuildWTreeProportional(t *testing.T) {
	// Forced into a real tree: two single-node clusters, fanout 2.
	nodes := []WeightedNode{{Node: "small", Weight: 1}, {Node: "big", Weight: 9}}

	s, err := BuildW(nodes, WithClusterSize(1))
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}

	total := 4000
	for i := range total {
		o, err := s.Owner("key" + string(rune(i)))
		if err != nil {
			t.Fatal(err)
		}
		counts[o]++
	}

	share := float64(counts["big"]) / float64(total)
	if share < 0.85 || share > 0.95 {
		t.Errorf("big share = %.2f, want ~0.90", share)
	}
}

func TestBuildWValidOwners(t *testing.T) {
	nodes := []WeightedNode{
		{Node: "n1", Weight: 2}, {Node: "n2", Weight: 1},
		{Node: "n3", Weight: 5}, {Node: "n4", Weight: 2}, {Node: "n5", Weight: 1},
	}

	s, err := BuildW(nodes, WithClusterSize(1))
	if err != nil {
		t.Fatal(err)
	}

	valid := map[string]bool{}
	for _, n := range nodes {
		valid[n.Node] = true
	}

	rng := rand.New(rand.NewPCG(41, 42))

	for range 500 {
		o, err := s.Owner("k" + string(rune(rng.IntN(9999))))
		if err != nil {
			t.Fatal(err)
		}

		if !valid[o] {
			t.Fatalf("invalid owner %q", o)
		}
	}
}
