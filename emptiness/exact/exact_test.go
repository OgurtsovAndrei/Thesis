package exact

import (
	"testing"
)

func TestVariantsMatchOnBasicQueries(t *testing.T) {
	keys := []uint64{1, 3, 5}
	const keyBits uint32 = 3

	classic, err := NewUint64WithVariant(keys, keyBits, VariantClassic)
	if err != nil {
		t.Fatalf("classic build failed: %v", err)
	}
	oneD, err := NewUint64WithVariant(keys, keyBits, VariantOneD)
	if err != nil {
		t.Fatalf("one_d build failed: %v", err)
	}

	queries := [][2]uint64{
		{1, 1},
		{2, 2},
		{1, 3},
		{4, 7},
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
