package hrw

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestAssignGolden(t *testing.T) {
	// Independent-oracle values: cap = ceil(4/2 * 1.0) = 2; greedy by score.
	got, err := Assign([]string{"a", "b", "c", "d"}, []string{"x", "y"}, WithEpsilon(0))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"a": "y", "b": "x", "c": "y", "d": "x"}

	for k, w := range want {
		if got[k] != w {
			t.Errorf("Assign(%q) = %q, want %q", k, got[k], w)
		}
	}
}

func TestAssignErrors(t *testing.T) {
	keys := []string{"k"}

	if _, err := Assign(keys, nil); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("empty nodes: got %v", err)
	}

	if _, err := Assign(keys, []string{"n"}, WithEpsilon(-0.1)); !errors.Is(err, ErrBadEpsilon) {
		t.Errorf("negative epsilon: got %v, want ErrBadEpsilon", err)
	}
}

func TestAssignEmptyKeys(t *testing.T) {
	got, err := Assign(nil, []string{"n"})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
}

func TestAssignRespectsCap(t *testing.T) {
	rng := rand.New(rand.NewPCG(51, 52))

	for range 50 {
		nKeys := rng.IntN(100) + 1
		nNodes := rng.IntN(7) + 1
		eps := float64(rng.IntN(4)) / 4

		keys := make([]string, nKeys)
		nodes := make([]string, nNodes)

		for i := range keys {
			keys[i] = fmt.Sprintf("key-%d-%d", rng.IntN(500), i)
		}

		for i := range nodes {
			nodes[i] = "node" + string(rune('a'+i))
		}

		got, err := Assign(keys, nodes, WithEpsilon(eps))
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != len(keys) {
			t.Fatalf("assigned %d of %d keys", len(got), len(keys))
		}

		cap := (nKeys + nNodes - 1) / nNodes // ceil
		cap = int(float64(cap)*(1+eps)) + 1  // conservative upper bound

		load := map[string]int{}
		for _, n := range got {
			load[n]++

			if load[n] > cap {
				t.Fatalf("node %s loaded %d > cap %d (keys=%d nodes=%d eps=%.2f)",
					n, load[n], cap, nKeys, nNodes, eps)
			}
		}
	}
}

func TestAssignDeterministic(t *testing.T) {
	keys := []string{"k1", "k2", "k3"}
	nodes := []string{"a", "b", "c"}

	first, err := Assign(keys, nodes)
	if err != nil {
		t.Fatal(err)
	}

	for range 5 {
		got, _ := Assign(keys, nodes)

		for k, v := range first {
			if got[k] != v {
				t.Fatalf("key %q moved from %q to %q", k, v, got[k])
			}
		}
	}
}

func TestAssignIgnoresDuplicateKeysAndNodes(t *testing.T) {
	a, _ := Assign([]string{"k", "k"}, []string{"n1", "n2", "n1"})
	b, _ := Assign([]string{"k"}, []string{"n1", "n2"})

	if a["k"] != b["k"] {
		t.Errorf("duplicates changed assignment: %q vs %q", a["k"], b["k"])
	}
}
