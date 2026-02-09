package azft

import (
	"Thesis/bits"
	"Thesis/trie/zft"
	"math/rand"
	"testing"
	"time"
)

// TestStreamingAZFT_MatchesHeavyBuilder verifies streaming builder produces
// identical results to the heavy ZFT-based builder.
func TestStreamingAZFT_MatchesHeavyBuilder(t *testing.T) {
	t.Parallel()

	const runs = 200
	const maxKeys = 32
	const maxBitLen = 24

	for run := 0; run < runs; run++ {
		seed := time.Now().UnixNano() + int64(run)
		r := rand.New(rand.NewSource(seed))

		numKeys := r.Intn(maxKeys-1) + 2
		bitLen := r.Intn(maxBitLen-8) + 8
		keys := zft.GenerateRandomBitStrings(numKeys, bitLen, r)

		fixedSeed := uint64(seed) // Use same seed for both builders

		// Build using heavy ZFT
		heavyAZFT, err := NewApproxZFastTrieWithSeed[uint16, uint32, uint32](keys, false, fixedSeed)
		if err != nil {
			t.Fatalf("Heavy builder failed: %v (seed: %d)", err, seed)
		}

		// Build using streaming
		streamingAZFT, err := NewApproxZFastTrieStreamingWithSeed[uint16, uint32, uint32](keys, false, fixedSeed)
		if err != nil {
			t.Fatalf("Streaming builder failed: %v (seed: %d)", err, seed)
		}

		// Handle empty cases
		if heavyAZFT == nil && streamingAZFT == nil {
			continue
		}
		if (heavyAZFT == nil) != (streamingAZFT == nil) {
			t.Fatalf("One builder returned nil, other didn't (seed: %d)", seed)
		}

		// Compare data lengths
		if len(heavyAZFT.data) != len(streamingAZFT.data) {
			t.Fatalf("Data length mismatch: heavy=%d, streaming=%d (seed: %d)",
				len(heavyAZFT.data), len(streamingAZFT.data), seed)
		}

		// Test behavior on all prefixes of all keys
		for _, key := range keys {
			for prefixLen := 1; prefixLen <= int(key.Size()); prefixLen++ {
				prefix := key.Prefix(prefixLen)

				heavyResult := heavyAZFT.GetExistingPrefix(prefix)
				streamingResult := streamingAZFT.GetExistingPrefix(prefix)

				// Compare extentLen
				if heavyResult == nil && streamingResult == nil {
					continue
				}
				if heavyResult == nil || streamingResult == nil {
					t.Errorf("GetExistingPrefix nil mismatch for prefix %s (seed: %d)",
						prefix.PrettyString(), seed)
					continue
				}
				if heavyResult.extentLen != streamingResult.extentLen {
					t.Errorf("extentLen mismatch for prefix %s: heavy=%d, streaming=%d (seed: %d)",
						prefix.PrettyString(), heavyResult.extentLen, streamingResult.extentLen, seed)
				}
			}
		}

		// Test LowerBound on all keys
		for _, key := range keys {
			heavyCands := heavyAZFT.GetExistingPrefix(key)
			streamingCands := streamingAZFT.GetExistingPrefix(key)

			if (heavyCands == nil) != (streamingCands == nil) {
				t.Errorf("LowerBound result mismatch for key %s (seed: %d)",
					key.PrettyString(), seed)
			}
		}
	}
}

// TestStreamingAZFT_EmptyInput tests streaming builder with empty input.
func TestStreamingAZFT_EmptyInput(t *testing.T) {
	keys := []bits.BitString{}

	azft, err := NewApproxZFastTrieStreaming[uint16, uint32, uint32](keys, false)
	if err != nil {
		t.Fatalf("Streaming builder failed on empty input: %v", err)
	}

	if azft == nil {
		t.Errorf("Expected non-nil empty AZFT, got nil")
	}
}

// TestStreamingAZFT_SingleKey tests streaming builder with single key.
func TestStreamingAZFT_SingleKey(t *testing.T) {
	key := bits.NewFromBinary("10110")
	keys := []bits.BitString{key}
	seed := uint64(12345)

	heavyAZFT, _ := NewApproxZFastTrieWithSeed[uint16, uint32, uint32](keys, false, seed)
	streamingAZFT, err := NewApproxZFastTrieStreamingWithSeed[uint16, uint32, uint32](keys, false, seed)

	if err != nil {
		t.Fatalf("Streaming builder failed: %v", err)
	}

	if (heavyAZFT == nil) != (streamingAZFT == nil) {
		t.Fatalf("Nil mismatch: heavy=%v, streaming=%v", heavyAZFT == nil, streamingAZFT == nil)
	}

	if heavyAZFT == nil {
		return
	}

	// Test query
	result := streamingAZFT.GetExistingPrefix(key)
	heavyResult := heavyAZFT.GetExistingPrefix(key)

	if (result == nil) != (heavyResult == nil) {
		t.Fatalf("Single key query nil mismatch")
	}

	if result != nil && heavyResult != nil && result.extentLen != heavyResult.extentLen {
		t.Errorf("Single key query mismatch: heavy=%d, streaming=%d",
			heavyResult.extentLen, result.extentLen)
	}
}

// TestStreamingAZFT_TwoKeys tests streaming builder with two keys.
func TestStreamingAZFT_TwoKeys(t *testing.T) {
	// Two keys with common prefix
	key1 := bits.NewFromBinary("10110")
	key2 := bits.NewFromBinary("10111")
	keys := []bits.BitString{key1, key2}
	seed := uint64(12345)

	heavyAZFT, _ := NewApproxZFastTrieWithSeed[uint16, uint32, uint32](keys, false, seed)
	streamingAZFT, err := NewApproxZFastTrieStreamingWithSeed[uint16, uint32, uint32](keys, false, seed)

	if err != nil {
		t.Fatalf("Streaming builder failed: %v", err)
	}

	if heavyAZFT == nil || streamingAZFT == nil {
		t.Fatalf("Unexpected nil")
	}

	// Test all prefixes
	for _, key := range keys {
		for prefixLen := 1; prefixLen <= int(key.Size()); prefixLen++ {
			prefix := key.Prefix(prefixLen)

			heavyResult := heavyAZFT.GetExistingPrefix(prefix)
			streamingResult := streamingAZFT.GetExistingPrefix(prefix)

			if (heavyResult == nil) != (streamingResult == nil) {
				t.Errorf("Nil mismatch for prefix %s", prefix.PrettyString())
				continue
			}

			if heavyResult != nil && streamingResult != nil && heavyResult.extentLen != streamingResult.extentLen {
				t.Errorf("Mismatch for prefix %s: heavy=%d, streaming=%d",
					prefix.PrettyString(), heavyResult.extentLen, streamingResult.extentLen)
			}
		}
	}
}

// BenchmarkStreamingVsHeavyAZFT benchmarks memory and time for both approaches.
func BenchmarkStreamingVsHeavyAZFT(b *testing.B) {
	r := rand.New(rand.NewSource(42))
	keys := zft.GenerateRandomBitStrings(1000, 64, r)
	seed := uint64(42)

	b.Run("Heavy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewApproxZFastTrieWithSeed[uint16, uint32, uint32](keys, false, seed)
		}
	})

	b.Run("Streaming", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewApproxZFastTrieStreamingWithSeed[uint16, uint32, uint32](keys, false, seed)
		}
	})
}
