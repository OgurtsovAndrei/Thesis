package are_trunc

import (
	"Thesis/emptiness/ere_one_d"
	"Thesis/errutil"
	"Thesis/utils"
	"fmt"
	mathbits "math/bits"
	"math"
	"sort"
)

// Config holds the construction parameters for TruncARE.
type Config struct {
	Eps float64
}

// TruncARE is a probabilistic data structure that answers 1D range emptiness
// queries with a guaranteed upper bound on the false positive probability (epsilon).
// Uses prefix truncation with key normalization: keys are shifted relative to minKey so that
// the spread occupies all K bits effectively (avoids all-zero-prefix collapse for small-valued keys).
type TruncARE struct {
	exact     *ere_one_d.ExactRangeEmptiness
	K         uint32
	keyBits   uint32
	minKey    uint64
	maxKey    uint64
	spreadLen uint32 // Len64(maxKey - minKey): number of significant bits in (maxKey - minKey)
}

// NewTruncARE builds a TruncARE from a copy of keys (sorted and deduplicated internally).
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewTruncARE(keys []uint64, keyBits uint32, cfg Config) (*TruncARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	cp := append([]uint64(nil), keys...)
	return NewTruncAREInPlace(cp, keyBits, cfg)
}

// NewTruncAREInPlace builds a TruncARE, sorting and deduplicating keys in place.
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewTruncAREInPlace(keys []uint64, keyBits uint32, cfg Config) (*TruncARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)

	n := len(keys)
	if n == 0 {
		return &TruncARE{exact: nil, K: 0, keyBits: keyBits}, nil
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	// deduplicate in-place
	out := keys[:1]
	for _, v := range keys[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	keys = out

	val := (2.0 * float64(len(keys))) / cfg.Eps
	K := uint32(math.Ceil(math.Log2(val)))
	if K == 0 {
		K = 1
	}

	return newTruncAREFromKSorted(keys, keyBits, K)
}

// NewTruncAREFromK builds a TruncARE with an explicit fingerprint width K.
// keys must be sorted in ascending order and deduplicated.
func NewTruncAREFromK(keys []uint64, keyBits uint32, K uint32) (*TruncARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)

	n := len(keys)
	if n == 0 {
		return &TruncARE{exact: nil, K: 0, keyBits: keyBits}, nil
	}
	if K == 0 {
		K = 1
	}

	return newTruncAREFromKSorted(keys, keyBits, K)
}

// newTruncAREFromKSorted is the shared build path. keys must be sorted ascending and deduplicated.
func newTruncAREFromKSorted(keys []uint64, keyBits uint32, K uint32) (*TruncARE, error) {
	n := len(keys)
	minKey := keys[0]
	maxKey := keys[n-1]

	spread := maxKey - minKey
	spreadLen := uint32(mathbits.Len64(spread)) // number of significant bits in spread

	truncatedKeys := make([]uint64, 0, n)
	var lastTrunc uint64
	var hasLast bool
	for _, k := range keys {
		trunc := normalizeToK(k, minKey, spreadLen, K)
		if !hasLast || trunc > lastTrunc {
			truncatedKeys = append(truncatedKeys, trunc)
			lastTrunc = trunc
			hasLast = true
		} else if trunc == lastTrunc {
			continue
		} else {
			return nil, fmt.Errorf("keys must be sorted ascending")
		}
	}

	exact, err := ere_one_d.NewExactRangeEmptiness(truncatedKeys, K)
	if err != nil {
		return nil, err
	}

	return &TruncARE{
		exact:     exact,
		K:         K,
		keyBits:   keyBits,
		minKey:    minKey,
		maxKey:    maxKey,
		spreadLen: spreadLen,
	}, nil
}

// normalizeToK maps key into a K-bit fingerprint in [0, 2^K-1], monotone in key:
//  1. Subtract minKey (offset in [0, spread]).
//  2. Scale so spread occupies K bits:
//     - if K <= spreadLen: right-shift by (spreadLen - K) to take top K bits
//     - if K >  spreadLen: left-shift by (K - spreadLen) to fill K bits
func normalizeToK(key, minKey uint64, spreadLen, K uint32) uint64 {
	if K == 0 || spreadLen == 0 {
		return 0
	}
	offset := key - minKey
	if K <= spreadLen {
		return offset >> (spreadLen - K)
	}
	// K > spreadLen: scale up; result fits because offset < 2^spreadLen
	return offset << (K - spreadLen)
}

// IsEmpty reports whether [lo, hi] contains no stored key.
func (are *TruncARE) IsEmpty(lo, hi uint64) bool {
	if are.exact == nil {
		return true
	}

	if hi < are.minKey {
		return true
	}
	if lo > are.maxKey {
		return true
	}

	var truncLo uint64
	if lo <= are.minKey {
		truncLo = 0
	} else {
		truncLo = normalizeToK(lo, are.minKey, are.spreadLen, are.K)
	}

	var truncHi uint64
	if hi >= are.maxKey {
		truncHi = normalizeToK(are.maxKey, are.minKey, are.spreadLen, are.K)
	} else {
		truncHi = normalizeToK(hi, are.minKey, are.spreadLen, are.K)
	}

	return are.exact.IsEmpty(truncLo, truncHi)
}

func (are *TruncARE) SizeInBits() uint64 {
	if are.exact == nil {
		return 0
	}
	return are.exact.SizeInBits()
}

func (are *TruncARE) ByteSize() int {
	if are == nil || are.exact == nil {
		return 0
	}
	return are.exact.ByteSize() + 8
}

func (are *TruncARE) MemDetailed() utils.MemReport {
	if are == nil || are.exact == nil {
		return utils.MemReport{Name: "TruncARE", TotalBytes: 0}
	}
	return are.exact.MemDetailed()
}
