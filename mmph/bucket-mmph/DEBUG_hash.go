package bucket

import (
	"Thesis/errutil"
	"fmt"
	"math"

	"Thesis/bits"

	"github.com/dgryski/go-boomphf"
)

type DebugMonotoneHash struct {
	bucketSize int

	// d0: KeyHash -> LCP Length (in bits)
	d0Table    *boomphf.H
	d0Lengths  []uint16
	d0DebugMap map[bits.BitString]uint16 // DEBUG

	// d1: PrefixHash -> Bucket Index
	d1Table    *boomphf.H
	d1Indices  []int32
	d1DebugMap map[bits.BitString]int32 // DEBUG

	// buckets: KeyHash -> Local Rank
	buckets         []*boomphf.H
	bucketRanks     [][]uint8
	bucketsDebugMap []map[bits.BitString]uint8 // DEBUG
}

func NewDebugMonotoneHash(data []bits.BitString) *DebugMonotoneHash {
	if len(data) == 0 {
		return &DebugMonotoneHash{}
	}

	bucketSize := max(int(math.Ceil(math.Log2(float64(len(data))))), 1)

	if bucketSize > 256 {
		panic("bucketSize must be <= 256 when using uint8 optimization")
	}

	n := len(data)
	numBuckets := (n + bucketSize - 1) / bucketSize

	mh := &DebugMonotoneHash{
		bucketSize:      bucketSize,
		buckets:         make([]*boomphf.H, numBuckets),
		bucketRanks:     make([][]uint8, numBuckets),
		bucketsDebugMap: make([]map[bits.BitString]uint8, numBuckets), // DEBUG
		d1DebugMap:      make(map[bits.BitString]int32),               // DEBUG
		d0DebugMap:      make(map[bits.BitString]uint16),              // DEBUG
	}

	var allKeys []bits.BitString
	var allLcps []bits.BitString

	keyToLcpLen := make(map[bits.BitString]int, n)
	prefixToBucketIdx := make(map[bits.BitString]int, numBuckets)

	for i := 0; i < numBuckets; i++ {
		start := i * bucketSize
		end := start + bucketSize
		if end > n {
			end = n
		}

		bucketKeys := data[start:end]

		if len(bucketKeys) > 0 {
			bucketHashes := make([]uint64, len(bucketKeys))
			for j, k := range bucketKeys {
				bucketHashes[j] = bitStringToHash(k)
			}

			gamma := 2.0
			if len(bucketKeys) < 10 {
				gamma = 10.0
			}
			mh.buckets[i] = boomphf.New(gamma, bucketHashes)

			for j, h := range bucketHashes {
				if idx := mh.buckets[i].Query(h); idx == 0 {
					panic(fmt.Sprintf("boomphf failed immediately on construction for bucket %d key %d", i, j))
				}
			}

			mh.bucketsDebugMap[i] = make(map[bits.BitString]uint8)

			mh.bucketRanks[i] = make([]uint8, len(bucketKeys))
			for localRank, k := range bucketKeys {
				phfIdx := mh.buckets[i].Query(bitStringToHash(k)) - 1
				mh.bucketRanks[i][phfIdx] = uint8(localRank)
				mh.bucketsDebugMap[i][k] = uint8(localRank)
			}

			lcp := bucketKeys[0]
			for _, k := range bucketKeys[1:] {
				lcpLen := lcp.GetLCPLength(k)
				lcp = lcp.Prefix(int(lcpLen))
			}

			if _, exists := prefixToBucketIdx[lcp]; !exists {
				allLcps = append(allLcps, lcp)
				prefixToBucketIdx[lcp] = i
			}

			lcpSize := int(lcp.Size())
			for _, k := range bucketKeys {
				keyToLcpLen[k] = lcpSize
			}
			allKeys = append(allKeys, bucketKeys...)
		}
	}

	for p, idx := range prefixToBucketIdx {
		mh.d1DebugMap[p] = int32(idx)
	}

	if len(allKeys) > 0 {
		allKeyHashes := make([]uint64, len(allKeys))
		for i, k := range allKeys {
			allKeyHashes[i] = bitStringToHash(k)
		}
		mh.d0Table = boomphf.New(2.0, allKeyHashes)

		mh.d0Lengths = make([]uint16, len(allKeys))
		for _, k := range allKeys {
			phfIdx := mh.d0Table.Query(bitStringToHash(k)) - 1
			if phfIdx+1 == 0 {
				panic("boomphf construction failure: Query returned 0 for known key in d0Table")
			}
			mh.d0Lengths[phfIdx] = uint16(keyToLcpLen[k])
			mh.d0DebugMap[k] = uint16(keyToLcpLen[k])
		}
	}

	if len(allLcps) > 0 {
		lcpHashes := make([]uint64, len(allLcps))
		for i, p := range allLcps {
			lcpHashes[i] = bitStringToHash(p)
		}
		mh.d1Table = boomphf.New(2.0, lcpHashes)

		mh.d1Indices = make([]int32, len(allLcps))
		for _, p := range allLcps {
			phfIdx := mh.d1Table.Query(bitStringToHash(p)) - 1
			mh.d1Indices[phfIdx] = int32(prefixToBucketIdx[p])
		}
	}

	fmt.Println("=== DEBUG: Constructed DebugMonotoneHash Maps ===")
	fmt.Println("--- D0 (Key -> LCP Length) ---")
	for k, v := range mh.d0DebugMap {
		fmt.Printf("Key: %s, Hash: %x, LCP Len: %d\n", k.String(), bitStringToHash(k), v)
	}
	fmt.Println("--- D1 (Prefix -> Bucket Index) ---")
	for k, v := range mh.d1DebugMap {
		fmt.Printf("Prefix: %s, Hash: %x, Bucket: %d\n", k.String(), bitStringToHash(k), v)
	}
	fmt.Println("--- Buckets (Key -> Local Rank) ---")
	for i, m := range mh.bucketsDebugMap {
		fmt.Printf("Bucket %d:\n", i)
		for k, v := range m {
			fmt.Printf("  Key: %s, Hash: %x, Rank: %d\n", k.String(), bitStringToHash(k), v)
		}
	}
	fmt.Println("============================================")

	return mh
}

