package ere_pef

import (
	"reflect"
	"testing"
)

func costFnDefault(u, n uint64) uint64 { return minCodecBitsize(u, n) + defaultFixCost }

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
	part, cost := s.compute(keys, 42, 43, costFnDefault, defaultEps1, defaultEps2, nil)
	if !reflect.DeepEqual(part, []uint32{1}) {
		t.Errorf("part=%v want [1]", part)
	}
	if cost != defaultFixCost {
		t.Errorf("cost=%d want %d", cost, defaultFixCost)
	}
}

func TestPartitionContiguousRunIsSingleBlock(t *testing.T) {
	s := &partitionScratch{}
	keys := []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	part, cost := s.compute(keys, 0, 8, costFnDefault, defaultEps1, defaultEps2, nil)
	if !reflect.DeepEqual(part, []uint32{8}) {
		t.Errorf("part=%v want [8]", part)
	}
	if cost != defaultFixCost {
		t.Errorf("cost=%d want %d", cost, defaultFixCost)
	}
}

func TestPartitionDenseRunPlusOutlierSplits(t *testing.T) {
	s := &partitionScratch{}
	// 1000 contiguous keys followed by one far outlier — plain EF over the
	// whole sequence pays ~8.6Kb; splitting at index 1000 yields
	// allOnes(1000,1000)=64 + ef(99001,1)+1+64=85 = 149 bits.
	keys := make([]uint64, 0, 1001)
	for i := uint64(0); i < 1000; i++ {
		keys = append(keys, i)
	}
	keys = append(keys, 100000)
	part, cost := s.compute(keys, 0, 100001, costFnDefault, defaultEps1, defaultEps2, nil)
	want := []uint32{1000, 1001}
	if !reflect.DeepEqual(part, want) {
		t.Errorf("part=%v want %v", part, want)
	}
	if cost != 149 {
		t.Errorf("cost=%d want 149", cost)
	}
}

func TestPartitionScratchReused(t *testing.T) {
	s := &partitionScratch{}
	// First call sizes scratch buffers to fit 100 keys
	keys1 := make([]uint64, 100)
	for i := range keys1 {
		keys1[i] = uint64(i)
	}
	_, _ = s.compute(keys1, 0, 100, costFnDefault, defaultEps1, defaultEps2, nil)
	pAddr := &s.path[0]
	mAddr := &s.minCost[0]

	// Subsequent smaller call must not reallocate
	keys2 := []uint64{0, 1, 2, 3, 4}
	_, _ = s.compute(keys2, 0, 5, costFnDefault, defaultEps1, defaultEps2, nil)

	if &s.path[0] != pAddr {
		t.Error("path slice reallocated unexpectedly")
	}
	if &s.minCost[0] != mAddr {
		t.Error("minCost slice reallocated unexpectedly")
	}
}
