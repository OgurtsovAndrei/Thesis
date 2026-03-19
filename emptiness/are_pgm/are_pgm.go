package are_pgm

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"fmt"
	"math"
	"sort"

	"github.com/agnivade/pgm"
)

// CDFPoint is a control point of the piecewise-linear CDF approximation.
type CDFPoint struct {
	Key  uint64
	Rank float64 // normalized position in [0, 1)
}

// PGMApproximateRangeEmptiness uses a PGM-inspired piecewise-linear CDF
// to map keys to a near-uniform distribution, then stores them in ERE.
//
// The CDF is a monotonic function, which guarantees zero false negatives:
//   x ∈ [a,b] ⟹ CDF(x) ∈ [CDF(a), CDF(b)]
//
// FPR comes only from uint64 quantization at boundaries: a query endpoint
// and a nearby stored key may round to the same mapped position.
// To control this, K must account for both n/ε and the rangeLen.
//
// Limitation: keys must fit in uint64; float64 conversion in PGM
// loses precision for keys > 2^53.
type PGMApproximateRangeEmptiness struct {
	cdf       []CDFPoint
	ere       *ere.ExactRangeEmptiness
	K         uint32
	n         int
	minKey    uint64
	maxKey    uint64
	smoothing float64 // 0 = pure CDF, 1 = pure uniform
}

// NewPGMApproximateRangeEmptinessSmooth builds a CDF-mapped range emptiness filter
// with adjustable smoothing (0 = pure CDF, 1 = pure uniform).
func NewPGMApproximateRangeEmptinessSmooth(keys []uint64, rangeLen uint64, epsilon float64, pgmEpsilon int, smoothing float64) (*PGMApproximateRangeEmptiness, error) {
	return newPGMARE(keys, rangeLen, epsilon, pgmEpsilon, smoothing)
}

// NewPGMApproximateRangeEmptiness builds a CDF-mapped range emptiness filter.
//
// Parameters:
//   - keys: input keys (will be sorted internally)
//   - rangeLen: expected query range length
//   - epsilon: target false positive rate
//   - pgmEpsilon: PGM approximation error bound (controls CDF granularity;
//     smaller = more CDF points = better approximation but more storage)
func NewPGMApproximateRangeEmptiness(keys []uint64, rangeLen uint64, epsilon float64, pgmEpsilon int) (*PGMApproximateRangeEmptiness, error) {
	return newPGMARE(keys, rangeLen, epsilon, pgmEpsilon, 0)
}

func newPGMARE(keys []uint64, rangeLen uint64, epsilon float64, pgmEpsilon int, smoothing float64) (*PGMApproximateRangeEmptiness, error) {
	const maxPGMKeys = 1 << 20
	if len(keys) > maxPGMKeys {
		return nil, fmt.Errorf("CDF-ARE: n=%d exceeds maximum %d (PGM build is O(n²))", len(keys), maxPGMKeys)
	}

	n := len(keys)
	if n == 0 {
		return &PGMApproximateRangeEmptiness{n: 0}, nil
	}

	sorted := make([]uint64, n)
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	// Build PGM index on float64 keys to get CDF approximation
	fkeys := make([]float64, n)
	for i, k := range sorted {
		fkeys[i] = float64(k)
	}
	pgmIdx := pgm.NewIndex(fkeys, pgmEpsilon)

	// Sample PGM at each key and fix monotonicity.
	//
	// BUG(agnivade/pgm): Search panics on large datasets (n ≥ ~65K) due to
	// negative pos from computePos (float64 rounding of key*slope + intercept).
	// In pgm_index.go:76-77:
	//   lo := max(pos-ind.epsilon, 0)       // clamped OK
	//   hi := min(pos+ind.epsilon, len(...)) // NOT clamped to 0 → negative slice bound
	// We recover from the panic and fall back to pos=i (linear CDF).
	// TODO: fork agnivade/pgm and fix: add hi = max(hi, 0) after line 77.
	rawPos := make([]int, n)
	for i, k := range sorted {
		pos := i // fallback: linear CDF
		func() {
			defer func() { recover() }()
			if ap, err := pgmIdx.Search(float64(k)); err == nil {
				pos = ap.Pos
			}
		}()
		rawPos[i] = pos
	}

	// Enforce weak monotonicity via running max
	fixedPos := make([]int, n)
	fixedPos[0] = rawPos[0]
	if fixedPos[0] < 0 {
		fixedPos[0] = 0
	}
	for i := 1; i < n; i++ {
		fixedPos[i] = rawPos[i]
		if fixedPos[i] <= fixedPos[i-1] {
			fixedPos[i] = fixedPos[i-1] + 1
		}
	}
	maxPos := fixedPos[n-1]

	// Build CDF control points by sampling every pgmEpsilon keys
	step := pgmEpsilon
	if step < 1 {
		step = 1
	}
	cdf := make([]CDFPoint, 0, n/step+2)
	cdf = append(cdf, CDFPoint{Key: sorted[0], Rank: float64(fixedPos[0]) / float64(maxPos)})

	for i := step; i < n; i += step {
		cdf = append(cdf, CDFPoint{Key: sorted[i], Rank: float64(fixedPos[i]) / float64(maxPos)})
	}

	// Always include the last key
	if sorted[n-1] != cdf[len(cdf)-1].Key {
		cdf = append(cdf, CDFPoint{Key: sorted[n-1], Rank: float64(fixedPos[n-1]) / float64(maxPos)})
	}

	// Effective range after CDF mapping: use the peak CDF gradient to find
	// worst-case mapped range width for a query of width rangeLen.
	// L_eff = rangeLen × max_segment_density, where density = Δrank/Δkey × n.
	var maxDensity float64
	for i := 1; i < len(cdf); i++ {
		dk := float64(cdf[i].Key - cdf[i-1].Key)
		if dk == 0 {
			continue
		}
		dr := cdf[i].Rank - cdf[i-1].Rank
		density := dr / dk * float64(n)
		if density > maxDensity {
			maxDensity = density
		}
	}
	lEff := float64(rangeLen) * maxDensity
	if lEff < 1 {
		lEff = 1
	}

	K := uint32(math.Ceil(math.Log2(float64(n) * lEff / epsilon)))
	if K < 1 {
		K = 1
	}
	if K > 64 {
		return nil, fmt.Errorf("required K=%d exceeds 64 bits", K)
	}

	filter := &PGMApproximateRangeEmptiness{
		cdf:       cdf,
		K:         K,
		n:         n,
		minKey:    sorted[0],
		maxKey:    sorted[n-1],
		smoothing: smoothing,
	}

	// Map all keys through the same cdfMap used for queries (consistency!)
	mapped := make([]bits.BitString, 0, n)
	var lastVal uint64
	for i, key := range sorted {
		m := filter.cdfMap(key)
		if i == 0 || m != lastVal {
			mapped = append(mapped, bits.NewFromTrieUint64(m, K))
			lastVal = m
		}
	}

	universe := bits.NewBitString(K)
	ereFilter, err := ere.NewExactRangeEmptiness(mapped, universe)
	if err != nil {
		return nil, err
	}
	filter.ere = ereFilter

	return filter, nil
}

