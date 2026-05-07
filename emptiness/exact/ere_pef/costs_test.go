package ere_pef

import "testing"

func TestEFBitsizePaper(t *testing.T) {
	cases := []struct {
		name       string
		u, n, want uint64
	}{
		{"u100_n10", 100, 10, 54},
		{"u11_n10", 11, 10, 23},
		{"u8_n8", 8, 8, 18},
		{"u16_n4", 16, 4, 18},
		{"u1_n1", 1, 1, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := efBitsizePaper(c.u, c.n)
			if got != c.want {
				t.Fatalf("efBitsizePaper(%d, %d) = %d, want %d", c.u, c.n, got, c.want)
			}
		})
	}
}

func TestBitmapBitsizePaper(t *testing.T) {
	if got := bitmapBitsizePaper(100, 10); got != 100 {
		t.Fatalf("bitmapBitsizePaper(100, 10) = %d, want 100", got)
	}
}

func TestAllOnesBitsize(t *testing.T) {
	if got := allOnesBitsize(8, 8); got != 0 {
		t.Fatalf("allOnesBitsize(8, 8) = %d, want 0", got)
	}
	if got := allOnesBitsize(8, 7); got != ^uint64(0) {
		t.Fatalf("allOnesBitsize(8, 7) = %d, want max uint64", got)
	}
}

func TestMinCodecBitsize(t *testing.T) {
	if got := minCodecBitsize(8, 8); got != 0 {
		t.Fatalf("contiguous run picks all-ones: got %d, want 0", got)
	}
	if got := minCodecBitsize(11, 10); got != 12 {
		t.Fatalf("dense small chunk picks bitmap: got %d, want 11+1=12", got)
	}
	if got := minCodecBitsize(100, 10); got != 55 {
		t.Fatalf("sparse chunk picks EF: got %d, want 54+1=55", got)
	}
}
