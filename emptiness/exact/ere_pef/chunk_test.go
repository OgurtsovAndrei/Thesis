package ere_pef

import "testing"

// selectCodec now takes (lastRel, n) where lastRel = last - base.
func TestSelectCodec(t *testing.T) {
	// lastRel=7, n=8: 7 == n-1=7 → all-ones
	if got := selectCodec(7, 8); got != kindAllOnes {
		t.Errorf("lastRel=7 n=8: got %d, want allOnes", got)
	}
	// lastRel=3, n=3: 3 != n-1=2 → not all-ones; bitmap(4)<ef → bitmap
	if got := selectCodec(3, 3); got != kindBitmap {
		t.Errorf("lastRel=3 n=3: got %d, want bitmap", got)
	}
	// lastRel=40, n=3: sparse → EF
	if got := selectCodec(40, 3); got != kindEF {
		t.Errorf("lastRel=40 n=3: got %d, want EF", got)
	}
}

// makeChunk builds a single-chunk PEF in `p` with the supplied keys
// (caller-relative to `base`) and returns the chunk index for use with
// p.chunkIntersects. firstKey/lastKey are populated so chunkBaseAt
// works as in the production builder.
func appendChunk(p *PEF, base uint64, keys []uint64) int {
	c := chunk{
		last:  keys[len(keys)-1],
		nKind: packNKind(uint32(len(keys)), 0),
	}
	if len(p.chunks) == 0 {
		p.firstKey = base
	}
	p.writeChunk(&c, base, keys)
	idx := len(p.chunks)
	p.chunks = append(p.chunks, c)
	p.lastKey = c.last
	p.n += len(keys)
	return idx
}

func TestChunkAllOnes(t *testing.T) {
	keys := []uint64{5, 6, 7, 8}
	p := &PEF{}
	idx := appendChunk(p, 5, keys)
	if p.chunks[idx].kind() != kindAllOnes {
		t.Fatalf("kind=%d, want allOnes", p.chunks[idx].kind())
	}
	for _, q := range []struct{ a, b uint64 }{
		{5, 5}, {7, 7}, {6, 7}, {5, 8},
	} {
		if !p.chunkIntersects(idx, q.a, q.b) {
			t.Errorf("intersects(%d,%d) = false, want true", q.a, q.b)
		}
	}
}

func TestChunkEF(t *testing.T) {
	keys := []uint64{10, 30, 50}
	p := &PEF{}
	idx := appendChunk(p, 10, keys)
	if p.chunks[idx].kind() != kindEF {
		t.Fatalf("kind=%d, want EF", p.chunks[idx].kind())
	}
	cases := []struct {
		a, b uint64
		want bool
	}{
		{10, 10, true},
		{11, 29, false},
		{11, 49, true},
		{25, 30, true},
		{31, 49, false},
		{30, 30, true},
		{49, 50, true},
		{50, 50, true},
		{29, 30, true},
		{29, 29, false},
	}
	for _, tc := range cases {
		got := p.chunkIntersects(idx, tc.a, tc.b)
		if got != tc.want {
			t.Errorf("intersects(%d,%d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestChunkEFAllSparse(t *testing.T) {
	// EF with ell > 0 chosen, single key per bucket
	keys := []uint64{0, 100, 200, 300}
	p := &PEF{}
	idx := appendChunk(p, 0, keys)
	if p.chunks[idx].kind() != kindEF {
		t.Fatalf("kind=%d", p.chunks[idx].kind())
	}
	for _, k := range keys {
		if !p.chunkIntersects(idx, k, k) {
			t.Errorf("self-membership %d failed", k)
		}
	}
	// in-gap queries
	for _, q := range []struct{ a, b uint64 }{
		{1, 99}, {101, 199}, {201, 299},
	} {
		if p.chunkIntersects(idx, q.a, q.b) {
			t.Errorf("gap (%d,%d) reported non-empty", q.a, q.b)
		}
	}
}

func TestChunkBitmap(t *testing.T) {
	keys := []uint64{0, 1, 3}
	p := &PEF{}
	idx := appendChunk(p, 0, keys)
	if p.chunks[idx].kind() != kindBitmap {
		t.Fatalf("kind=%d, want bitmap", p.chunks[idx].kind())
	}
	cases := []struct {
		a, b uint64
		want bool
	}{
		{0, 0, true},
		{1, 1, true},
		{2, 2, false},
		{3, 3, true},
		{0, 3, true},
		{2, 3, true},
		{0, 1, true},
	}
	for _, tc := range cases {
		got := p.chunkIntersects(idx, tc.a, tc.b)
		if got != tc.want {
			t.Errorf("intersects(%d,%d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestChunkEFBinarySearchPath(t *testing.T) {
	// Force a single EF bucket with > linearScanThreshold (=128) elements
	// to exercise efBucketHasLow's binary-search branch. Natural NewPEF
	// DP would split such a chunk; we use appendChunk to bypass it.
	keys := make([]uint64, 0, 201)
	for i := uint64(0); i < 200; i++ {
		keys = append(keys, i)
	}
	keys = append(keys, 100000) // single outlier — forces ell=8 ⇒ bucket size 256
	p := &PEF{}
	idx := appendChunk(p, 0, keys)
	if p.chunks[idx].kind() != kindEF {
		t.Fatalf("kind=%d, want EF (chunk should not collapse to all-ones/bitmap)", p.chunks[idx].kind())
	}
	for _, k := range keys {
		if !p.chunkIntersects(idx, k, k) {
			t.Errorf("self-membership %d failed", k)
		}
	}
	for _, q := range []struct{ a, b uint64 }{
		{200, 99999}, {99999, 99999},
	} {
		if p.chunkIntersects(idx, q.a, q.b) {
			t.Errorf("gap (%d,%d) reported non-empty", q.a, q.b)
		}
	}
}

func TestChunkAcrossWordBoundary(t *testing.T) {
	// Build two EF chunks back-to-back to exercise lowBits offsets across a 64-bit boundary.
	// Cumulative base layout: chunk[i+1].base = chunk[i].last + 1.
	p := &PEF{}
	idx1 := appendChunk(p, 10, []uint64{10, 30, 50})
	idx2 := appendChunk(p, 51, []uint64{100, 200, 300, 400, 500})
	if !p.chunkIntersects(idx2, 200, 200) {
		t.Error("c2 self-membership 200 failed")
	}
	if p.chunkIntersects(idx2, 201, 299) {
		t.Error("c2 gap (201,299) reported non-empty")
	}
	if !p.chunkIntersects(idx1, 30, 30) {
		t.Error("c1 self-membership 30 failed (offset corruption?)")
	}
}
