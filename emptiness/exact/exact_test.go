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
	pef, err := NewUint64WithVariant(keys, keyBits, VariantPEF)
	if err != nil {
		t.Fatalf("pef build failed: %v", err)
	}
	auto, err := NewUint64WithVariant(keys, keyBits, VariantAuto)
	if err != nil {
		t.Fatalf("auto build failed: %v", err)
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
		gotPEF := pef.IsEmpty(q[0], q[1])
		gotAuto := auto.IsEmpty(q[0], q[1])
		if gotClassic != gotOneD || gotOneD != gotPEF || gotPEF != gotAuto {
			t.Fatalf("query %d mismatch: classic=%v one_d=%v pef=%v auto=%v",
				i, gotClassic, gotOneD, gotPEF, gotAuto)
		}
	}
}

func TestParseVariantPEFAndAuto(t *testing.T) {
	cases := map[string]Variant{
		"":          VariantAuto,
		"auto":      VariantAuto,
		"AUTO":      VariantAuto,
		"pef":       VariantPEF,
		"ere_pef":   VariantPEF,
		"one_d":     VariantOneD,
		"ere_one_d": VariantOneD,
		"classic":   VariantClassic,
	}
	for s, want := range cases {
		got, err := ParseVariant(s)
		if err != nil {
			t.Fatalf("ParseVariant(%q): %v", s, err)
		}
		if got != want {
			t.Fatalf("ParseVariant(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestAutoSelectsPEFForSmallInputs(t *testing.T) {
	// At small N, VariantAuto must pick PEF (which exposes a non-zero
	// chunk count via its synthesized Stats), not OneD.
	keys := make([]uint64, 1024)
	for i := range keys {
		keys[i] = uint64(i * 7)
	}
	f, err := NewUint64WithVariant(keys, 16, VariantAuto)
	if err != nil {
		t.Fatalf("auto build failed: %v", err)
	}
	if _, ok := f.(pefFilter); !ok {
		t.Fatalf("VariantAuto on small N should pick PEF, got %T", f)
	}
}

func TestAutoFallsBackToOneDForWideKeys(t *testing.T) {
	// PEF now supports the full 2^64 universe (AutoPEFMaxKeyBits == 64),
	// so keyBits=64 should use PEF for small inputs.
	keys := []uint64{0, 1 << 60, 1 << 61, 1 << 62, 1 << 63}
	f, err := NewUint64WithVariant(keys, 64, VariantAuto)
	if err != nil {
		t.Fatalf("auto build failed: %v", err)
	}
	if _, ok := f.(pefFilter); !ok {
		t.Fatalf("VariantAuto on keyBits=64 with small N should pick PEF, got %T", f)
	}
}

func TestResolveAutoVariantLargeNPicksOneD(t *testing.T) {
	// VariantAuto falls back to OneD when N > AutoPEFThreshold regardless
	// of keyBits (OneD is 2-3× faster at query time for large inputs).
	if got := resolveAutoVariant(AutoPEFThreshold+1, 64); got != VariantOneD {
		t.Fatalf("N>AutoPEFThreshold should pick OneD, got %v", got)
	}
	if got := resolveAutoVariant(AutoPEFThreshold, 64); got != VariantPEF {
		t.Fatalf("N==AutoPEFThreshold should pick PEF, got %v", got)
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