func (p *PGMApproximateRangeEmptiness) universeMax() uint64 {
	if p.K == 64 {
		return ^uint64(0)
	}
	return (uint64(1) << p.K) - 1
}

// cdfMap maps a uint64 key to [0, 2^K) using piecewise-linear interpolation.
// Guaranteed monotonic: x1 < x2 ⟹ cdfMap(x1) ≤ cdfMap(x2).
func (p *PGMApproximateRangeEmptiness) cdfMap(x uint64) uint64 {
	segs := p.cdf
	nSegs := len(segs)

	if x <= segs[0].Key {
		return 0
	}
	if x >= segs[nSegs-1].Key {
		return p.universeMax()
	}

	// Binary search: find largest i such that segs[i].Key ≤ x
	lo, hi := 0, nSegs-1
	for lo < hi-1 {
		mid := (lo + hi) / 2
		if segs[mid].Key <= x {
			lo = mid
		} else {
			hi = mid
		}
	}

	// Linear interpolation between segs[lo] and segs[hi]
	keyRange := float64(segs[hi].Key - segs[lo].Key)
	rankRange := segs[hi].Rank - segs[lo].Rank

	var cdfVal float64
	if keyRange == 0 {
		cdfVal = segs[lo].Rank
	} else {
		frac := float64(x-segs[lo].Key) / keyRange
		cdfVal = segs[lo].Rank + frac*rankRange
	}

	// Blend CDF with uniform: mapped = (1-smooth)*CDF + smooth*uniform
	if p.smoothing > 0 && p.maxKey > p.minKey {
		uniformVal := float64(x-p.minKey) / float64(p.maxKey-p.minKey)
		cdfVal = (1-p.smoothing)*cdfVal + p.smoothing*uniformVal
	}

	uMax := p.universeMax()
	m := uint64(cdfVal * float64(uMax))
	if m > uMax {
		m = uMax
	}
	return m
}

func (p *PGMApproximateRangeEmptiness) IsEmpty(a, b uint64) bool {
	if p.n == 0 || a > b {
		return true
	}
	// Queries fully outside the stored key range are definitely empty
	if b < p.minKey || a > p.maxKey {
		return true
	}

	mappedA := p.cdfMap(a)
	mappedB := p.cdfMap(b)

	return p.ere.IsEmpty(
		bits.NewFromTrieUint64(mappedA, p.K),
		bits.NewFromTrieUint64(mappedB, p.K),
	)
}

func (p *PGMApproximateRangeEmptiness) SizeInBits() uint64 {
	if p.ere == nil {
		return 0
	}
	return p.ere.SizeInBits()
}

// CDFSizeInBits returns the space used by the CDF model.
// Each control point: uint64 key (64 bits) + float64 rank (64 bits).
func (p *PGMApproximateRangeEmptiness) CDFSizeInBits() uint64 {
	return uint64(len(p.cdf)) * 128
}

// TotalSizeInBits returns ERE size + CDF model size.
func (p *PGMApproximateRangeEmptiness) TotalSizeInBits() uint64 {
	return p.SizeInBits() + p.CDFSizeInBits()
}
