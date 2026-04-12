package exact

import (
	"Thesis/bits"
	"testing"
)

func TestVariantsMatchOnBasicQueries(t *testing.T) {
	keys := []bits.BitString{
		bits.NewFromBinary("001"),
		bits.NewFromBinary("011"),
		bits.NewFromBinary("101"),
	}
	universe := bits.NewBitString(3)

	classic, err := NewWithVariant(keys, universe, VariantClassic)
	if err != nil {
		t.Fatalf("classic build failed: %v", err)
	}
	oneD, err := NewWithVariant(keys, universe, VariantOneD)
	if err != nil {
		t.Fatalf("one_d build failed: %v", err)
	}

	queries := [][2]bits.BitString{
		{bits.NewFromBinary("001"), bits.NewFromBinary("001")},
		{bits.NewFromBinary("010"), bits.NewFromBinary("010")},
		{bits.NewFromBinary("001"), bits.NewFromBinary("011")},
		{bits.NewFromBinary("100"), bits.NewFromBinary("111")},
	}
	for i, q := range queries {
		gotClassic := classic.IsEmpty(q[0], q[1])
		gotOneD := oneD.IsEmpty(q[0], q[1])
		if gotClassic != gotOneD {
			t.Fatalf("query %d mismatch: classic=%v one_d=%v", i, gotClassic, gotOneD)
		}
	}
}

func TestSetVariant(t *testing.T) {
	old := CurrentVariant()
	defer func() {
		_ = SetVariant(old)
	}()

	if err := SetVariant(VariantOneD); err != nil {
		t.Fatalf("SetVariant(one_d): %v", err)
	}
	if CurrentVariant() != VariantOneD {
		t.Fatalf("CurrentVariant = %v, want %v", CurrentVariant(), VariantOneD)
	}

	if err := SetVariantByName("classic"); err != nil {
		t.Fatalf("SetVariantByName(classic): %v", err)
	}
	if CurrentVariant() != VariantClassic {
		t.Fatalf("CurrentVariant = %v, want %v", CurrentVariant(), VariantClassic)
	}
}
