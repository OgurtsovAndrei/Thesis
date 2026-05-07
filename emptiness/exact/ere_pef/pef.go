package ere_pef

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"Thesis/succinct_bit_vector/rsdic"
)

// PEF is a Partitioned Elias-Fano range emptiness structure.
// Built once via NewPEF; queries are read-only thereafter.
//
// All per-chunk rank/select state lives in a single shared RSDic
// (`rs`) instead of one RSDic per chunk: an RSDic value carries
// ~200 B of struct + slice-header overhead and ~6 small backing
// allocations regardless of payload, which dominates BPK at the
// chunk counts produced on 10⁸+-key datasets.
type PEF struct {
	n        int
	keyBits  uint32
	firstKey uint64
	lastKey  uint64
	chunks   []chunk
	efMeta   []efChunkMeta // one entry per kindEF chunk
	bmMeta   []bmChunkMeta // one entry per kindBitmap chunk
	rs       rsdic.RSDic   // shared rank/select for ALL EF + bitmap chunks
	lowBits  []uint64
	lowBitsN uint64
}

// chunkKind selects the codec used to encode a chunk's keys.
type chunkKind uint8

const (
	kindAllOnes chunkKind = iota
	kindEF
	kindBitmap
)

// chunkKindMask masks out the codec discriminator from chunk.nKind.
const chunkKindMask uint32 = 0x3

// chunk describes one partition piece. base is derived cumulatively
// from the previous chunk's last (or PEF.firstKey for chunk 0).
//
//	nKind   = (n << 2) | kind     (kind ∈ {0,1,2}, n ≤ 2³⁰-1)
//	metaIdx = index into PEF.efMeta (kindEF) or PEF.bmMeta (kindBitmap);
//	          unused for kindAllOnes.
//
// Sized to keep per-chunk overhead at 16 bytes.
type chunk struct {
	last    uint64
	nKind   uint32
	metaIdx uint32
}

func packNKind(n uint32, k chunkKind) uint32 {
	return (n << 2) | uint32(k)
}

func (c *chunk) kind() chunkKind { return chunkKind(c.nKind & chunkKindMask) }
func (c *chunk) n() uint32       { return c.nKind >> 2 }

// efChunkMeta is the per-EF-chunk state required for Select1 over the
// shared RSDic plus the chunk's slice of PEF.lowBits.
//
//	lowOff      — start bit offset in PEF.lowBits for this chunk's lows.
//	globalOff   — start bit offset in PEF.rs for this chunk's upper bits.
//	onesBefore  — number of 1s in PEF.rs strictly before globalOff.
//
// At query time, the local Select1(k) of an EF chunk equals
// `rs.Select1(onesBefore+k) - globalOff`.
type efChunkMeta struct {
	lowOff     uint64
	globalOff  uint64
	onesBefore uint64
}

// bmChunkMeta is the per-bitmap-chunk state — only the global offset.
// Rank differences within a chunk cancel out the count of 1s before
// the chunk, so onesBefore is not needed here.
type bmChunkMeta struct {
	globalOff uint64
}

// NewPEF builds a Partitioned Elias-Fano range emptiness structure
// from the given sorted key slice using the PISA-paper default
// hyperparameters (DefaultPartitionConfig). Duplicate keys are silently
// deduplicated (set semantics is preserved for IsEmpty). keyBits is
// the effective key width used by callers; queries with values
// outside [0, 2^keyBits) are still defined (IsEmpty returns true for
// queries that do not overlap any stored key).
func NewPEF(keys []uint64, keyBits uint32) (*PEF, error) {
	return NewPEFWithConfig(keys, keyBits, DefaultPartitionConfig())
}

