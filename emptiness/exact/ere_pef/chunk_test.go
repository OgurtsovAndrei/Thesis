package ere_pef

import "testing"

func TestSelectCodec(t *testing.T) {
	if got := selectCodec(8, 8); got != kindAllOnes {
		t.Errorf("u=n=8: got %d, want allOnes", got)
	}
	if got := selectCodec(4, 3); got != kindBitmap {
		t.Errorf("u=4 n=3: got %d, want bitmap", got)
	}
	if got := selectCodec(41, 3); got != kindEF {
		t.Errorf("u=41 n=3: got %d, want EF", got)
	}
}

func TestChunkAllOnes(t *testing.T) {
	keys := []uint64{5, 6, 7, 8}
	p := &PEF{}
	c := chunk{base: 5, last: 8, n: 4}
	p.writeChunk(&c, keys)
	if c.kind != kindAllOnes {
		t.Fatalf("kind=%d, want allOnes", c.kind)
	}
	for _, q := range []struct{ a, b uint64 }{
		{5, 5}, {7, 7}, {6, 7}, {5, 8},
	} {
		if !p.chunkIntersects(&c, q.a, q.b) {
			t.Errorf("intersects(%d,%d) = false, want true", q.a, q.b)
		}
	}
}

func TestChunkEF(t *testing.T) {
	keys := []uint64{10, 30, 50}
	p := &PEF{}
	c := chunk{base: 10, last: 50, n: 3}
	p.writeChunk(&c, keys)
	if c.kind != kindEF {
		t.Fatalf("kind=%d, want EF", c.kind)
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
		got := p.chunkIntersects(&c, tc.a, tc.b)
		if got != tc.want {
			t.Errorf("intersects(%d,%d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestChunkEFAllSparse(t *testing.T) {
	// EF with ell > 0 chosen, single key per bucket
	keys := []uint64{0, 100, 200, 300}
	p := &PEF{}
	c := chunk{base: 0, last: 300, n: 4}
	p.writeChunk(&c, keys)
	if c.kind != kindEF {
		t.Fatalf("kind=%d", c.kind)
	}
	for _, k := range keys {
		if !p.chunkIntersects(&c, k, k) {
			t.Errorf("self-membership %d failed", k)
		}
	}
	// in-gap queries
	for _, q := range []struct{ a, b uint64 }{
		{1, 99}, {101, 199}, {201, 299},
	} {
		if p.chunkIntersects(&c, q.a, q.b) {
			t.Errorf("gap (%d,%d) reported non-empty", q.a, q.b)
		}
	}
}

func TestChunkBitmap(t *testing.T) {
	keys := []uint64{0, 1, 3}
	p := &PEF{}
	c := chunk{base: 0, last: 3, n: 3}
	p.writeChunk(&c, keys)
	if c.kind != kindBitmap {
		t.Fatalf("kind=%d, want bitmap", c.kind)
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
		got := p.chunkIntersects(&c, tc.a, tc.b)
		if got != tc.want {
			t.Errorf("intersects(%d,%d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestChunkAcrossWordBoundary(t *testing.T) {
	// Build two EF chunks back-to-back to exercise lowBits offsets across a 64-bit boundary
	p := &PEF{}
	c1 := chunk{base: 10, last: 50, n: 3}
	p.writeChunk(&c1, []uint64{10, 30, 50})
	c2 := chunk{base: 100, last: 500, n: 5}
	p.writeChunk(&c2, []uint64{100, 200, 300, 400, 500})
	if !p.chunkIntersects(&c2, 200, 200) {
		t.Error("c2 self-membership 200 failed")
	}
	if p.chunkIntersects(&c2, 201, 299) {
		t.Error("c2 gap (201,299) reported non-empty")
	}
	if !p.chunkIntersects(&c1, 30, 30) {
		t.Error("c1 self-membership 30 failed (offset corruption?)")
	}
}
