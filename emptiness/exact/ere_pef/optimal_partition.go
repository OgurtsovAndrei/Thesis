package ere_pef

// PISA-default partitioning parameters: fix_cost=64 bit, eps1=0.03,
// eps2=0.3, eps3=0.01 (Ottaviano-Venturini SIGIR 2014 §4 / PISA's
// partitioned_sequence::compute_partition).
const (
	defaultFixCost = uint64(64)
	defaultEps1    = 0.03
	defaultEps2    = 0.3
	defaultEps3    = 0.01
)

// superblockSize is the maximum number of keys passed to one DP call.
// Larger superblocks pay quadratic-ish memory; PISA caps at fix_cost/eps3.
func superblockSize(fixCost uint64, eps3 float64) int {
	if eps3 <= 0 {
		return 1 << 30
	}
	return int(float64(fixCost) / eps3)
}

// costWindow tracks a sliding sub-range of the keys backing array
// alongside its current cost upper bound. Mirrors PISA's
// optimal_partition::cost_window — Forward iterators replaced with
// `start`/`end` indices into `keys`.
type costWindow struct {
	keys           []uint64
	start, end     uint32
	minP, maxP     uint64
	costUpperBound uint64
}

func (w *costWindow) universe() uint64 { return w.maxP - w.minP + 1 }
func (w *costWindow) size() uint32     { return w.end - w.start }

func (w *costWindow) advanceStart() {
	w.minP = w.keys[w.start] + 1
	w.start++
}

func (w *costWindow) advanceEnd() {
	w.maxP = w.keys[w.end]
	w.end++
}

// partitionScratch holds reusable buffers for repeated DP calls (one
// per superblock). Caller is expected to share one scratch across the
// whole NewPEF build.
type partitionScratch struct {
	minCost []uint64
	path    []uint32
	windows []costWindow
}

func (s *partitionScratch) reset(size uint32) {
	if uint32(cap(s.minCost)) < size+1 {
		s.minCost = make([]uint64, size+1)
		s.path = make([]uint32, size+1)
	} else {
		s.minCost = s.minCost[:size+1]
		s.path = s.path[:size+1]
		for i := range s.path {
			s.path[i] = 0
		}
	}
	s.windows = s.windows[:0]
}

// compute runs the (1+ε)-approximate optimal partition DP on keys[0:],
// returning the partition as a slice of ascending end-indices (so the
// k-th chunk is keys[partition[k-1]:partition[k]] with partition[-1] = 0).
// The total cost in bits is also returned.
//
// `keys` must be sorted ascending; `base` is the universe lower bound
// (typically keys[0] for the first superblock, prev_universe afterward),
// `universe` is the upper bound exclusive (last key + 1, or the input
// universe for the final superblock). `out` is appended to (truncated
// first) so callers can pool the result slice.
func (s *partitionScratch) compute(
	keys []uint64,
	base, universe uint64,
	costFn func(uint64, uint64) uint64,
	eps1, eps2 float64,
	out []uint32,
) ([]uint32, uint64) {
	size := uint32(len(keys))
	if size == 0 {
		return out[:0], 0
	}
	s.reset(size)

	singleBlockCost := costFn(universe-base, uint64(size))
	for i := range s.minCost {
		s.minCost[i] = singleBlockCost
	}
	s.minCost[0] = 0

	costLb := costFn(1, 1)
	costBound := costLb
	for {
		if eps1 > 0 && float64(costBound) >= float64(costLb)/eps1 {
			break
		}
		s.windows = append(s.windows, costWindow{
			keys: keys, minP: base, costUpperBound: costBound,
		})
		if costBound >= singleBlockCost {
			break
		}
		next := uint64(float64(costBound) * (1 + eps2))
		if next == costBound {
			next++
		}
		costBound = next
	}

	for i := uint32(0); i < size; i++ {
		lastEnd := i + 1
		for w := range s.windows {
			win := &s.windows[w]
			for win.end < lastEnd {
				win.advanceEnd()
			}
			for {
				wc := costFn(win.universe(), uint64(win.size()))
				if s.minCost[i]+wc < s.minCost[win.end] {
					s.minCost[win.end] = s.minCost[i] + wc
					s.path[win.end] = i
				}
				lastEnd = win.end
				if win.end == size {
					break
				}
				if wc >= win.costUpperBound {
					break
				}
				win.advanceEnd()
			}
			win.advanceStart()
		}
	}

	out = out[:0]
	for curr := size; curr != 0; curr = s.path[curr] {
		out = append(out, curr)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, s.minCost[size]
}