// NewPEFWithConfig is NewPEF with explicit partitioning hyperparameters.
// See PartitionConfig for the tuning levers.
func NewPEFWithConfig(keys []uint64, keyBits uint32, cfg PartitionConfig) (*PEF, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if keyBits > 64 {
		return nil, fmt.Errorf("keyBits must be <= 64, got %d", keyBits)
	}
	p := &PEF{keyBits: keyBits}
	if len(keys) == 0 {
		return p, nil
	}

	deduped := make([]uint64, 0, len(keys))
	for i, k := range keys {
		if i > 0 && k < keys[i-1] {
			return nil, fmt.Errorf("keys must be sorted (idx=%d)", i)
		}
		if i > 0 && k == keys[i-1] {
			continue
		}
		deduped = append(deduped, k)
	}

	n := len(deduped)
	p.n = n
	p.firstKey = deduped[0]
	p.lastKey = deduped[n-1]

	if n == 1 {
		c := chunk{last: deduped[0], nKind: packNKind(1, kindAllOnes)}
		p.writeChunk(&c, deduped[0], deduped)
		p.chunks = append(p.chunks, c)
		return p, nil
	}

	costFn := func(lastRel, n uint64) uint64 {
		return minCodecBitsize(lastRel, n) + cfg.FixCost
	}
	superSize := superblockSize(cfg.FixCost, cfg.Eps3)

	// Phase 0 — enumerate superblock jobs (sequential, cheap).
	// superJob.lastKey is the inclusive upper bound of the superblock
	// (the last key in that superblock's slice). Using an inclusive bound
	// avoids lastKey+1 overflow when the final key is ^uint64(0).
	type superJob struct {
		keys    []uint64
		base    uint64
		lastKey uint64
	}
	var jobs []superJob
	{
		superPos := 0
		superBase := deduped[0]
		for superPos < n {
			sz := superSize
			if sz > n-superPos {
				sz = n - superPos
			}
			// Merge a tiny tail into current superblock so no superblock is
			// shorter than `superSize` (mirrors PISA compute_partition).
			if rem := n - (superPos + sz); rem > 0 && rem < superSize {
				sz = n - superPos
			}
			superKeys := deduped[superPos : superPos+sz]
			superLastKey := deduped[superPos+sz-1]
			jobs = append(jobs, superJob{keys: superKeys, base: superBase, lastKey: superLastKey})
			superPos += sz
			// Next superblock starts just after this one's last key.
			// If superLastKey == ^uint64(0) there is no next superblock
			// (superPos == n), so the +1 wrap is never observed.
			superBase = superLastKey + 1
		}
	}

	// Phase 1 — parallel DP per superblock. Each worker has its own
	// partitionScratch + reusable `buf`. Results live in a fixed-index
	// slice; no channel needed for ordering.
	partitions := make([][]uint32, len(jobs))
	numWorkers := cfg.NumWorkers
	if numWorkers <= 0 {
		numWorkers = defaultNumWorkers
	}
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}
	var idx int64
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scratch := &partitionScratch{}
			var buf []uint32
			for {
				i := atomic.AddInt64(&idx, 1) - 1
				if i >= int64(len(jobs)) {
					return
				}
				job := &jobs[i]
				result, _ := scratch.compute(
					job.keys, job.base, job.lastKey,
					costFn, cfg.Eps1, cfg.Eps2, buf,
				)
				// Copy out — `result` aliases `buf`, which is reused on
				// the next iteration in this worker.
				out := make([]uint32, len(result))
				copy(out, result)
				partitions[i] = out
				buf = result[:0]
			}
		}()
	}
	wg.Wait()

	// Phase 2 — sequential chunk emission. Must be sequential because
	// writeChunk captures p.rs.Num() / p.rs.OneNum() / p.lowBitsN at the
	// start of each chunk; concurrent writes would interleave offsets.
	chunkBase := deduped[0]
	for i := range jobs {
		job := &jobs[i]
		prevEnd := uint32(0)
		for _, end := range partitions[i] {
			ks := job.keys[prevEnd:end]
			c := chunk{
				last:  ks[len(ks)-1],
				nKind: packNKind(uint32(len(ks)), 0), // kind set inside writeChunk
			}
			p.writeChunk(&c, chunkBase, ks)
			p.chunks = append(p.chunks, c)
			// chunkBase for the next chunk: c.last+1 may wrap to 0 when
			// c.last==^uint64(0), but in that case this is the final chunk
			// (no key can exceed ^uint64(0)), so the value is never read.
			chunkBase = c.last + 1
			prevEnd = end
		}
	}

	return p, nil
}

// chunkBaseAt returns the absolute lower bound of chunks[i] (i.e. the
// smallest key the chunk could contain). It is derived cumulatively
// from chunks[i-1].last to keep chunk struct at 16 bytes.
func (p *PEF) chunkBaseAt(i int) uint64 {
	if i == 0 {
		return p.firstKey
	}
	return p.chunks[i-1].last + 1
}

// IsEmpty reports whether the closed range [a, b] contains no stored key.
func (p *PEF) IsEmpty(a, b uint64) bool {
	if p.n == 0 || a > b {
		return true
	}
	if b < p.firstKey || a > p.lastKey {
		return true
	}
	if a < p.firstKey {
		a = p.firstKey
	}
	if b > p.lastKey {
		b = p.lastKey
	}

	chunks := p.chunks
	i := sort.Search(len(chunks), func(k int) bool { return chunks[k].last >= a })
	j := sort.Search(len(chunks), func(k int) bool { return chunks[k].last >= b })

	if i != j {
		return false
	}
	return !p.chunkIntersects(i, a, b)
}

// ByteSize returns total in-memory footprint in bytes (struct +
// chunks + per-chunk metadata + shared rsdic backing + low bits buffer).
func (p *PEF) ByteSize() int {
	if p == nil || p.n == 0 {
		return 0
	}
	size := int(unsafe.Sizeof(*p))
	size += len(p.chunks) * int(unsafe.Sizeof(chunk{}))
	size += len(p.efMeta) * int(unsafe.Sizeof(efChunkMeta{}))
	size += len(p.bmMeta) * int(unsafe.Sizeof(bmChunkMeta{}))
	size += p.rs.AllocSize()
	size += len(p.lowBits) * 8
	return size
}

// SizeInBits returns the logical (information-theoretic) bit count of
// the encoded payload: shared rsdic Num() plus the EF low-bits stream.
func (p *PEF) SizeInBits() uint64 {
	if p == nil || p.n == 0 {
		return 0
	}
	return p.rs.Num() + p.lowBitsN
}

// NumChunks returns the number of chunks the input was partitioned
// into. Useful as a sanity-check / characterization metric in
// benchmarks (more chunks ⇒ DP found a less-uniform-density input).
func (p *PEF) NumChunks() int {
	if p == nil {
		return 0
	}
	return len(p.chunks)
}

// MetadataAllocBits returns the rank/select dictionary backing
// allocation in bits — the structural metadata that supports O(1) rank
// and select. EF low bits are excluded (those are payload).
func (p *PEF) MetadataAllocBits() uint64 {
	if p == nil || p.n == 0 {
		return 0
	}
	b := uint64(p.rs.AllocSize()) * 8
	b += uint64(len(p.efMeta)) * uint64(unsafe.Sizeof(efChunkMeta{})) * 8
	b += uint64(len(p.bmMeta)) * uint64(unsafe.Sizeof(bmChunkMeta{})) * 8
	return b
}
