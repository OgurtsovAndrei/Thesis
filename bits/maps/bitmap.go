package maps

import (
	"Thesis/bits"
	"math/rand"
	"time"
)

// Entry represents a key-value pair in a BitMap.
type Entry[V any] struct {
	Key   bits.BitString
	Value V
}

// BitMap is a high-performance map using BitString as keys.
// It uses a single entry per hash and rehashes the entire map with a new seed
// if a hash collision is detected (Las Vegas algorithm).
type BitMap[V any] struct {
	m     map[uint64]Entry[V]
	seed  uint64
	count int
}

func NewBitMap[V any]() *BitMap[V] {
	return &BitMap[V]{
		m:    make(map[uint64]Entry[V]),
		seed: uint64(time.Now().UnixNano()),
	}
}

func (bm *BitMap[V]) Put(key bits.BitString, value V) {
	h := key.HashWithSeed(bm.seed)
	if existing, ok := bm.m[h]; ok {
		if !existing.Key.Equal(key) {
			// Collision detected! Rehash with a new seed.
			bm.rehash()
			bm.Put(key, value)
			return
		}
		// Update existing value
		bm.m[h] = Entry[V]{key, value}
		return
	}
	bm.m[h] = Entry[V]{key, value}
	bm.count++
}

func (bm *BitMap[V]) Get(key bits.BitString) (V, bool) {
	h := key.HashWithSeed(bm.seed)
	if e, ok := bm.m[h]; ok {
		if e.Key.Equal(key) {
			return e.Value, true
		}
	}
	var zero V
	return zero, false
}

func (bm *BitMap[V]) Delete(key bits.BitString) {
	h := key.HashWithSeed(bm.seed)
	if e, ok := bm.m[h]; ok {
		if e.Key.Equal(key) {
			delete(bm.m, h)
			bm.count--
		}
	}
}

func (bm *BitMap[V]) Len() int {
	return bm.count
}

func (bm *BitMap[V]) Range(f func(key bits.BitString, value V) bool) {
	for _, e := range bm.m {
		if !f(e.Key, e.Value) {
			return
		}
	}
}

func (bm *BitMap[V]) rehash() {
	newSeed := uint64(rand.Uint64())
	// Avoid same seed
	if newSeed == bm.seed {
		newSeed++
	}

	oldMap := bm.m
	bm.m = make(map[uint64]Entry[V], len(oldMap))
	bm.seed = newSeed
	bm.count = 0

	for _, e := range oldMap {
		bm.Put(e.Key, e.Value)
	}
}
