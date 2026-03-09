package testutils

import (
	"testing"
)

func TestGenSpreadKeys_CountAndSorted(t *testing.T) {
	n := 100
	keys := GenSpreadKeys(n, 10, 42)
	if len(keys) != n {
		t.Fatalf("expected %d keys, got %d", n, len(keys))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i].Compare(keys[i-1]) < 0 {
			t.Fatalf("keys not sorted at index %d", i)
		}
	}
}

func TestGenClusteredKeys_CountAndSorted(t *testing.T) {
	n := 200
	clusters := 5
	keys := GenClusteredKeys(n, clusters, 42)
	if len(keys) != n {
		t.Fatalf("expected %d keys, got %d", n, len(keys))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i].Compare(keys[i-1]) < 0 {
			t.Fatalf("keys not sorted at index %d", i)
		}
	}
}

func TestGenClusteredKeys_Uniqueness(t *testing.T) {
	n := 500
	keys := GenClusteredKeys(n, 10, 99)
	seen := make(map[string]bool, n)
	for _, k := range keys {
		s := string(k.Data())
		if seen[s] {
			t.Fatal("duplicate key found in clustered generation")
		}
		seen[s] = true
	}
}

func TestGenGapQueries_BetweenKeys(t *testing.T) {
	keys := GenSpreadKeys(100, 20, 1)
	queries := GenGapQueries(keys, 50, 1000, 42)
	if len(queries) == 0 {
		t.Fatal("expected at least one gap query")
	}
	for i, q := range queries {
		loVal := leUint64(q[0].Data())
		hiVal := leUint64(q[1].Data())
		if loVal > hiVal {
			t.Fatalf("query %d: lo (%d) > hi (%d)", i, loVal, hiVal)
		}
	}
}

func TestGenGapQueries_FewKeys(t *testing.T) {
	keys := GenSpreadKeys(1, 10, 1)
	queries := GenGapQueries(keys, 10, 100, 42)
	if len(queries) != 0 {
		t.Fatalf("expected 0 queries for 1 key, got %d", len(queries))
	}
}

func TestLeUint64_Roundtrip(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	v := leUint64(data)
	expected := uint64(0x0807060504030201)
	if v != expected {
		t.Fatalf("expected %x, got %x", expected, v)
	}
}

func TestLeUint64_ShortData(t *testing.T) {
	data := []byte{0xFF, 0x00}
	v := leUint64(data)
	if v != 0xFF {
		t.Fatalf("expected 0xFF, got %x", v)
	}
}
