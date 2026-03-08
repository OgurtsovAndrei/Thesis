package lemonhash

/*
#cgo CXXFLAGS: -std=c++20
#cgo LDFLAGS: -L${SRCDIR}/build -llemonhash_wrapper -L${SRCDIR}/build/ext/LeMonHash -lsdsl -L${SRCDIR}/build/ext/LeMonHash/extlib/simpleRibbon -lSimpleRibbon -lRibbonSorter -L${SRCDIR}/build/ext/LeMonHash/extlib/tlx/tlx -ltlx -lc++ -lm
#include "wrapper.h"
#include <stdlib.h>
*/
import "C"
import (
	"Thesis/bits"
	"runtime"
	"unsafe"
)

type LeMonHash struct {
	ptr C.LeMonHashPtr
}

// New creates a new LeMonHash wrapper.
// The keys MUST be sorted and unique, just like in other MMPH implementations.
func New(keys []bits.BitString) *LeMonHash {
	if len(keys) == 0 {
		return &LeMonHash{ptr: nil}
	}
	if len(keys) == 1 {
		return &LeMonHash{ptr: nil} // We can return 0 for any query when N=1
	}

	numKeys := len(keys)
	
	// Prepare C arrays
	cStrings := make([]*C.char, numKeys)
	lengths := make([]C.size_t, numKeys)
	
	for i, k := range keys {
		data := k.Data()
		if len(data) == 0 {
			cStrings[i] = nil
		} else {
			cStrings[i] = (*C.char)(C.CBytes(data))
		}
		lengths[i] = C.size_t(len(data))
	}

	ptr := C.lemonhash_vl_new((**C.char)(&cStrings[0]), (*C.size_t)(&lengths[0]), C.size_t(numKeys))

	// Free C strings
	for _, cStr := range cStrings {
		if cStr != nil {
			C.free(unsafe.Pointer(cStr))
		}
	}

	lh := &LeMonHash{ptr: ptr}
	runtime.SetFinalizer(lh, func(obj *LeMonHash) {
		if obj.ptr != nil {
			C.lemonhash_vl_free(obj.ptr)
			obj.ptr = nil
		}
	})

	return lh
}

func (lh *LeMonHash) Rank(key bits.BitString) int {
	if lh.ptr == nil {
		return 0
	}
	
	data := key.Data()
	var cStr *C.char
	if len(data) > 0 {
		cStr = (*C.char)(C.CBytes(data))
		defer C.free(unsafe.Pointer(cStr))
	}
	
	rank := C.lemonhash_vl_query(lh.ptr, cStr, C.size_t(len(data)))
	return int(rank)
}

func (lh *LeMonHash) ByteSize() int {
	if lh.ptr == nil {
		return 0
	}
	return int(C.lemonhash_vl_space_bits(lh.ptr) / 8)
}
