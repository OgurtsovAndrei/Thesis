package rsdic

import (
	"math/rand"
	"testing"
)

type rawBitVector struct {
	orig   []uint8
	ranks  []uint64
	num    uint64
	oneNum uint64
}

func initBitVector(num uint64, ratio float32) (*rawBitVector, *RSDic) {
	orig := make([]uint8, num)
	ranks := make([]uint64, num)
	oneNum := uint64(0)
	rsd := New()
	for i := uint64(0); i < num; i++ {
		ranks[i] = oneNum
		if rand.Float32() > ratio {
			orig[i] = 0
			rsd.PushBack(false)
		} else {
			orig[i] = 1
			rsd.PushBack(true)
			oneNum++
		}
	}
	return &rawBitVector{orig, ranks, num, oneNum}, rsd
}

func TestEmptyRSDic(t *testing.T) {
	rsd := New()
	if rsd.Num() != 0 {
		t.Errorf("Num: got %d, want 0", rsd.Num())
	}
	if rsd.ZeroNum() != 0 {
		t.Errorf("ZeroNum: got %d, want 0", rsd.ZeroNum())
	}
	if rsd.OneNum() != 0 {
		t.Errorf("OneNum: got %d, want 0", rsd.OneNum())
	}
	if rsd.Rank(0, true) != 0 {
		t.Errorf("Rank(0,true): got %d, want 0", rsd.Rank(0, true))
	}
	if rsd.AllocSize() != 0 {
		t.Errorf("AllocSize: got %d, want 0", rsd.AllocSize())
	}
}

const testNum = 100

func runTestRSDic(t *testing.T, name string, rsd *RSDic, raw *rawBitVector) {
	t.Run(name, func(t *testing.T) {
		if rsd.Num() != raw.num {
			t.Fatalf("Num: got %d, want %d", rsd.Num(), raw.num)
		}
		if rsd.OneNum() != raw.oneNum {
			t.Fatalf("OneNum: got %d, want %d", rsd.OneNum(), raw.oneNum)
		}
		if rsd.Rank(raw.num, true) != raw.oneNum {
			t.Fatalf("Rank(num,true): got %d, want %d", rsd.Rank(raw.num, true), raw.oneNum)
		}
		for i := 0; i < testNum; i++ {
			ind := uint64(rand.Int31n(int32(raw.num)))
			if i == 0 {
				ind = 0
			}
			wantBit := raw.orig[ind] == 1
			if rsd.Bit(ind) != wantBit {
				t.Fatalf("Bit(%d): got %v, want %v", ind, rsd.Bit(ind), wantBit)
			}
			if rsd.Rank(ind, false) != ind-raw.ranks[ind] {
				t.Fatalf("Rank(%d,false): got %d, want %d", ind, rsd.Rank(ind, false), ind-raw.ranks[ind])
			}
			if rsd.Rank(ind, true) != raw.ranks[ind] {
				t.Fatalf("Rank(%d,true): got %d, want %d", ind, rsd.Rank(ind, true), raw.ranks[ind])
			}
			bit, rank := rsd.BitAndRank(ind)
			if bit != wantBit {
				t.Fatalf("BitAndRank(%d) bit: got %v, want %v", ind, bit, wantBit)
			}
			wantRank := bitNum(raw.ranks[ind], ind, bit)
			if rank != wantRank {
				t.Fatalf("BitAndRank(%d) rank: got %d, want %d", ind, rank, wantRank)
			}
			if rsd.Select(rank, bit) != ind {
				t.Fatalf("Select(%d,%v): got %d, want %d", rank, bit, rsd.Select(rank, bit), ind)
			}
		}

		out, err := rsd.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		newrsd := New()
		if err := newrsd.UnmarshalBinary(out); err != nil {
			t.Fatalf("UnmarshalBinary: %v", err)
		}
		for i := 0; i < testNum; i++ {
			ind := uint64(rand.Int31n(int32(raw.num)))
			wantBit := raw.orig[ind] == 1
			if newrsd.Bit(ind) != wantBit {
				t.Fatalf("After unmarshal Bit(%d): got %v, want %v", ind, newrsd.Bit(ind), wantBit)
			}
			if newrsd.Rank(ind, true) != raw.ranks[ind] {
				t.Fatalf("After unmarshal Rank(%d,true): got %d, want %d", ind, newrsd.Rank(ind, true), raw.ranks[ind])
			}
		}
	})
}

func TestSmallRSDic(t *testing.T) {
	raw, rsd := initBitVector(100, 0.5)
	runTestRSDic(t, "small", rsd, raw)
}

func TestLargeRSDic(t *testing.T) {
	raw, rsd := initBitVector(100000, 0.5)
	runTestRSDic(t, "large_dense", rsd, raw)
}

func TestVeryLargeRSDic(t *testing.T) {
	raw, rsd := initBitVector(4000000, 0.8)
	runTestRSDic(t, "very_large", rsd, raw)
}

func TestSparseRSDic(t *testing.T) {
	raw, rsd := initBitVector(100000, 0.01)
	runTestRSDic(t, "large_sparse", rsd, raw)
}

func TestAllZeroRSDic(t *testing.T) {
	raw, rsd := initBitVector(100, 0)
	runTestRSDic(t, "all_zero", rsd, raw)
}

// Benchmarks

func setupRSDic(num uint64, ratio float32) *RSDic {
	rsd := New()
	for i := uint64(0); i < num; i++ {
		if rand.Float32() < ratio {
			rsd.PushBack(true)
		} else {
			rsd.PushBack(false)
		}
	}
	return rsd
}

const benchN = 1000000

func BenchmarkBit(b *testing.B) {
	rsd := setupRSDic(benchN, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchN))))
	}
}

func BenchmarkRank(b *testing.B) {
	rsd := setupRSDic(benchN, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchN))), true)
	}
}

func BenchmarkSelect(b *testing.B) {
	rsd := setupRSDic(benchN, 0.5)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}

func BenchmarkSparseBit(b *testing.B) {
	rsd := setupRSDic(benchN, 0.01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchN))))
	}
}

func BenchmarkSparseRank(b *testing.B) {
	rsd := setupRSDic(benchN, 0.01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchN))), true)
	}
}

func BenchmarkSparseSelect(b *testing.B) {
	rsd := setupRSDic(benchN, 0.01)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}
