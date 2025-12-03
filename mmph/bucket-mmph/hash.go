package bucket

import (
	"Thesis/errutil"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"

	"Thesis/bits"

	"Thesis/mmph/go-boomphf"
)

type MonotoneHash struct {
	bucketSize int

	// d0: KeyHash -> LCP Length (in bits)
	d0Table   *boomphf.H // overhead of 2-3 bits per key
	d0Lengths []uint16   // O(n * log w) bits

	// d1: PrefixHash -> Bucket Index
	d1Table   *boomphf.H // overhead of 2-3 bits per bucket
	d1Indices []int32    // O((n/b) * log(n/b)) <= O(n) bits, when b = log n

	// buckets: KeyHash -> Local Rank
	buckets     []*boomphf.H // overhead of 2-3 bits per key
	bucketRanks [][]uint8    // O(n log(b)) = O(b log(log(n))) bits, when b = log n
}

func NewMonotoneHash(data []bits.BitString) *MonotoneHash {
	if len(data) == 0 {
		return &MonotoneHash{}
	}

	bucketSize := max(int(math.Ceil(math.Log2(float64(len(data))))), 1)

	if bucketSize > 256 {
		panic("bucketSize must be <= 256 when using uint8 optimization")
	}

	n := len(data)
	numBuckets := (n + bucketSize - 1) / bucketSize

	mh := &MonotoneHash{
		bucketSize:  bucketSize,
		buckets:     make([]*boomphf.H, numBuckets),
		bucketRanks: make([][]uint8, numBuckets),
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

			mh.buckets[i] = boomphf.New(2.0, bucketHashes)

			for j, h := range bucketHashes {
				if idx := mh.buckets[i].Query(h); idx == 0 {
					panic(fmt.Sprintf("boomphf failed immediately on construction for bucket %d key %d", i, j))
				}
			}

			mh.bucketRanks[i] = make([]uint8, len(bucketKeys))
			for localRank, k := range bucketKeys {
				phfIdx := mh.buckets[i].Query(bitStringToHash(k)) - 1
				mh.bucketRanks[i][phfIdx] = uint8(localRank)
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

	return mh
}

// todo: remove, rewrite go-boomphf to use BitString's
func bitStringToHash(bs bits.BitString) uint64 {
	h := fnv.New64a()
	_, err := h.Write(bs.Data())
	errutil.FatalIf(err)

	err = binary.Write(h, binary.LittleEndian, bs.Size())
	errutil.FatalIf(err)
	return h.Sum64()
}

func (mh *MonotoneHash) GetRank(key bits.BitString) int {
	if mh.d0Table == nil {
		return -1
	}

	keyHash := bitStringToHash(key)

	d0PhfIdx := mh.d0Table.Query(keyHash)
	if d0PhfIdx == 0 || int(d0PhfIdx) > len(mh.d0Lengths) {
		return -1
	}
	lcpLen := int(mh.d0Lengths[d0PhfIdx-1])

	if int(key.Size()) < lcpLen {
		return -1
	}

	prefix := key.Prefix(lcpLen)
	prefixHash := bitStringToHash(prefix)

	d1PhfIdx := mh.d1Table.Query(prefixHash)
	if d1PhfIdx == 0 || int(d1PhfIdx) > len(mh.d1Indices) {
		errutil.Bug("d1Table miss for prefix %s (key %s). Likely hash collision or unsorted input.", prefix.String(), key.String())
		return -1
	}
	bucketIdx := int(mh.d1Indices[d1PhfIdx-1])

	if bucketIdx >= len(mh.buckets) || mh.buckets[bucketIdx] == nil {
		return -1
	}

	localPhfIdx := mh.buckets[bucketIdx].Query(keyHash)
	if localPhfIdx == 0 || int(localPhfIdx) > len(mh.bucketRanks[bucketIdx]) {
		errutil.Bug("Local bucket %d miss for key %s", bucketIdx, key.String())
		return -1
	}

	localOffset := int(mh.bucketRanks[bucketIdx][localPhfIdx-1])

	return bucketIdx*mh.bucketSize + localOffset
}

// Size returns the total size of the structure in bytes.
// It accounts for:
// - d0Table (MPHF) and d0Lengths (array)
// - d1Table (MPHF) and d1Indices (array)
// - buckets (MPHF per bucket) and bucketRanks (array per bucket)
func (mh *MonotoneHash) Size() int {
	var size = 4 // bucket size

	if mh.d0Table != nil {
		size += mh.d0Table.Size()
	}
	size += len(mh.d0Lengths) * 2

	if mh.d1Table != nil {
		size += mh.d1Table.Size()
	}
	size += len(mh.d1Indices) * 4

	for _, b := range mh.buckets {
		if b != nil {
			size += b.Size()
		}
	}
	for _, r := range mh.bucketRanks {
		size += len(r) * 1
	}
	return size
}
