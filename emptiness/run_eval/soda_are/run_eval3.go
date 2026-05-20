package main
import (
    "Thesis/emptiness/approx/are_soda_hash"
    "fmt"
    "math/rand"
    "math"
    "sort"
)
func sodaK(n int, rangeLen uint64, epsilon float64) uint32 {
	rTarget := float64(n) * float64(rangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}
	return K
}
func main() {
    n := 1<<20
    keys := make([]uint64, n)
    for i := 0; i < n; i++ {
        keys[i] = rand.Uint64()
    }
    sort.Slice(keys, func(i,j int)bool { return keys[i] < keys[j] })
    f, _ := are_soda_hash.NewSodaAREFromK(keys, sodaK(n, 128, 0.01), 128)
    fmt.Printf("BPK: %.3f\n", float64(f.SizeInBits())/float64(n))
}
