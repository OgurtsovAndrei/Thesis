package are_hybrid

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func loadFBKeys(maxKeys int) ([]uint64, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	dataPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "bench", "sosd_data", "fb_200M_uint64")

	f, err := os.Open(dataPath)
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

	keys := make([]uint64, readN)
	if err := binary.Read(f, binary.LittleEndian, keys); err != nil {
		return nil, fmt.Errorf("read keys: %w", err)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	j := 0
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[j] {
			j++
			keys[j] = keys[i]
		}
	}
	return keys[:j+1], nil
}

func TestFB_GapHistogram(t *testing.T) {
	const n = 65536

	keys, err := loadFBKeys(n)
	if err != nil {
		t.Skipf("SOSD data not available: %v", err)
	}
	t.Logf("loaded %d keys (requested %d), range [%d, %d]", len(keys), n, keys[0], keys[len(keys)-1])

	// Compute gaps between consecutive sorted keys.
	gaps := make([]uint64, len(keys)-1)
	for i := 0; i < len(keys)-1; i++ {
		gaps[i] = keys[i+1] - keys[i]
	}

	// Sort gaps for percentile computation.
	sortedGaps := make([]uint64, len(gaps))
	copy(sortedGaps, gaps)
	sort.Slice(sortedGaps, func(i, j int) bool { return sortedGaps[i] < sortedGaps[j] })

	minGap := sortedGaps[0]
	maxGap := sortedGaps[len(sortedGaps)-1]
	medianGap := sortedGaps[len(sortedGaps)/2]
	p95Gap := sortedGaps[int(float64(len(sortedGaps))*0.95)]
	p99Gap := sortedGaps[int(float64(len(sortedGaps))*0.99)]

	var sumGap float64
	for _, g := range gaps {
		sumGap += float64(g)
	}
	meanGap := sumGap / float64(len(gaps))

	keyRange := keys[len(keys)-1] - keys[0]
	density := float64(len(keys)) / float64(keyRange)
	expectedGap := float64(keyRange) / float64(len(keys)-1)

	t.Logf("=== Summary Statistics ===")
	t.Logf("  N (after dedup):  %d", len(keys))
	t.Logf("  Total key range:  %d  (max-min)", keyRange)
	t.Logf("  Density (N/range): %.6e keys per unit", density)
	t.Logf("  Expected gap (uniform): %.2f", expectedGap)
	t.Logf("")
	t.Logf("  Min gap:    %d", minGap)
	t.Logf("  Max gap:    %d", maxGap)
	t.Logf("  Mean gap:   %.2f", meanGap)
	t.Logf("  Median gap: %d", medianGap)
	t.Logf("  P95 gap:    %d", p95Gap)
	t.Logf("  P99 gap:    %d", p99Gap)

	// How many split points does using P95 as a threshold create?
	// A gap >= p95Gap is treated as a cluster boundary.
	splitCount := 0
	for _, g := range gaps {
		if g >= p95Gap {
			splitCount++
		}
	}
	t.Logf("")
	t.Logf("  Using P95 gap (%d) as split threshold:", p95Gap)
	t.Logf("    Split points: %d  =>  %d segments", splitCount, splitCount+1)

	// Build log-scale histogram: buckets [2^k, 2^(k+1)) starting from 1.
	// Find the highest power of 2 needed to cover maxGap.
	maxBit := 0
	for (uint64(1) << uint(maxBit)) <= maxGap {
		maxBit++
	}

	type bucket struct {
		lo, hi uint64
		count  int
	}
	buckets := make([]bucket, maxBit)
	for k := 0; k < maxBit; k++ {
		buckets[k] = bucket{lo: uint64(1) << uint(k), hi: uint64(1) << uint(k+1)}
	}

	for _, g := range gaps {
		if g == 0 {
			continue
		}
		k := int(math.Log2(float64(g)))
		if k >= maxBit {
			k = maxBit - 1
		}
		buckets[k].count++
	}

	total := len(gaps)
	t.Logf("")
	t.Logf("=== Log-Scale Gap Histogram ===")
	t.Logf("%-22s  %8s  %7s  %s", "Bucket [lo, hi)", "Count", "Pct%", "Bar")

	maxCount := 0
	for _, b := range buckets {
		if b.count > maxCount {
			maxCount = b.count
		}
	}
	const barWidth = 50

	for _, b := range buckets {
		if b.count == 0 {
			continue
		}
		pct := 100.0 * float64(b.count) / float64(total)
		barLen := 0
		if maxCount > 0 {
			barLen = int(float64(b.count) / float64(maxCount) * barWidth)
		}
		bar := ""
		for i := 0; i < barLen; i++ {
			bar += "*"
		}
		t.Logf("[%9d, %9d)  %8d  %6.2f%%  %s", b.lo, b.hi, b.count, pct, bar)
	}
}
