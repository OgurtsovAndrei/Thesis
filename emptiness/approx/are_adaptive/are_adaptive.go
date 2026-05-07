package are_adaptive

import (
	"Thesis/emptiness/exact"
	internalhash "Thesis/emptiness/internal/hash"
	"Thesis/utils/errutil"
	mbits "math/bits"
	"math/rand"
	"sort"
)

// Config holds the construction parameters for AdaptiveARE. K is the fingerprint
// width in bits (the payload size per key, excluding metadata). Threshold is the
// number of low-order bits to truncate from each key before hashing.
//
// EREBackend selects the underlying exact range-emptiness implementation
// (see package exact). Use WithEREBackend to set it explicitly; the
// default is exact.VariantAuto.
type Config struct {
	K          uint32
	Threshold  int
	EREBackend exact.Variant
	backendSet bool
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend.
func (cfg Config) WithEREBackend(v exact.Variant) Config {
	cfg.EREBackend = v
	cfg.backendSet = true
	return cfg
}

func (cfg Config) backend() exact.Variant {
	if cfg.backendSet {
		return cfg.EREBackend
	}
	return exact.VariantAuto
}

// AdaptiveARE is an approximate range emptiness filter that adaptively chooses
// between exact mode (when the key spread fits in K bits) and SODA hashing mode.
type AdaptiveARE struct {
	ere          exact.Filter
	K            uint32
	minKey       uint64
	keyBits      uint32
	truncateBits uint32
	IsExactMode  bool
	n            int
	hashA        uint64
	hashB        uint64
}

// hashBlockIndex hashes a block index (uint64) to a K-bit uint64.
func hashBlockIndex(block uint64, a, b uint64, K uint32) uint64 {
	return internalhash.PairwiseHash(block, a, b, K)
}

// ExactModeViable reports whether exact mode would trigger for a segment
// with the given spread, without building the filter.
// spread is max(S) - min(S) in the original key space.
func ExactModeViable(spread uint64, K uint32) bool {
	if K == 0 || K > 64 {
		return false
	}
	var M uint32
	if spread > 0 {
		M = uint32(mbits.Len64(spread))
	}
	return M <= K
}

// NewAdaptiveARE builds an AdaptiveARE from a copy of keys.
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewAdaptiveARE(keys []uint64, keyBits uint32, cfg Config) (*AdaptiveARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return NewAdaptiveAREInPlace(cp, keyBits, cfg)
}

// NewAdaptiveAREInPlace builds an AdaptiveARE, sorting keys in place.
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewAdaptiveAREInPlace(keys []uint64, keyBits uint32, cfg Config) (*AdaptiveARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)

	n := len(keys)
	if n == 0 {
		return &AdaptiveARE{n: 0, keyBits: keyBits}, nil
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return newAdaptiveAREFromKSorted(keys, keyBits, cfg.K, uint32(cfg.Threshold), cfg.backend())
}

// NewAdaptiveAREFromK builds an AdaptiveARE with an explicit fingerprint width K
// using the default exact backend (VariantAuto).
// keys need not be sorted; they are sorted in place. Used by other ARE packages
// that already have K computed and want to skip Config wrapping.
func NewAdaptiveAREFromK(keys []uint64, keyBits uint32, K uint32, threshold int) (*AdaptiveARE, error) {
	return NewAdaptiveAREFromKWithBackend(keys, keyBits, K, threshold, exact.VariantAuto)
}

// NewAdaptiveAREFromKWithBackend is NewAdaptiveAREFromK with an explicit ERE backend.
func NewAdaptiveAREFromKWithBackend(keys []uint64, keyBits uint32, K uint32, threshold int, backend exact.Variant) (*AdaptiveARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)

	n := len(keys)
	if n == 0 {
		return &AdaptiveARE{n: 0, keyBits: keyBits}, nil
	}
	errutil.BugOn(K == 0 || K > 64, "K must be in (0, 64], got %d", K)

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return newAdaptiveAREFromKSorted(keys, keyBits, K, uint32(threshold), backend)
}

