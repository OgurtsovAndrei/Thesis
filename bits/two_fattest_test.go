package bits

import "testing"

func TestFast(t *testing.T) {
	t.Parallel()
	if MostSignificantBit(0) != -1 {
		t.Fatal("MostSignificantBit(0) failed")
	}

	if TwoFattest(0, 0) != 0 {
		t.Fatal("TwoFattest(0, 0] failed")
	}
	if TwoFattest(0, 6) != 4 {
		t.Fatal("TwoFattest(0, 6] failed")
	}
	if TwoFattest(0, 8) != 8 {
		t.Fatal("TwoFattest(0, 8] failed")
	}
	if TwoFattest(1, 8) != 8 {
		t.Fatal("TwoFattest(1, 8] failed")
	}
	if TwoFattest(0, 9) != 8 {
		t.Fatal("TwoFattest(0, 9] failed")
	}
	if TwoFattest(0, 4) != 4 {
		t.Fatal("TwoFattest(0, 4] failed")
	}
	if TwoFattest(0, 7) != 4 {
		t.Fatal("TwoFattest(0, 7] failed")
	}
	if TwoFattest(5, 7) != 6 {
		t.Fatal("TwoFattest(5, 7] failed")
	}
	if TwoFattest(4, 7) != 6 {
		t.Fatal("TwoFattest(4, 7] failed")
	}
	if TwoFattest(3, 7) != 4 {
		t.Fatal("TwoFattest(3, 7] failed")
	}
	if TwoFattest(10, 11) != 11 {
		t.Fatal("TwoFattest(10, 11] failed")
	}
	if TwoFattest(9, 11) != 10 {
		t.Fatal("TwoFattest(9, 11] failed")
	}
	if TwoFattest(^uint64(0), 0) != 0 {
		t.Fatal("TwoFattest(-1, 0] failed")
	}
	if TwoFattest(^uint64(0), 8) != 0 {
		t.Fatal("TwoFattest(-1, 8) failed")
	}
	if TwoFattest(7, 8) != 8 {
		t.Fatal("TwoFattest(7, 8] failed")
	}
	if TwoFattest(8, 8) != 0 {
		t.Fatal("TwoFattest(8, 8] failed")
	}
}
