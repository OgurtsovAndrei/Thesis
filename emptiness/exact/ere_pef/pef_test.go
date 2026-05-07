package ere_pef

import (
	"testing"
)

func TestNewPEFEmpty(t *testing.T) {
	p, err := NewPEF(nil, 60)
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsEmpty(0, 1<<59) {
		t.Error("empty PEF must report empty for any range")
	}
	if p.ByteSize() != 0 {
		t.Errorf("empty ByteSize=%d want 0", p.ByteSize())
	}
}

func TestNewPEFRejectsUnsorted(t *testing.T) {
	_, err := NewPEF([]uint64{5, 3, 7}, 60)
	if err == nil {
		t.Error("expected error on unsorted input")
	}
}

func TestNewPEFDeduplicates(t *testing.T) {
	p, err := NewPEF([]uint64{1, 1, 2, 2, 3}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if p.n != 3 {
		t.Errorf("dedup n=%d want 3", p.n)
	}
}

func TestPEFSingleKey(t *testing.T) {
	p, err := NewPEF([]uint64{42}, 60)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		a, b uint64
		want bool
	}{
		{42, 42, false},
		{0, 41, true},
		{43, 100, true},
		{40, 50, false},
		{0, 1 << 30, false},
	}
	for _, c := range cases {
		if got := p.IsEmpty(c.a, c.b); got != c.want {
			t.Errorf("IsEmpty(%d,%d)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestPEFContiguousRun(t *testing.T) {
	keys := []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	p, err := NewPEF(keys, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.chunks) != 1 {
		t.Errorf("contiguous run should fit in one chunk, got %d", len(p.chunks))
	}
	if p.chunks[0].kind() != kindAllOnes {
		t.Errorf("contiguous run should pick all-ones, got kind=%d", p.chunks[0].kind())
	}
	for _, k := range keys {
		if p.IsEmpty(k, k) {
			t.Errorf("self-membership %d failed", k)
		}
	}
	if !p.IsEmpty(8, 100) {
		t.Error("post-run query not reported empty")
	}
}

func TestPEFDenseRunPlusOutlier(t *testing.T) {
	keys := make([]uint64, 0, 1001)
	for i := uint64(0); i < 1000; i++ {
		keys = append(keys, i)
	}
	keys = append(keys, 100000)
	p, err := NewPEF(keys, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.chunks) != 2 {
		t.Errorf("expected split into 2 chunks, got %d", len(p.chunks))
	}
	for _, k := range []uint64{0, 500, 999, 100000} {
		if p.IsEmpty(k, k) {
			t.Errorf("self-membership %d failed", k)
		}
	}
	if !p.IsEmpty(1000, 99999) {
		t.Error("gap query not empty")
	}
	if p.IsEmpty(99999, 100001) {
		t.Error("crossing-into-outlier query reported empty")
	}
	if p.IsEmpty(500, 100000) {
		t.Error("range covering both clusters reported empty")
	}
}

func TestPEFQueryOutsideUniverse(t *testing.T) {
	p, _ := NewPEF([]uint64{100, 200, 300}, 60)
	if !p.IsEmpty(0, 99) {
		t.Error("[0,99] should be empty")
	}
	if !p.IsEmpty(301, 1<<59) {
		t.Error("[301,..] should be empty")
	}
	if p.IsEmpty(0, 1<<59) {
		t.Error("[0,..] covers all keys, not empty")
	}
}

func TestPEFCrossSuperblock(t *testing.T) {
	// PISA superblock_size = fix_cost / eps3 = 64 / 0.01 = 6400.
	// Build > 2 superblocks and verify queries straddling boundaries.
	const N = 20000
	keys := make([]uint64, N)
	for i := range keys {
		keys[i] = uint64(i) * 10
	}
	p, err := NewPEF(keys, 60)
	if err != nil {
		t.Fatal(err)
	}
	for _, idx := range []int{0, 1000, 6399, 6400, 6401, 12800, N - 1} {
		k := keys[idx]
		if p.IsEmpty(k, k) {
			t.Errorf("self-membership keys[%d]=%d failed", idx, k)
		}
		// gap before this key
		if idx > 0 && !p.IsEmpty(keys[idx-1]+1, k-1) {
			t.Errorf("gap before keys[%d] not empty", idx)
		}
	}
}

func TestNewPEFWithConfigRejectsBadConfig(t *testing.T) {
	keys := []uint64{1, 2, 3}
	cases := []PartitionConfig{
		{Eps1: -0.1, Eps2: 0.3, Eps3: 0.01, FixCost: 64},
		{Eps1: 1.0, Eps2: 0.3, Eps3: 0.01, FixCost: 64},
		{Eps1: 0.03, Eps2: 0, Eps3: 0.01, FixCost: 64},
		{Eps1: 0.03, Eps2: 0.3, Eps3: -0.1, FixCost: 64},
		{Eps1: 0.03, Eps2: 0.3, Eps3: 0.01, FixCost: 0},
	}
	for i, cfg := range cases {
		if _, err := NewPEFWithConfig(keys, 60, cfg); err == nil {
			t.Errorf("case #%d: expected error for cfg=%+v", i, cfg)
		}
	}
}

func TestNewPEFWithConfigParityVsDefault(t *testing.T) {
	// On the same input, NewPEF (defaults) and NewPEFWithConfig (looser
	// eps) must produce IsEmpty answers that agree byte-for-byte. The
	// looser cfg may yield a different partition (slightly more chunks
	// or slightly worse BPK), but functional emptiness is preserved.
	keys := make([]uint64, 0, 8192)
	for i := uint64(0); i < 8192; i++ {
		keys = append(keys, i*7+i*i)
	}
	def, err := NewPEF(keys, 60)
	if err != nil {
		t.Fatal(err)
	}
	loose, err := NewPEFWithConfig(keys, 60, PartitionConfig{
		Eps1: 0.1, Eps2: 0.5, Eps3: 0.01, FixCost: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < 8192*8; i++ {
		a := i * 13
		b := a + 50
		if def.IsEmpty(a, b) != loose.IsEmpty(a, b) {
			t.Fatalf("divergence at (%d,%d): default=%v loose=%v",
				a, b, def.IsEmpty(a, b), loose.IsEmpty(a, b))
		}
	}
	// Parity sanity: both must report non-empty for every actual key.
	for _, k := range keys {
		if def.IsEmpty(k, k) || loose.IsEmpty(k, k) {
			t.Fatalf("self-membership %d failed", k)
		}
	}
}
