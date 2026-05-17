package rsdic

import (
	"fmt"
	"math/rand"
	"testing"
)

// TestSelect1Pair_MatchesSelect1 verifies that Select1Pair(r) returns the same
// pair as (Select1(r), Select1(r+1)) for every valid rank, plus the off-end
// boundary cases.
func TestSelect1Pair_MatchesSelect1(t *testing.T) {
	for _, n := range []uint64{1 << 10, 1 << 14, 1 << 18} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			rs, ones := buildRSDic(n, 42)
			if ones < 2 {
				t.Skip("need at least two ones")
			}
			step := ones / 1024
			if step == 0 {
				step = 1
			}
			for r := uint64(0); r < ones; r += step {
				wantA := rs.Select1(r)
				wantB := rs.Select1(r + 1)
				gotA, gotB := rs.Select1Pair(r)
				if gotA != wantA || gotB != wantB {
					t.Fatalf("n=%d r=%d: got (%d,%d), want (%d,%d)",
						n, r, gotA, gotB, wantA, wantB)
				}
			}
			// Edge: r == ones-1 (rank+1 falls off end)
			wantA := rs.Select1(ones - 1)
			wantB := rs.Select1(ones)
			gotA, gotB := rs.Select1Pair(ones - 1)
			if gotA != wantA || gotB != wantB {
				t.Fatalf("ones-1: got (%d,%d), want (%d,%d)", gotA, gotB, wantA, wantB)
			}
			// Edge: r == ones (both off)
			wantA = rs.Select1(ones)
			wantB = rs.Select1(ones + 1)
			gotA, gotB = rs.Select1Pair(ones)
			if gotA != wantA || gotB != wantB {
				t.Fatalf("ones: got (%d,%d), want (%d,%d)", gotA, gotB, wantA, wantB)
			}
		})
	}
}

// TestSelect1Pair_Patterns covers structurally interesting bitvectors where
// consecutive ranks may live in different small/large blocks (advance path).
func TestSelect1Pair_Patterns(t *testing.T) {
	cases := []struct {
		name string
		gen  func(n uint64) *RSDic
	}{
		{
			name: "all-ones-then-zeros",
			gen: func(n uint64) *RSDic {
				rs := New()
				for i := uint64(0); i < n/2; i++ {
					rs.PushBack(true)
				}
				for i := uint64(0); i < n/2; i++ {
					rs.PushBack(false)
				}
				return rs
			},
		},
		{
			name: "alternating",
			gen: func(n uint64) *RSDic {
				rs := New()
				for i := uint64(0); i < n; i++ {
					rs.PushBack(i%2 == 0)
				}
				return rs
			},
		},
		{
			name: "sparse-1-in-64",
			gen: func(n uint64) *RSDic {
				rs := New()
				for i := uint64(0); i < n; i++ {
					rs.PushBack(i%64 == 0)
				}
				return rs
			},
		},
		{
			name: "cluster-then-sparse",
			gen: func(n uint64) *RSDic {
				rs := New()
				for i := uint64(0); i < n/4; i++ {
					rs.PushBack(true)
				}
				for i := uint64(0); i < 3*n/4; i++ {
					rs.PushBack(i%64 == 0)
				}
				return rs
			},
		},
		{
			name: "random-90pct",
			gen: func(n uint64) *RSDic {
				rng := rand.New(rand.NewSource(11))
				rs := New()
				for i := uint64(0); i < n; i++ {
					rs.PushBack(rng.Intn(10) < 9)
				}
				return rs
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rs := c.gen(1 << 14)
			ones := rs.OneNum()
			step := ones / 4096
			if step == 0 {
				step = 1
			}
			for r := uint64(0); r < ones; r += step {
				wantA := rs.Select1(r)
				wantB := rs.Select1(r + 1)
				gotA, gotB := rs.Select1Pair(r)
				if gotA != wantA || gotB != wantB {
					t.Fatalf("%s r=%d: got (%d,%d), want (%d,%d)",
						c.name, r, gotA, gotB, wantA, wantB)
				}
			}
		})
	}
}
