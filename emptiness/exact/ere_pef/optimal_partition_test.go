package ere_pef

import (
	"reflect"
	"testing"
)

// costFnDefault uses the updated (lastRel, n) signature for minCodecBitsize.
func costFnDefault(lastRel, n uint64) uint64 { return minCodecBitsize(lastRel, n) + defaultFixCost }

func TestPartitionEmpty(t *testing.T) {
	s := &partitionScratch{}
	part, cost := s.compute(nil, 0, 0, costFnDefault, defaultEps1, defaultEps2, nil)
	if len(part) != 0 || cost != 0 {
		t.Errorf("empty: part=%v cost=%d", part, cost)
	}
}

func TestPartitionSingleKey(t *testing.T) {
	s := &partitionScratch{}
	keys := []uint64{42}
	// third arg is now lastKey (inclusive), not universe (exclusive)
	part, cost := s.compute(keys, 42, 42, costFnDefault, defaultEps1, defaultEps2, nil)
	if !reflect.DeepEqual(part, []uint32{1}) {
		t.Errorf("part=%v want [1]", part)
	}
	// costFn(lastKey-base=0, n=1) = minCodecBitsize(0,1)+64 = 0+64 = 64
	if cost != defaultFixCost {
		t.Errorf("cost=%d want %d", cost, defaultFixCost)
	}
}

func TestPartitionContiguousRunIsSingleBlock(t *testing.T) {
	s := &partitionScratch{}
	keys := []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	// lastKey=7 (inclusive); costFn(7-0=7, 8) = allOnesBitsize(7,8)=0 → 64
	part, cost := s.compute(keys, 0, 7, costFnDefault, defaultEps1, defaultEps2, nil)
	if !reflect.DeepEqual(part, []uint32{8}) {
		t.Errorf("part=%v want [8]", part)
	}
	if cost != defaultFixCost {
		t.Errorf("cost=%d want %d", cost, defaultFixCost)
	}
}

func TestPartitionDenseRunPlusOutlierSplits(t *testing.T) {
	s := &partitionScratch{}
	// 1000 contiguous keys [0..999] followed by one far outlier 100000.
	// Optimal split: chunk0 = allOnes(lastRel=999,n=1000) → 0+64=64 bits;
	// chunk1 = ef(lastRel=99000,n=1)+1+64 = 21+1+64=86 bits. Total = 150.
	keys := make([]uint64, 0, 1001)
	for i := uint64(0); i < 1000; i++ {
		keys = append(keys, i)
	}
	keys = append(keys, 100000)
	// lastKey=100000 (inclusive)
	part, cost := s.compute(keys, 0, 100000, costFnDefault, defaultEps1, defaultEps2, nil)
	want := []uint32{1000, 1001}
	if !reflect.DeepEqual(part, want) {
		t.Errorf("part=%v want %v", part, want)
	}
	if cost != 150 {
		t.Errorf("cost=%d want 150", cost)
	}
}

func TestPartitionScratchReused(t *testing.T) {
	s := &partitionScratch{}
	// First call sizes scratch buffers to fit 100 keys
	keys1 := make([]uint64, 100)
	for i := range keys1 {
		keys1[i] = uint64(i)
	}
	_, _ = s.compute(keys1, 0, 99, costFnDefault, defaultEps1, defaultEps2, nil)
	pAddr := &s.path[0]
	mAddr := &s.minCost[0]

	// Subsequent smaller call must not reallocate
	keys2 := []uint64{0, 1, 2, 3, 4}
	_, _ = s.compute(keys2, 0, 4, costFnDefault, defaultEps1, defaultEps2, nil)

	if &s.path[0] != pAddr {
		t.Error("path slice reallocated unexpectedly")
	}
	if &s.minCost[0] != mAddr {
		t.Error("minCost slice reallocated unexpectedly")
	}
}
