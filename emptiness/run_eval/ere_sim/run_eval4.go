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
		keys[i] = rand.Uint64() >> 30 // To simulate 34-bit hashed keys
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

	e1, _ := ere.NewExactRangeEmptiness(keys, 34)
	e2, _ := ere_one_d.NewExactRangeEmptiness(keys, 34)

	fmt.Printf("e1 SizeInBits: %d (%.3f BPK)\n", e1.SizeInBits(), float64(e1.SizeInBits())/float64(n))
	fmt.Printf("e2 SizeInBits: %d (%.3f BPK)\n", e2.SizeInBits(), float64(e2.SizeInBits())/float64(n))
	fmt.Printf("Delta: %.3f BPK\n", float64(e1.SizeInBits()-e2.SizeInBits())/float64(n))
}
