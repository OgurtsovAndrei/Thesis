package ere_pef

import "sync/atomic"

type QueryStats struct {
	TotalQueries    uint64
	PEF_EarlyExit   uint64
	PEF_MultiChunk  uint64
	PEF_SingleChunk uint64

	KindAllOnes uint64
	KindEF      uint64
	KindBitmap  uint64

	EF_HighSame uint64
	EF_HighDiff uint64

	EF_LinearScan   uint64
	EF_BinarySearch uint64
	
	SumBucketSize uint64
	CountBucketChecks uint64
	
	SumChunkSize uint64
	CountChunkChecks uint64
}

var GlobalStats QueryStats

func ResetGlobalStats() {
	atomic.StoreUint64(&GlobalStats.TotalQueries, 0)
	atomic.StoreUint64(&GlobalStats.PEF_EarlyExit, 0)
	atomic.StoreUint64(&GlobalStats.PEF_MultiChunk, 0)
	atomic.StoreUint64(&GlobalStats.PEF_SingleChunk, 0)
	atomic.StoreUint64(&GlobalStats.KindAllOnes, 0)
	atomic.StoreUint64(&GlobalStats.KindEF, 0)
	atomic.StoreUint64(&GlobalStats.KindBitmap, 0)
	atomic.StoreUint64(&GlobalStats.EF_HighSame, 0)
	atomic.StoreUint64(&GlobalStats.EF_HighDiff, 0)
	atomic.StoreUint64(&GlobalStats.EF_LinearScan, 0)
	atomic.StoreUint64(&GlobalStats.EF_BinarySearch, 0)
	atomic.StoreUint64(&GlobalStats.SumBucketSize, 0)
	atomic.StoreUint64(&GlobalStats.CountBucketChecks, 0)
	atomic.StoreUint64(&GlobalStats.SumChunkSize, 0)
	atomic.StoreUint64(&GlobalStats.CountChunkChecks, 0)
}
