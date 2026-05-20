package main

import (
	"Thesis/emptiness/exact/ere"
	ere_one_d "Thesis/emptiness/exact/ere_one_d"
	"fmt"
	"math/rand"
    "sort"
)

func main() {
	n := 1 << 20
	keys := make([]uint64, n)
	for i := 0; i < n; i++ {
		keys[i] = rand.Uint64()
	}
    sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
    // unique
    unique := make([]uint64, 0, n)
    unique = append(unique, keys[0])
    for i := 1; i < n; i++ {
        if keys[i] != keys[i-1] {
            unique = append(unique, keys[i])
        }
    }
    keys = unique
    n = len(keys)

	e1, _ := ere.NewExactRangeEmptiness(keys, 64)
	e2, _ := ere_one_d.NewExactRangeEmptiness(keys, 64)

	fmt.Printf("D1+D2 MetadataAllocBits: %d (%.3f BPK)\n", e1.MetadataAllocBits(), float64(e1.MetadataAllocBits())/float64(n))
	fmt.Printf("D MetadataAllocBits: %d (%.3f BPK)\n", e2.MetadataAllocBits(), float64(e2.MetadataAllocBits())/float64(n))
	fmt.Printf("Delta: %.3f BPK\n", float64(e1.MetadataAllocBits()-e2.MetadataAllocBits())/float64(n))
}