// newAdaptiveAREFromKSorted is the shared build path.
// keys must be sorted ascending.
func newAdaptiveAREFromKSorted(keys []uint64, keyBits uint32, K uint32, t uint32, backend exact.Variant) (*AdaptiveARE, error) {
	n := len(keys)

	minKey := keys[0]
	maxKey := keys[n-1]

	// spread after subtraction and truncation
	spreadVal := (maxKey - minKey) >> t

	var M uint32
	if spreadVal > 0 {
		M = uint32(mbits.Len64(spreadVal))
	}

	isExactMode := (M <= K)
	finalUniverseBits := K
	if isExactMode {
		finalUniverseBits = M
	}

	rng := rand.New(rand.NewSource(int64(n) ^ int64(K)))
	hashA := rng.Uint64() | 1
	hashB := rng.Uint64()

	hashedKeys := make([]uint64, n)

	rMask := (uint64(1) << K) - 1
	if K == 64 {
		rMask = ^uint64(0)
	}

	for i, x := range keys {
		xPrime := (x - minKey) >> t

		if isExactMode {
			hashedKeys[i] = xPrime
		} else {
			var block uint64
			var offsetVal uint64

			if xPrime>>K > 0 {
				// xPrime has more than K bits: top bits form block, low K bits are offset
				block = xPrime >> K
				offsetVal = xPrime & rMask
			} else {
				block = 0
				offsetVal = xPrime
			}

			u := hashBlockIndex(block, hashA, hashB, K)
			hashedKeys[i] = (u + offsetVal) & rMask
		}
	}

	uniqueHashed := internalhash.SortAndDedupUint64Adaptive(hashedKeys, finalUniverseBits)

	ereFilter, err := exact.NewUint64WithVariant(uniqueHashed, finalUniverseBits, backend)
	if err != nil {
		return nil, err
	}

	return &AdaptiveARE{
		ere:          ereFilter,
		K:            finalUniverseBits,
		minKey:       minKey,
		keyBits:      keyBits,
		truncateBits: t,
		IsExactMode:  isExactMode,
		n:            n,
		hashA:        hashA,
		hashB:        hashB,
	}, nil
}

// IsEmpty reports whether [lo, hi] contains no stored key.
func (are *AdaptiveARE) IsEmpty(lo, hi uint64) bool {
	if are.n == 0 || lo > hi {
		return true
	}

	t := are.truncateBits

	// Normalize lo: clamp values below minKey to 0 in shifted space
	var loPrime uint64
	if lo < are.minKey {
		loPrime = 0
	} else {
		loPrime = (lo - are.minKey) >> t
	}

	// If hi < minKey, no keys can be in [lo, hi]
	if hi < are.minKey {
		return true
	}
	hiPrime := (hi - are.minKey) >> t

	if are.IsExactMode {
		return are.ere.IsEmpty(loPrime, hiPrime)
	}
	return are.sodaIsEmpty(loPrime, hiPrime)
}

func (are *AdaptiveARE) sodaIsEmpty(a, b uint64) bool {
	K := are.K
	rMask := (uint64(1) << K) - 1
	if K == 64 {
		rMask = ^uint64(0)
	}

	var blockA, blockB uint64
	var offA, offB uint64

	if a>>K > 0 {
		blockA = a >> K
		offA = a & rMask
	} else {
		blockA = 0
		offA = a
	}

	if b>>K > 0 {
		blockB = b >> K
		offB = b & rMask
	} else {
		blockB = 0
		offB = b
	}

	if blockA == blockB {
		u := hashBlockIndex(blockA, are.hashA, are.hashB, K)
		hA := (u + offA) & rMask
		hB := (u + offB) & rMask

		if hA <= hB {
			return are.ere.IsEmpty(hA, hB)
		}
		if !are.ere.IsEmpty(hA, rMask) {
			return false
		}
		return are.ere.IsEmpty(0, hB)
	}

	// Multi-block: check suffix of first block
	uA := hashBlockIndex(blockA, are.hashA, are.hashB, K)
	hAStart := (uA + offA) & rMask
	hAEnd := (uA + rMask) & rMask
	if hAStart <= hAEnd {
		if !are.ere.IsEmpty(hAStart, hAEnd) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(hAStart, rMask) ||
			!are.ere.IsEmpty(0, hAEnd) {
			return false
		}
	}

	// Intermediate full blocks
	if !are.ere.IsEmpty(0, rMask) {
		return false
	}

	// Prefix of last block
	uB := hashBlockIndex(blockB, are.hashA, are.hashB, K)
	hBStart := (uB + 0) & rMask
	hBEnd := (uB + offB) & rMask
	if hBStart <= hBEnd {
		if !are.ere.IsEmpty(hBStart, hBEnd) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(hBStart, rMask) ||
			!are.ere.IsEmpty(0, hBEnd) {
			return false
		}
	}

	return true
}

func (are *AdaptiveARE) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
