package bucket

import (
	"Thesis/bits"
	"Thesis/errutil"
	"Thesis/mmph/go-boomphf"
	"Thesis/zfasttrie"
	"fmt"
	"math"
	"math/rand"
)

// Bucket represents a single bucket in the hash structure
type Bucket struct {
	mphf      *boomphf.H     // MPHF for this bucket
	ranks     []uint8        // Local ranks within this bucket
	delimiter bits.BitString // The delimiter (last key) of this bucket
}

// MonotoneHashWithTrie implements monotone minimal perfect hashing using
// approximate z-fast trie for relative ranking as described in Section 5 of the MMPH paper.
// This approach uses a relative trie on bucket delimiters for more efficient ranking.
// E, S, I are the type parameters for the underlying ApproxZFastTrie.
type MonotoneHashWithTrie[E zfasttrie.UNumber, S zfasttrie.UNumber, I zfasttrie.UNumber] struct {
	bucketSize int

	// Approximate Z-Fast Trie for relative ranking of bucket delimiters
	delimiterTrie *zfasttrie.ApproxZFastTrie[E, S, I]

	// Efficient bucket storage with signatures instead of full keys
	buckets []*Bucket

	// Statistics
	TrieRebuildAttempts int // Number of trie rebuild attempts during construction
}

const maxTrieRebuilds = 100 // Maximum number of attempts to build a working trie

// NewMonotoneHashWithTrie creates a new monotone hash function using approximate z-fast trie
// for bucket identification. The buckets are divided based on TrieCompare ordering,
// with delimiters being the last key in each bucket.
//
// IMPORTANT: Input data must be sorted in TrieCompare order for mixed-size string support.
// Use sort.Sort(TrieCompareSorter(data)) to ensure correct ordering.
//
// It validates that all keys work correctly with the trie and rebuilds with new seeds if needed.
// S - used in PSig should be at least ((log log n) + (log log w) - (log eps)) bits
// E - used for Max String Len
// I - used for indexing in Delimiters Trie, should be at least log(N / 256)
func NewMonotoneHashWithTrie[E zfasttrie.UNumber, S zfasttrie.UNumber, I zfasttrie.UNumber](data []bits.BitString) (*MonotoneHashWithTrie[E, S, I], error) {
	if len(data) == 0 {
		return &MonotoneHashWithTrie[E, S, I]{}, nil
	}

	// Validate that input data is sorted according to TrieCompare ordering
	// This is required for the MMPH algorithm to work correctly with mixed-size strings
	for i := 1; i < len(data); i++ {
		if data[i-1].TrieCompare(data[i]) >= 0 {
			return nil, fmt.Errorf("input data must be sorted in TrieCompare order: data[%d] (%s) > data[%d] (%s)",
				i-1, data[i-1].PrettyString(), i, data[i].PrettyString())
		}
	}

	// Choose bucket size as log n (as suggested in the paper)
	minBucketSize := max(int(math.Ceil(math.Log2(float64(len(data))))), 1)
	if minBucketSize > 256 {
		errutil.Bug("bucketSize must be <= 256 when using uint8 optimization")
	}
	bucketSize := 256 // max value of uint8

	n := len(data)
	numBuckets := (n + bucketSize - 1) / bucketSize

	mh := &MonotoneHashWithTrie[E, S, I]{
		bucketSize: bucketSize,
		buckets:    make([]*Bucket, numBuckets),
	}

	// Build buckets efficiently with only delimiter storage
	var delimiters []bits.BitString
	for i := 0; i < numBuckets; i++ {
		start := i * bucketSize
		end := start + bucketSize
		if end > n {
			end = n
		}

		bucketKeys := data[start:end]
		if len(bucketKeys) > 0 {
			// Build MPHF for this bucket
			bucketHashes := make([]uint64, len(bucketKeys))
			for j, k := range bucketKeys {
				bucketHashes[j] = k.Hash()
			}

			bucketMPHF := boomphf.New(2.0, bucketHashes)

			// Verify MPHF construction
			for j, h := range bucketHashes {
				if idx := bucketMPHF.Query(h); idx == 0 {
					errutil.Bug("boomphf failed immediately on construction for bucket %d key %d", i, j)
				}
			}

			// Set up local ranks within the bucket
			bucketRanks := make([]uint8, len(bucketKeys))
			for localRank, k := range bucketKeys {
				phfIdx := bucketMPHF.Query(k.Hash()) - 1
				bucketRanks[phfIdx] = uint8(localRank)
			}

			// Last key of bucket becomes delimiter
			delimiter := bucketKeys[len(bucketKeys)-1]
			delimiters = append(delimiters, delimiter)

			// Create efficient bucket
			mh.buckets[i] = &Bucket{
				mphf:      bucketMPHF,
				ranks:     bucketRanks,
				delimiter: delimiter,
			}
		}
	}

	// Build approximate z-fast trie with validation and retry logic
	if len(delimiters) > 0 {
		err := mh.buildValidatedTrieWithIndices(data, delimiters)
		if err != nil {
			return nil, err
		}
	}

	return mh, nil
}

