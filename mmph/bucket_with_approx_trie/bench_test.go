package bucket

import (
	"Thesis/bits"
	"fmt"
	"sort"
	"sync"
	"testing"
)

var (
	benchKeyCounts = []int{1 << 5, 1 << 8, 1 << 10, 1 << 13, 1 << 15, 1 << 18, 1 << 20, 1 << 23}
	benchKeys      map[int][]bits.BitString
	benchOnce      sync.Once
)

func initBenchKeys() {
	benchOnce.Do(func() {
		benchKeys = make(map[int][]bits.BitString)
		for _, count := range benchKeyCounts {
			rawKeys := buildUniqueStrKeys(count)

			bsKeys := make([]bits.BitString, count)
			for i, k := range rawKeys {
				bsKeys[i] = bits.NewFromText(k)
			}

			sort.Sort(bitStringSorter(bsKeys))

			benchKeys[count] = bsKeys
		}
	})
}

func BenchmarkMonotoneHashWithTrieBuild(b *testing.B) {
	initBenchKeys()

	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]

			var totalTrieAttempts int64
			var maxTrieAttempts int

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				table, err := NewMonotoneHashWithTrie[uint8, uint32, uint16](keys)
				if err != nil {
					b.Fatalf("Failed to build: %v", err)
				}

				totalTrieAttempts += int64(table.TrieRebuildAttempts)
				if table.TrieRebuildAttempts > maxTrieAttempts {
					maxTrieAttempts = table.TrieRebuildAttempts
				}

				b.ReportMetric(float64(table.Size())*8/float64(count), "bits/key_in_mem")
				b.ReportMetric(float64(table.Size()), "bytes_in_mem")
			}

			b.ReportMetric(float64(totalTrieAttempts)/float64(b.N), "avg_trie_attempts")
			b.ReportMetric(float64(maxTrieAttempts), "max_trie_attempts")
		})
	}
}

func BenchmarkMonotoneHashWithTrieLookup(b *testing.B) {
	initBenchKeys()

	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]
			mh, err := NewMonotoneHashWithTrie[uint8, uint32, uint16](keys)
			if err != nil {
				b.Fatalf("Failed to build: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Query keys cyclically
				_ = mh.GetRank(keys[i%count])
			}
		})
	}
}

func BenchmarkMonotoneHashWithTrieLookupMiss(b *testing.B) {
	initBenchKeys()

	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]
			mh, err := NewMonotoneHashWithTrie[uint8, uint32, uint16](keys)
			if err != nil {
				b.Fatalf("Failed to build: %v", err)
			}

			// Generate some keys that are likely not in the set
			missKeys := buildUniqueStrKeys(100)
			missBitKeys := make([]bits.BitString, len(missKeys))
			for i, s := range missKeys {
				missBitKeys[i] = bits.NewFromText(s)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Query non-existent keys cyclically
				_ = mh.GetRank(missBitKeys[i%len(missBitKeys)])
			}
		})
	}
}

// Benchmark to measure trie construction overhead specifically
func BenchmarkTrieRebuilds(b *testing.B) {
	initBenchKeys()

	// Focus on medium-sized datasets where trie rebuilds are more likely
	testCounts := []int{1 << 10, 1 << 13, 1 << 15}

	for _, count := range testCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]

			successfulBuilds := 0
			totalAttempts := int64(0)
			failedBuilds := 0

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				table, err := NewMonotoneHashWithTrie[uint8, uint32, uint16](keys)
				if err != nil {
					failedBuilds++
					continue
				}

				successfulBuilds++
				totalAttempts += int64(table.TrieRebuildAttempts)
			}

			if successfulBuilds > 0 {
				b.ReportMetric(float64(totalAttempts)/float64(successfulBuilds), "avg_attempts_per_success")
			}
			b.ReportMetric(float64(failedBuilds), "failed_builds")
			b.ReportMetric(float64(successfulBuilds), "successful_builds")
		})
	}
}

// Memory usage benchmark
func BenchmarkMonotoneHashWithTrieMemory(b *testing.B) {
	initBenchKeys()

	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				table, err := NewMonotoneHashWithTrie[uint8, uint32, uint16](keys)
				if err != nil {
					b.Fatalf("Failed to build: %v", err)
				}

				size := table.Size()

				// Report various memory metrics
				b.ReportMetric(float64(size), "total_bytes")
				b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
				b.ReportMetric(float64(len(table.buckets)), "num_buckets")
				b.ReportMetric(float64(table.bucketSize), "bucket_size")

				// Calculate exact trie size and overhead
				trieSize := 0
				if table.delimiterTrie != nil {
					trieSize = table.delimiterTrie.ByteSize()
				}

				bucketOverhead := 0
				delimiterOverhead := 0
				for _, bucket := range table.buckets {
					if bucket != nil {
						// MPHF + ranks array + delimiter
						bucketOverhead += bucket.mphf.Size()
						bucketOverhead += len(bucket.ranks)
						delimiterOverhead += int(bucket.delimiter.Size())/8 + 1
					}
				}

				// Report exact measurements
				b.ReportMetric(float64(trieSize), "exact_trie_bytes")
				b.ReportMetric(float64(trieSize)*100/float64(size), "trie_percent_of_total")
				b.ReportMetric(float64(delimiterOverhead), "delimiter_storage_bytes")
				b.ReportMetric(float64(bucketOverhead), "bucket_overhead_bytes")
			}
		})
	}
}
