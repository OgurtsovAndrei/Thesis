package are_hybrid

import (
	"Thesis/bits"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// loadSOSDFB reads the first maxKeys uint64 values from the SOSD binary file.
// Format: [uint64 count][count × uint64 keys].
func loadSOSDFB(path string, maxKeys int) ([]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	readN := int(count)
	if maxKeys > 0 && maxKeys < readN {
		readN = maxKeys
	}

	raw := make([]uint64, readN)
	if err := binary.Read(f, binary.LittleEndian, raw); err != nil {
		return nil, fmt.Errorf("read keys: %w", err)
	}

	// Sort and deduplicate (SOSD files may have rare duplicates)
	sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })
	j := 0
	for i := 1; i < len(raw); i++ {
		if raw[i] != raw[j] {
			j++
			raw[j] = raw[i]
		}
	}
	return raw[:j+1], nil
}

func sosdFBPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "bench", "sosd_data", "fb_200M_uint64")
}

func TestHybrid_SOSD_FB_Diagnostic(t *testing.T) {
	const (
		N         = 65536
		queryCount = 10000
		eps       = 0.01
		maxL      = uint64(1024)
	)

	path := sosdFBPath()
	rawKeys, err := loadSOSDFB(path, N)
	if err != nil {
		t.Skipf("SOSD data not available: %v", err)
	}
	t.Logf("=== SOSD fb_200M: loaded %d keys, range [%d, %d] ===",
		len(rawKeys), rawKeys[0], rawKeys[len(rawKeys)-1])

	span := rawKeys[len(rawKeys)-1] - rawKeys[0]
	density := float64(len(rawKeys)) / float64(span)
	t.Logf("Key span: %d, density: %.6f keys/value (1 key per %.1f values)", span, density, 1.0/density)

	// Convert to 64-bit BitStrings (matching bench/comparison_test.go usage)
	bs := make([]bits.BitString, len(rawKeys))
	for i, k := range rawKeys {
		bs[i] = bits.NewFromTrieUint64(k, 64)
	}

	// --- Phase 1: Inspect detectClusters ---
	t.Log("--- detectClusters(0.95, 0.01) ---")
	segments, fallbackKeys := detectClusters(bs, 0.95, 0.01)
	t.Logf("Result: %d segments, %d fallback keys (%.2f%%)",
		len(segments), len(fallbackKeys), 100.0*float64(len(fallbackKeys))/float64(len(rawKeys)))

	totalClusterKeys := 0
	for i, seg := range segments {
		totalClusterKeys += len(seg.keys)
		segSpan := seg.maxKey - seg.minKey
		t.Logf("  segment[%d]: %d keys (%.1f%%), range [%d, %d], span=%d",
			i, len(seg.keys), 100.0*float64(len(seg.keys))/float64(len(rawKeys)),
			seg.minKey, seg.maxKey, segSpan)
	}
	t.Logf("Total keys in clusters: %d, fallback: %d", totalClusterKeys, len(fallbackKeys))

	// --- Phase 2: Build HybridARE for each range length ---
	for _, rangeLen := range []uint64{1, 16, 128, maxL} {
		t.Run(fmt.Sprintf("L=%d", rangeLen), func(t *testing.T) {
			h, err := NewHybridARE(bs, rangeLen, eps)
			if err != nil {
				t.Fatalf("NewHybridARE(eps=%.3f, L=%d): %v", eps, rangeLen, err)
			}

			nc, nf, nt := h.Stats()
			bpk := float64(h.SizeInBits()) / float64(nt)
			t.Logf("HybridARE: clusters=%d, fallback=%d, total=%d, BPK=%.4f, SizeInBits=%d",
				nc, nf, nt, bpk, h.SizeInBits())

			for i, c := range h.clusters {
				// Find the matching segment for key count
				nKeys := 0
				for _, seg := range segments {
					if seg.minKey == c.minKey {
						nKeys = len(seg.keys)
						break
					}
				}
				cBPK := float64(0)
				if nKeys > 0 {
					cBPK = float64(c.filter.SizeInBits()) / float64(nKeys)
				}
				t.Logf("  cluster[%d]: ~%d keys, range [%d, %d], span=%d, SizeInBits=%d, BPK=%.4f, IsExact=%v, K=%d",
					i, nKeys, c.minKey, c.maxKey, c.maxKey-c.minKey,
					c.filter.SizeInBits(), cBPK,
					c.filter.IsExactMode, c.filter.K)
			}
			if h.fallback != nil {
				fbBPK := float64(h.fallback.SizeInBits()) / float64(nf)
				t.Logf("  fallback: %d keys, SizeInBits=%d, BPK=%.4f", nf, h.fallback.SizeInBits(), fbBPK)
			}

			// --- Phase 3: False negative check — every key must not be reported empty ---
			fnCount := 0
			for _, k := range rawKeys {
				a := bits.NewFromTrieUint64(k, 64)
				if h.IsEmpty(a, a) {
					fnCount++
					if fnCount <= 5 {
						t.Errorf("FALSE NEGATIVE: key %d reported as empty", k)
					}
				}
			}
			if fnCount > 0 {
				t.Logf("CRITICAL: %d/%d keys produce false negatives!", fnCount, len(rawKeys))
			} else {
				t.Logf("False negatives: 0 (correct)")
			}

			// --- Phase 4: FPR measurement with truly empty ranges ---
			rng := rand.New(rand.NewSource(42))
			fp, totalEmpty, attempted := 0, 0, 0
			minK, maxK := rawKeys[0], rawKeys[len(rawKeys)-1]

			for totalEmpty < queryCount && attempted < queryCount*100 {
				attempted++
				var a, b uint64
				if maxK > minK {
					a = minK + uint64(rng.Int63n(int64(maxK-minK)))
				} else {
					a = minK
				}
				if a+rangeLen-1 < a { // overflow guard
					continue
				}
				b = a + rangeLen - 1

				// Ground truth: check if [a, b] contains any key
				idx := sort.Search(len(rawKeys), func(j int) bool { return rawKeys[j] >= a })
				if idx < len(rawKeys) && rawKeys[idx] <= b {
					continue // non-empty range, skip
				}
				totalEmpty++
				aBS := bits.NewFromTrieUint64(a, 64)
				bBS := bits.NewFromTrieUint64(b, 64)
				if !h.IsEmpty(aBS, bBS) {
					fp++
				}
			}

			if totalEmpty == 0 {
				t.Log("WARNING: could not generate any empty queries within key range")
			} else {
				fpr := float64(fp) / float64(totalEmpty)
				t.Logf("Empty-range FPR: %.6f (%d false positives / %d empty queries, target ε=%.3f)",
					fpr, fp, totalEmpty, eps)
				if fpr > 3*eps {
					t.Errorf("FPR %.6f exceeds 3*epsilon=%.3f", fpr, 3*eps)
				}
			}

			// --- Phase 5: Probe known non-keys (values between consecutive keys) ---
			// Check FPR for ranges [key[i]+1, key[i+1]-1] (gaps between keys, truly empty)
			gapFP, gapTotal := 0, 0
			for i := 0; i+1 < len(rawKeys) && gapTotal < 1000; i++ {
				lo := rawKeys[i] + 1
				hi := rawKeys[i+1] - 1
				if lo > hi {
					continue // consecutive keys, no gap
				}
				// Query [lo, lo] — a single non-key value
				gapTotal++
				aBS := bits.NewFromTrieUint64(lo, 64)
				if !h.IsEmpty(aBS, aBS) {
					gapFP++
				}
			}
			if gapTotal > 0 {
				gapFPR := float64(gapFP) / float64(gapTotal)
				t.Logf("Gap-point FPR (probing [key[i]+1, key[i]+1]): %.6f (%d/%d, target ε=%.3f)",
					gapFPR, gapFP, gapTotal, eps)
			} else {
				t.Log("No gaps found between consecutive keys (keys are truly consecutive)")
			}

			// --- Phase 6: Diagnosis — check cluster IsEmpty for a known-empty point ---
			// Pick a value not in rawKeys and probe each cluster filter individually
			var knownNonKey uint64
			if len(rawKeys) > 0 && rawKeys[0] > 0 {
				// Before all keys
				knownNonKey = 0
			} else if rawKeys[len(rawKeys)-1] < ^uint64(0) {
				// After all keys
				knownNonKey = rawKeys[len(rawKeys)-1] + 1
			}
			// Also try inside a gap if one exists
			for i := 0; i+1 < len(rawKeys); i++ {
				if rawKeys[i+1]-rawKeys[i] > 1 {
					knownNonKey = rawKeys[i] + 1
					break
				}
			}
			if knownNonKey != 0 {
				aBS := bits.NewFromTrieUint64(knownNonKey, 64)
				t.Logf("--- Diagnosing known non-key %d ---", knownNonKey)
				t.Logf("  HybridARE.IsEmpty(%d, %d) = %v (want true)",
					knownNonKey, knownNonKey, h.IsEmpty(aBS, aBS))
				for i, c := range h.clusters {
					clusterResult := c.filter.IsEmpty(aBS, aBS)
					t.Logf("  cluster[%d] (range [%d,%d]) IsEmpty = %v", i, c.minKey, c.maxKey, clusterResult)
				}
				if h.fallback != nil {
					t.Logf("  fallback.IsEmpty = %v", h.fallback.IsEmpty(aBS, aBS))
				}
			}
		})
	}

	// --- Phase 7: Verify key bit-width sensitivity ---
	// Rebuild with 60-bit keys (like clustered/zipfian benchmarks) and compare
	t.Run("60bit_keys_comparison", func(t *testing.T) {
		// Mask to 60 bits
		seen := make(map[uint64]bool, len(rawKeys))
		keys60 := make([]uint64, 0, len(rawKeys))
		for _, k := range rawKeys {
			k60 := k & ((uint64(1) << 60) - 1)
			if !seen[k60] {
				seen[k60] = true
				keys60 = append(keys60, k60)
			}
		}
		sort.Slice(keys60, func(i, j int) bool { return keys60[i] < keys60[j] })
		t.Logf("60-bit keys: %d (from %d original), range [%d, %d]", len(keys60), len(rawKeys), keys60[0], keys60[len(keys60)-1])

		bs60 := make([]bits.BitString, len(keys60))
		for i, k := range keys60 {
			bs60[i] = bits.NewFromTrieUint64(k, 64)
		}

		const rangeLen = uint64(1)
		h60, err := NewHybridARE(bs60, rangeLen, eps)
		if err != nil {
			t.Fatalf("build 60-bit: %v", err)
		}
		nc, nf, nt := h60.Stats()
		t.Logf("HybridARE (60-bit masked): clusters=%d, fallback=%d, total=%d, BPK=%.4f",
			nc, nf, nt, float64(h60.SizeInBits())/float64(nt))

		// Quick FPR spot-check with 60-bit masked queries
		rng := rand.New(rand.NewSource(42))
		fp, totalEmpty := 0, 0
		minK, maxK := keys60[0], keys60[len(keys60)-1]
		for totalEmpty < 1000 {
			a := minK + uint64(rng.Int63n(int64(maxK-minK)))
			b := a + rangeLen - 1
			if b < a {
				continue
			}
			idx := sort.Search(len(keys60), func(j int) bool { return keys60[j] >= a })
			if idx < len(keys60) && keys60[idx] <= b {
				continue
			}
			totalEmpty++
			if !h60.IsEmpty(bits.NewFromTrieUint64(a, 64), bits.NewFromTrieUint64(b, 64)) {
				fp++
			}
		}
		t.Logf("60-bit FPR: %.6f (%d/%d)", float64(fp)/float64(totalEmpty), fp, totalEmpty)
	})

	// --- Phase 8: Root cause confirmation ---
	// Hypothesis: the fallback ApproximateRangeEmptiness uses prefix truncation.
	// Keys are 64-bit BitStrings with values in [1, ~23M]. Since 23M < 2^25,
	// the top 39 bits of every key are all zeros. When K=24 prefix is taken,
	// ALL keys truncate to the zero bitstring. The ERE then always returns
	// non-empty for any query, causing FPR=1.0.
	t.Run("root_cause_prefix_truncation", func(t *testing.T) {
		// Compute K that ApproximateRangeEmptiness would use for 63127 fallback keys
		nFallback := 63127
		epsilon := 0.01
		val := 2.0 * float64(nFallback) / epsilon
		K := uint32(0)
		for float64(uint64(1)<<K) < val {
			K++
		}
		t.Logf("Fallback ApproximateRangeEmptiness: nFallback=%d, K=%d (top K bits of 64-bit strings)", nFallback, K)

		// Count bits needed to represent max key
		maxKeyVal := rawKeys[len(rawKeys)-1]
		bitsNeeded := 0
		for v := maxKeyVal; v > 0; v >>= 1 {
			bitsNeeded++
		}
		t.Logf("Max key: %d (= 0x%x), bits needed: ~%d", maxKeyVal, maxKeyVal, bitsNeeded)

		// For a 64-bit BitString, trie bit 0 is the MSB (most significant bit).
		// Prefix(K) takes the top K bits in trie order = the K most significant bits.
		// For keys <= 23M = ~0x161_8880:
		//   top K=24 bits out of 64 are: bits 63..40 in standard notation = all zeros.
		prefixes := make(map[string]int)
		for _, k := range rawKeys {
			bs := bits.NewFromTrieUint64(k, 64)
			pfx := bs.Prefix(int(K))
			key := fmt.Sprintf("%v", pfx.TrieUint64())
			prefixes[key]++
		}
		t.Logf("Distinct top-%d-bit prefixes across all %d keys: %d", K, len(rawKeys), len(prefixes))
		for pfxVal, count := range prefixes {
			t.Logf("  prefix=%s: %d keys", pfxVal, count)
		}

		if len(prefixes) == 1 {
			t.Logf("ROOT CAUSE CONFIRMED: All %d keys share the same top-%d-bit prefix.", len(rawKeys), K)
			t.Logf("The fallback ApproximateRangeEmptiness sees only 1 unique truncated key.")
			t.Logf("Every query [a,b] also has Prefix(%d)=0, so ERE always reports non-empty => FPR=1.0", K)
			t.Logf("")
			t.Logf("FIX OPTIONS:")
			t.Logf("  1. Use relative encoding in the fallback: subtract minKey before truncation.")
			t.Logf("     (Same approach as AdaptiveARE.NewAdaptiveARE — normalize to [0, spread] before hashing/truncating)")
			t.Logf("  2. Detect when all keys share the same prefix and fall back to AdaptiveARE instead.")
			t.Logf("  3. In HybridARE, pass the global key min/max to the fallback builder so it can normalize.")
		}
	})
}
