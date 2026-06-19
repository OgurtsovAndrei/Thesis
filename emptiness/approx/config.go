package approx

import (
	"Thesis/emptiness/exact"
	"fmt"
)

// FallbackPolicy decides whether to use TruncARE or AdaptiveARE (SODA) for fallback keys.
type FallbackPolicy interface {
	UseTrunc(keys []uint64, K uint32, rangeLen uint64) bool
	String() string
}

// Config holds construction parameters for Approximate Range Emptiness (ARE) filters.
// Supports three mutually exclusive operating modes: K, TargetBPK, or Epsilon.
type Config struct {
	// -------------------------------------------------------------
	// 1. Operating Point Selection (exactly one must be set)
	// -------------------------------------------------------------

	// K is the discrete fingerprint width in bits.
	K uint32

	// TargetBPK is the target bits-per-key budget.
	TargetBPK float64

	// Epsilon is the target false positive rate (FPR).
	Epsilon float64

	// -------------------------------------------------------------
	// 2. Query / Optimization Parameters
	// -------------------------------------------------------------

	// RangeLen is the expected query range length.
	// Required if Epsilon is set; also used by SODA and hybrid fallback policies.
	RangeLen uint64

	// -------------------------------------------------------------
	// 3. Algorithm-Specific Tuning
	// -------------------------------------------------------------

	// EREBackend selects the underlying exact range-emptiness implementation.
	// Zero value defaults to exact.VariantAuto.
	EREBackend exact.Variant

	// Seed is the random seed used to draw pairwise-hash coefficients (SODA).
	Seed int64

	// Threshold is the number of low-order bits to truncate (Adaptive).
	Threshold int

	// PGMEpsilon is the PGM approximation error bound (CDF-ARE).
	PGMEpsilon int

	// Smoothing is the CDF-to-uniform blending factor (CDF-ARE).
	Smoothing float64

	// FallbackPolicy decides between Trunc and SODA fallbacks in hybrid structures.
	FallbackPolicy FallbackPolicy
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend set.
func (cfg Config) WithEREBackend(v exact.Variant) Config {
	cfg.EREBackend = v
	return cfg
}

// Backend returns the chosen backend.
func (cfg Config) Backend() exact.Variant {
	return cfg.EREBackend
}

// Validate checks that exactly one operating mode is selected.
func (cfg Config) Validate() error {
	modesSet := 0
	if cfg.K > 0 {
		modesSet++
	}
	if cfg.TargetBPK > 0 {
		modesSet++
	}
	if cfg.Epsilon > 0 {
		modesSet++
	}

	if modesSet == 0 {
		return fmt.Errorf("approx.Config must specify exactly one operating mode: K, TargetBPK, or Epsilon")
	}
	if modesSet > 1 {
		return fmt.Errorf("approx.Config modes are mutually exclusive; specify only one: K, TargetBPK, or Epsilon")
	}
	if cfg.Epsilon > 0 && cfg.RangeLen == 0 {
		return fmt.Errorf("approx.Config requires RangeLen to be specified when Epsilon is used")
	}
	return nil
}
