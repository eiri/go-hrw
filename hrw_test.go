package hrw

import (
	"errors"
	"math/rand/v2"
	"testing"
)

func TestOwnerGolden(t *testing.T) {
	nodes := []string{"server1", "server2", "server3"}

	tests := []struct {
		key  string
		want string
	}{
		// Vectors computed independently of the Go implementation.
		{"192.168.0.1", "server2"},
		{"192.168.0.2", "server2"},
		{"user-42", "server2"},
	}

	for _, tt := range tests {
		got, err := Owner(tt.key, nodes)
		if err != nil {
			t.Fatalf("Owner(%q): %v", tt.key, err)
		}
		if got != tt.want {
			t.Errorf("Owner(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestOwnerErrors(t *testing.T) {
	if _, err := Owner("k", nil); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("empty nodes: got %v, want ErrEmptyNodes", err)
	}
	if _, err := Owner("k", []string{}); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("nil nodes: got %v, want ErrEmptyNodes", err)
	}
}

func TestOwnerIgnoresDuplicates(t *testing.T) {
	a, err := Owner("key", []string{"n1", "n2"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Owner("key", []string{"n1", "n1", "n2", "n2", "n1"})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("duplicates changed owner: %q vs %q", a, b)
	}
}

func TestOwnerStableOnNonOwnerRemoval(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))

	for range 200 {
		base := make([]string, 8)

		for i := range base {
			base[i] = "node-" + string(rune('a'+i))
		}

		key := "key-" + string(rune(rng.IntN(1000)))
		owner, _ := Owner(key, base)

		for _, victim := range base {
			if victim == owner {
				continue
			}

			reduced := sliceWithout(base, victim)
			still, _ := Owner(key, reduced)
			if still != owner {
				t.Fatalf("removing non-owner %q changed owner of %q: %s -> %s",
					victim, key, owner, still)
			}
		}
	}
}

func TestOwnerChangesOnlyToNewNode(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))

	for range 200 {
		nodes := []string{"a", "b", "c", "d"}
		key := "k" + string(rune(rng.IntN(1000)))
		before, _ := Owner(key, nodes)
		after, _ := Owner(key, append(nodes, "e"))

		if after != before && after != "e" {
			t.Fatalf("adding node e moved %q from %s to unrelated %s", key, before, after)
		}
	}
}

func TestOwnerDeterministic(t *testing.T) {
	nodes := []string{"x", "y", "z"}
	first, _ := Owner("some-key", nodes)

	for range 10 {
		if got, _ := Owner("some-key", nodes); got != first {
			t.Fatalf("expected %q, got %q", first, got)
		}
	}
}

func sliceWithout(nodes []string, s string) []string {
	out := make([]string, 0, len(nodes)-1)

	for _, n := range nodes {
		if n != s {
			out = append(out, n)
		}
	}

	return out
}

func TestOwnersGolden(t *testing.T) {
	nodes := []string{"server1", "server2", "server3"}

	got, err := Owners("192.168.0.1", nodes, 3)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"server2", "server1", "server3"}

	for i, w := range want {
		if got[i] != w {
			t.Fatalf("Owners rank %d = %q, want %q (full: %v)", i, got[i], w, got)
		}
	}
}

func TestOwnersErrors(t *testing.T) {
	nodes := []string{"a", "b"}

	if _, err := Owners("k", nil, 1); !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("empty nodes: got %v", err)
	}

	if _, err := Owners("k", nodes, -1); !errors.Is(err, ErrBadCount) {
		t.Errorf("negative count: got %v, want ErrBadCount", err)
	}
}

func TestOwnersCountExceedsNodes(t *testing.T) {
	got, err := Owners("k", []string{"b", "a"}, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Errorf("got %v, want [b a] sorted by score desc", got)
	}
}

func TestOwnersZeroCount(t *testing.T) {
	got, err := Owners("k", []string{"a"}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestOwnersPrefixOfRanking(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))

	for range 100 {
		nodes := make([]string, 10)

		for i := range nodes {
			nodes[i] = "n" + string(rune('a'+i))
		}

		key := "k" + string(rune(rng.IntN(500)))
		full, _ := Owners(key, nodes, 10)
		top3, _ := Owners(key, nodes, 3)

		for i := range top3 {
			if top3[i] != full[i] {
				t.Fatalf("top3[%d]=%s != full[%d]=%s for %q", i, top3[i], i, full[i], key)
			}
		}
	}
}