func (mh *DebugMonotoneHash) GetRank(key bits.BitString) int {
	fmt.Printf("\n--- GetRank for %s (Hash: %x) ---\n", key.String(), bitStringToHash(key))

	if mh.d0Table == nil {
		return -1
	}

	keyHash := bitStringToHash(key)

	d0PhfIdx := mh.d0Table.Query(keyHash)
	lcpLen := -1
	if d0PhfIdx != 0 && int(d0PhfIdx) <= len(mh.d0Lengths) {
		lcpLen = int(mh.d0Lengths[d0PhfIdx-1])
	}
	fmt.Printf("1. D0 Lookup: Hash=%x -> PHF_Idx=%d -> LCP_Len=%d (Expected: %d)\n", keyHash, d0PhfIdx, lcpLen, mh.d0DebugMap[key])

	if d0PhfIdx == 0 || int(d0PhfIdx) > len(mh.d0Lengths) {
		errutil.BugOn(true, "d0Table miss for key %s", key)
		return -1
	}

	expectedLcpLen, ok := mh.d0DebugMap[key]
	if !ok {
		errutil.BugOn(true, "DEBUG: Key %s not found in d0DebugMap!", key)
	} else if uint16(lcpLen) != expectedLcpLen {
		errutil.BugOn(true, "DEBUG: d0 mismatch for key %s. PHF got %d, Map got %d", key, lcpLen, expectedLcpLen)
	}

	if int(key.Size()) < lcpLen {
		return -1
	}

	prefix := key.Prefix(lcpLen)
	prefixHash := bitStringToHash(prefix)

	d1PhfIdx := mh.d1Table.Query(prefixHash)
	bucketIdx := -1
	if d1PhfIdx != 0 && int(d1PhfIdx) <= len(mh.d1Indices) {
		bucketIdx = int(mh.d1Indices[d1PhfIdx-1])
	}
	fmt.Printf("2. D1 Lookup: Prefix=%s, Hash=%x -> PHF_Idx=%d -> Bucket=%d (Expected: %d)\n", prefix.String(), prefixHash, d1PhfIdx, bucketIdx, mh.d1DebugMap[prefix])

	if d1PhfIdx == 0 || int(d1PhfIdx) > len(mh.d1Indices) {
		errutil.BugOn(true, "d1Table miss for prefix %s (key %s)", prefix, key)
		return -1
	}

	expectedBucketIdx, ok := mh.d1DebugMap[prefix]
	if !ok {
		errutil.BugOn(true, "DEBUG: Prefix %s not found in d1DebugMap! (key %s)", prefix, key)
	} else if int32(bucketIdx) != expectedBucketIdx {
		errutil.BugOn(true, "DEBUG: d1 mismatch for prefix %s. PHF got %d, Map got %d", prefix, bucketIdx, expectedBucketIdx)
	}

	if bucketIdx >= len(mh.buckets) || mh.buckets[bucketIdx] == nil {
		return -1
	}

	localPhfIdx := mh.buckets[bucketIdx].Query(keyHash)
	localOffset := -1
	if localPhfIdx != 0 && int(localPhfIdx) <= len(mh.bucketRanks[bucketIdx]) {
		localOffset = int(mh.bucketRanks[bucketIdx][localPhfIdx-1])
	}
	fmt.Printf("3. Bucket %d Lookup: Hash=%x -> PHF_Idx=%d -> LocalRank=%d (Expected: %d)\n", bucketIdx, keyHash, localPhfIdx, localOffset, mh.bucketsDebugMap[bucketIdx][key])

	if localPhfIdx == 0 || int(localPhfIdx) > len(mh.bucketRanks[bucketIdx]) {
		errutil.BugOn(true, "Local bucket %d miss for key %s", bucketIdx, key)
		return -1
	}

	expectedOffset, ok := mh.bucketsDebugMap[bucketIdx][key]
	if !ok {
		errutil.BugOn(true, "DEBUG: Key %s not found in bucketsDebugMap[%d]!", key, bucketIdx)
	} else if uint8(localOffset) != expectedOffset {
		errutil.BugOn(true, "DEBUG: Local rank mismatch for key %s in bucket %d. PHF got %d, Map got %d", key, bucketIdx, localOffset, expectedOffset)
	}

	return bucketIdx*mh.bucketSize + localOffset
}
