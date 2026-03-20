package are_greedy_scan

import (
	"Thesis/bits"
	"math"
)

// segment holds the final key slice for building AdaptiveARE.
type segment struct {
	keys   []bits.BitString
	minKey uint64
	maxKey uint64
}

// segmentRef is a lightweight reference into the original sorted key array.
// Used during segmentation and merge — no key copying until finalize.
type segmentRef struct {
	start, end     int // indices into original keys [start, end)
	minKey, maxKey uint64
}

func (r segmentRef) nKeys() int { return r.end - r.start }

// segmentBySpreadRefs scans sorted keys left-to-right, creating a new cluster
// whenever the spread would exceed 2^K. O(n) time, every key assigned.
// Returns lightweight refs — no key copying.
func segmentBySpreadRefs(keys []bits.BitString, K uint32) []segmentRef {
	n := len(keys)
	if n == 0 {
		return nil
	}

	var maxSpread uint64
	if K >= 64 {
		maxSpread = ^uint64(0)
	} else {
		maxSpread = (uint64(1) << K) - 1
	}

	var refs []segmentRef
	clusterStart := 0
	startVal := keys[0].TrieUint64()

	for i := 1; i < n; i++ {
		curVal := keys[i].TrieUint64()
		if curVal-startVal > maxSpread {
			refs = append(refs, segmentRef{
				start:  clusterStart,
				end:    i,
				minKey: startVal,
				maxKey: keys[i-1].TrieUint64(),
			})
			clusterStart = i
			startVal = curVal
		}
	}
	refs = append(refs, segmentRef{
		start:  clusterStart,
		end:    n,
		minKey: keys[clusterStart].TrieUint64(),
		maxKey: keys[n-1].TrieUint64(),
	})

	return refs
}

// estimateRefCost estimates the storage cost (bits) of a segment ref.
// A cluster with spread S and n keys uses:
//   - exact (ERE) mode if S < 2^K: cost ≈ n * log2(2^K / n) bits
//   - SODA mode if S >= 2^K:       cost ≈ n * K bits
//
// Plus 128 bits for min/max boundary overhead per cluster.
func estimateRefCost(nKeys int, minKey, maxKey uint64, K uint32) float64 {
	n := float64(nKeys)
	if n == 0 {
		return 128
	}
	spread := maxKey - minKey + 1
	var universe float64
	if K >= 64 {
		universe = math.Exp2(64)
	} else {
		universe = math.Exp2(float64(K))
	}
	var dataCost float64
	if float64(spread) < universe {
		ratio := universe / n
		if ratio < 1 {
			ratio = 1
		}
		dataCost = n * math.Log2(ratio)
	} else {
		dataCost = n * float64(K)
	}
	return dataCost + 128
}

// mergeRefs combines two adjacent segment refs. O(1), zero allocation.
func mergeRefs(a, b segmentRef) segmentRef {
	return segmentRef{
		start:  a.start,
		end:    b.end,
		minKey: a.minKey,
		maxKey: b.maxKey,
	}
}

// mergeSmallClustersRefs performs greedy bottom-up merge passes over segment refs.
// Two adjacent refs A and B are merged whenever cost(A∪B) < cost(A)+cost(B).
// Repeats until no more beneficial merges. Each pass builds a new slice (no in-place
// shifting), so each pass is O(n) regardless of merge count.
func mergeSmallClustersRefs(refs []segmentRef, K uint32) []segmentRef {
	for {
		out := make([]segmentRef, 0, len(refs))
		merged := false
		i := 0
		for i < len(refs) {
			if i == len(refs)-1 {
				out = append(out, refs[i])
				i++
				continue
			}
			costA := estimateRefCost(refs[i].nKeys(), refs[i].minKey, refs[i].maxKey, K)
			costB := estimateRefCost(refs[i+1].nKeys(), refs[i+1].minKey, refs[i+1].maxKey, K)
			m := mergeRefs(refs[i], refs[i+1])
			costAB := estimateRefCost(m.nKeys(), m.minKey, m.maxKey, K)
			if costAB < costA+costB {
				out = append(out, m)
				i += 2
				merged = true
			} else {
				out = append(out, refs[i])
				i++
			}
		}
		refs = out
		if !merged {
			break
		}
	}
	return refs
}

// finalizeRefs converts segment refs back to segments with actual key slices.
func finalizeRefs(keys []bits.BitString, refs []segmentRef) []segment {
	segs := make([]segment, len(refs))
	for i, r := range refs {
		segs[i] = segment{
			keys:   keys[r.start:r.end],
			minKey: r.minKey,
			maxKey: r.maxKey,
		}
	}
	return segs
}
