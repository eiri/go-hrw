# go-hrw

[![CI](https://github.com/eiri/go-hrw/actions/workflows/ci.yml/badge.svg)](https://github.com/eiri/go-hrw/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/eiri/go-hrw.svg)](https://pkg.go.dev/github.com/eiri/go-hrw)

HRW (Highest Random Weight), aka rendezvous hashing, for Go. Maps keys to
nodes stably under membership churn. Port of [joladev/hrw](https://github.com/joladev/hrw) (Elixir).

```go
// Linear scan, O(n)
owner, _ := hrw.Owner("192.168.0.1", []string{"server1", "server2", "server3"})

// Top-n owners (replicas), O(n log n)
owners, _ := hrw.Owners("192.168.0.1", nodes, 2)

// Skeleton-backed lookup, O(log n); plain data, build once
skel, _ := hrw.Build(nodes)
owner, _ = skel.Owner("192.168.0.1")

// Weighted: weight-10 node gets ~10x share
owner, _ = hrw.OwnerW("192.168.0.1", []hrw.WeightedNode{
    {Node: "server1", Weight: 1}, {Node: "server3", Weight: 10},
})

// Bounded load: no node exceeds ceil(k/n * (1+eps)) keys
m, _ := hrw.Assign(keys, nodes, hrw.WithEpsilon(0))
```

## Notes

- Unlike Elixir, default hash is FNV-1a 64 with a splitmix64 finalizer;
  plug in your own via `WithHash`. Assignments are stable within a configuration
  but intentionally not comparable across hash functions or library versions.
- `Build`/`BuildW` accept `WithFanout` and `WithClusterSize`; defaults are
  auto-selected (fanout by cost model, cluster size 16).
- Zero dependencies beyond the standard library.

## Development

Requires [mise](https://mise.jdx.dev/): `mise trust`, then `mise run check`
(gofmt -> vet -> build -> test -race).

Benchmarks: `go test -bench . -benchtime 1s`.
