package maps

import (
	"Thesis/bits"
)

// ArrayBitMap is a map using BitString as keys.
// It handles collisions by storing slices of entries for each hash.
type ArrayBitMap[V any] struct {
	m     map[uint64][]Entry[V]
	count int
}

func NewArrayBitMap[V any]() *ArrayBitMap[V] {
	return &ArrayBitMap[V]{
		m: make(map[uint64][]Entry[V]),
	}
}

func (bm *ArrayBitMap[V]) Put(key bits.BitString, value V) {
	h := key.Hash()
	entries := bm.m[h]
	for i := range entries {
		if entries[i].Key.Equal(key) {
			entries[i].Value = value
			return
		}
	}
	bm.m[h] = append(entries, Entry[V]{Key: key, Value: value})
	bm.count++
}

func (bm *ArrayBitMap[V]) Get(key bits.BitString) (V, bool) {
	h := key.Hash()
	entries := bm.m[h]
	for i := range entries {
		if entries[i].Key.Equal(key) {
			return entries[i].Value, true
		}
	}
	var zero V
	return zero, false
}

func (bm *ArrayBitMap[V]) Delete(key bits.BitString) {
	h := key.Hash()
	entries := bm.m[h]
	for i := range entries {
		if entries[i].Key.Equal(key) {
			bm.m[h] = append(entries[:i], entries[i+1:]...)
			if len(bm.m[h]) == 0 {
				delete(bm.m, h)
			}
			bm.count--
			return
		}
	}
}

func (bm *ArrayBitMap[V]) Len() int {
	return bm.count
}

func (bm *ArrayBitMap[V]) Range(f func(key bits.BitString, value V) bool) {
	for _, entries := range bm.m {
		for _, e := range entries {
			if !f(e.Key, e.Value) {
				return
			}
		}
	}
}
