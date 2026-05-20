package main

import (
	"Thesis/emptiness/exact/ere"
	ere_one_d "Thesis/emptiness/exact/ere_one_d"
    "Thesis/emptiness/internal/hash"
	"fmt"
	"math/rand"
    "sort"
)

func main() {
	n := 1 << 20
	keys := make([]uint64, n)
	step := (uint64(1) << 64) / uint64(n)
	for i := 0; i < n; i++ {
		keys[i] = uint64(i)*step + (rand.Uint64() % step)
	}

    // SODA hash them
    K := uint32(34)
    h, _ := hash.NewLocalityPreservingHash(64, K, n)
    
    hashedKeys := make([]uint64, n)
    for i := 0; i < n; i++ {
        hashedKeys[i] = h.Hash(keys[i])
    }
    
    // SODA ERE requires sorted hashed keys
    sort.Slice(hashedKeys, func(i, j int) bool { return hashedKeys[i] < hashedKeys[j] })
    // Deduplicate
    unique := make([]uint64, 0, n)
    unique = append(unique, hashedKeys[0])
    for i := 1; i < n; i++ {
        if hashedKeys[i] != hashedKeys[i-1] {
            unique = append(unique, hashedKeys[i])
        }
    }
    fmt.Printf("Unique keys: %d / %d\n", len(unique), n)

	e1, _ := ere.NewExactRangeEmptiness(unique, K)
	e2, _ := ere_one_d.NewExactRangeEmptiness(unique, K)

	fmt.Printf("D1+D2 MetadataAllocBits: %d (%.3f BPK)\n", e1.MetadataAllocBits(), float64(e1.MetadataAllocBits())/float64(n))
	fmt.Printf("D MetadataAllocBits: %d (%.3f BPK)\n", e2.MetadataAllocBits(), float64(e2.MetadataAllocBits())/float64(n))
	fmt.Printf("Delta: %.3f BPK\n", float64(e1.MetadataAllocBits()-e2.MetadataAllocBits())/float64(n))
}