// buildValidatedTrieWithIndices builds the approximate z-fast trie with bucket indices and validates it works for all keys.
// It retries with different seeds if validation fails.
func (mh *MonotoneHashWithTrie[E, S, I]) buildValidatedTrieWithIndices(allKeys []bits.BitString, delimiters []bits.BitString) error {
	for attempt := 0; attempt < maxTrieRebuilds; attempt++ {
		mh.TrieRebuildAttempts = attempt + 1

		// Build trie with current attempt using delimiter indices
		var err error
		mh.delimiterTrie, err = zfasttrie.NewApproxZFastTrie[E, S, I](delimiters, false)
		if err != nil {
			return err
		}

		// Validate that all keys work correctly with this trie
		if mh.validateAllKeys(allKeys) {
			return nil // Success!
		}

		// Validation failed - perturb global rand state for next attempt
		// NewApproxZFastTrie uses global rand.Uint64() for seeds, so we need to
		// advance the global random state to get different hash functions
		// Consume some random values to change the state
		for i := 0; i < 10; i++ {
			rand.Uint64()
		}
	}

	return fmt.Errorf("failed to build working approximate z-fast trie after %d attempts, try to increase S and/or I", maxTrieRebuilds)
}

// validateAllKeys checks that the trie can correctly handle all input keys
// Uses two-pointer approach since both keys and buckets are sorted
func (mh *MonotoneHashWithTrie[E, S, I]) validateAllKeys(allKeys []bits.BitString) bool {
	if len(allKeys) == 0 || len(mh.buckets) == 0 {
		return true
	}

	bucketIdx := 0
	maxDelimiterIndex := I(^I(0))

	for _, key := range allKeys {
		// Find the correct bucket using two-pointer approach
		// Advance bucketIdx until we find a bucket where key <= bucket.delimiter
		// Use TrieCompare for consistent ordering with mixed-size strings
		for bucketIdx < len(mh.buckets) {
			bucket := mh.buckets[bucketIdx]
			if bucket != nil && key.TrieCompare(bucket.delimiter) <= 0 {
				// Found the correct bucket for this key
				break
			}
			bucketIdx++
		}

		if bucketIdx >= len(mh.buckets) {
			errutil.Bug("bucketIdx out of range")
		}

		expectedBucket := bucketIdx

		// Test if trie can find this bucket using LowerBound
		cand1, cand2, cand3 := mh.delimiterTrie.LowerBound(key)

		// Check if any of the candidates can lead us to the correct bucket
		foundCorrectBucket := false

		candidates := []*zfasttrie.NodeData[E, S, I]{cand1, cand2, cand3}
		for _, candidate := range candidates {
			if candidate == nil {
				continue
			}

			// Check if this candidate has a valid delimiter index pointing to expected bucket
			if candidate.Rank != maxDelimiterIndex {
				candidateBucketIdx := int(candidate.Rank)
				if candidateBucketIdx == expectedBucket {
					foundCorrectBucket = true
					break
				}
			}
		}

		// If trie failed to provide any candidate that leads to correct bucket,
		// this is a false negative
		if !foundCorrectBucket {
			return false
		}
	}

	return true
}

