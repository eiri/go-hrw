package hrw

import "testing"

func TestDefaultHashGolden(t *testing.T) {
	// Vectors computed independently of the Go implementation.
	tests := []struct {
		key, node string
		want      uint64
	}{
		{"", "", 0x25fc6dd36ce04b20},
		{"key", "node", 0x60a0c9bccb877744},
		{"192.168.0.1", "server1", 0x690e59980186c360},
	}

	for _, tt := range tests {
		if got := defaultHash(tt.key, tt.node); got != tt.want {
			t.Errorf("defaultHash(%q, %q) = %#x, want %#x", tt.key, tt.node, got, tt.want)
		}
	}
}

func TestDefaultHashOrderMatters(t *testing.T) {
	if defaultHash("a", "b") == defaultHash("b", "a") {
		t.Error("hash(a,b) must differ from hash(b,a)")
	}
}

func TestDefaultHashDeterministic(t *testing.T) {
	first := defaultHash("key1", "node1")

	for range 5 {
		if defaultHash("key1", "node1") != first {
			t.Fatal("same input produced different hashes")
		}
	}
}
