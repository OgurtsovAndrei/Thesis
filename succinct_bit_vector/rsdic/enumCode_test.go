package rsdic

import (
	"math/rand"
	"testing"
)

func runTestEnumCode(t *testing.T, x uint64) {
	rankSB := popCount(x)
	code := enumEncode(x, rankSB)

	decoded := enumDecode(code, rankSB)
	if decoded != x {
		t.Fatalf("enumDecode(enumEncode(%x)): got %x, want %x", x, decoded, x)
	}

	for i := uint8(0); i < 64; i++ {
		got := enumBit(code, rankSB, i)
		want := getBit(x, i)
		if got != want {
			t.Fatalf("enumBit(%x, pos=%d): got %v, want %v", x, i, got, want)
		}
	}

	rank := uint8(0)
	for i := uint8(0); i < 64; i++ {
		got := enumRank(code, rankSB, i)
		if got != rank {
			t.Fatalf("enumRank(%x, pos=%d): got %d, want %d", x, i, got, rank)
		}
		if getBit(x, i) {
			rank++
		}
	}

	oneNum := uint8(0)
	zeroNum := uint8(0)
	for i := uint8(0); i < 64; i++ {
		if getBit(x, i) {
			oneNum++
			got := enumSelect(code, rankSB, oneNum, true)
			if got != i {
				t.Fatalf("enumSelect1(%x, rank=%d): got %d, want %d", x, oneNum, got, i)
			}
		} else {
			zeroNum++
			got := enumSelect(code, rankSB, zeroNum, false)
			if got != i {
				t.Fatalf("enumSelect0(%x, rank=%d): got %d, want %d", x, zeroNum, got, i)
			}
		}
	}
}

func TestEnumCode(t *testing.T) {
	runTestEnumCode(t, 0)
	for pc := 0; pc < 64; pc++ {
		for i := 0; i < 2; i++ {
			x := uint64(0)
			for j := 0; j < pc; j++ {
				pos := uint8(rand.Intn(64))
				x |= (1 << pos)
			}
			runTestEnumCode(t, x)
		}
	}
}