// GetRank returns the rank of the given key in the original sorted order.
// It uses the approximate z-fast trie to find the correct bucket, then
// computes the local rank within that bucket.
func (mh *MonotoneHashWithTrie[E, S, I]) GetRank(key bits.BitString) int {
	if mh.delimiterTrie == nil || len(mh.buckets) == 0 {
		return -1
	}

	// Use the approximate z-fast trie to get candidates for the bucket
	// This implements the relative ranking approach from Section 4.2
	cand1, cand2, cand3 := mh.delimiterTrie.LowerBound(key)

	// Try candidates to find the correct bucket using O(1) delimiterIndex lookup
	bucketIdx := -1
	maxDelimiterIndex := I(^I(0)) // Max value for I

	// Helper function to try a candidate with O(1) lookup
	tryCandidate := func(candidate *zfasttrie.NodeData[E, S, I]) bool {
		if candidate == nil {
			return false
		}

		// Check if this candidate has a valid delimiter index
		if candidate.Rank != maxDelimiterIndex {
			candidateBucketIdx := int(candidate.Rank)

			// Verify key belongs to this bucket by checking range
			if candidateBucketIdx < len(mh.buckets) && mh.buckets[candidateBucketIdx] != nil {
				bucket := mh.buckets[candidateBucketIdx]

				// Check if key is <= delimiter and > previous bucket's delimiter
				// Use TrieCompare for consistent ordering with mixed-size strings
				if key.TrieCompare(bucket.delimiter) <= 0 {
					if candidateBucketIdx == 0 ||
						(candidateBucketIdx > 0 && mh.buckets[candidateBucketIdx-1] != nil &&
							key.TrieCompare(mh.buckets[candidateBucketIdx-1].delimiter) > 0) {
						bucketIdx = candidateBucketIdx
						return true
					}
				}
			}
		}
		return false
	}

	if !tryCandidate(cand1) {
		if !tryCandidate(cand2) {
			tryCandidate(cand3)
		}
	}

	if bucketIdx == -1 || bucketIdx >= len(mh.buckets) || mh.buckets[bucketIdx] == nil {
		return -1
	}

	// Get local rank within the bucket
	bucket := mh.buckets[bucketIdx]
	keyHash := key.Hash()
	localPhfIdx := bucket.mphf.Query(keyHash)
	if localPhfIdx == 0 || int(localPhfIdx) > len(bucket.ranks) {
		return -1 // Key not found in expected bucket
	}

	localOffset := int(bucket.ranks[localPhfIdx-1])

	return bucketIdx*mh.bucketSize + localOffset
}

// Size returns the total size of the structure in bytes.
func (mh *MonotoneHashWithTrie[E, S, I]) Size() int {
	size := 4 // bucket size

	// Size of approximate z-fast trie (exact calculation)
	if mh.delimiterTrie != nil {
		size += mh.delimiterTrie.ByteSize()
	}

	// Size of efficient buckets
	for _, bucket := range mh.buckets {
		if bucket != nil {
			// MPHF size
			size += bucket.mphf.Size()
			// Ranks array size
			size += len(bucket.ranks)
			// Delimiter size
			size += int(bucket.delimiter.Size())/8 + 1 // Convert bits to bytes
		}
	}

	// Statistics overhead (int for TrieRebuildAttempts)
	size += 4

	return size
}

// ByteSize returns the total size of the structure in bytes (same as Size for consistency).
func (mh *MonotoneHashWithTrie[E, S, I]) ByteSize() int {
	return mh.Size()
}
