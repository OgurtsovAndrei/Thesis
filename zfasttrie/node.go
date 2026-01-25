package zfasttrie

import (
	"Thesis/bits"
	"fmt"
)

// znode represents a Node in the ZFastTrie.
type znode[V comparable] struct {
	value      V
	extent     bits.BitString
	nameLength int32
	leftChild  *znode[V]
	rightChild *znode[V]
}

func newNode[V comparable](value V, extent bits.BitString) *znode[V] {
	return &znode[V]{
		value:      value,
		extent:     extent,
		nameLength: 0,
	}
}

func newNodeWithNameLength[V comparable](value V, extent bits.BitString, nameLength int32) *znode[V] {
	return &znode[V]{
		value:      value,
		extent:     extent,
		nameLength: nameLength,
	}
}

func (n *znode[V]) set(value V, extent bits.BitString, nameLength int32) {
	n.value = value
	n.extent = extent
	n.nameLength = nameLength
}

func (n *znode[V]) setExtent(extent bits.BitString) {
	if extent.IsEmpty() {
		n.nameLength = 0
	}
	n.extent = extent
}

func (n *znode[V]) extentLength() uint32 {
	return n.extent.Size()
}

func (n *znode[V]) handleLength() uint32 {
	// C++: Fast::twoFattest(nameLength_ - 1, extentLength());
	aFast := uint64(n.nameLength - 1)
	if aFast == ^uint64(0) {
		aFast = 0
	}
	bFast := uint64(n.extentLength())

	return uint32(bits.TwoFattest(aFast, bFast))
}

func (n *znode[V]) handle() bits.BitString {
	if n.extentLength() <= 0 {
		return bits.NewFromText("")
	}
	return bits.NewBitStringPrefix(n.extent, n.handleLength())
}

func (n *znode[V]) isLeaf() bool {
	return n.leftChild == nil && n.rightChild == nil
}

func (n *znode[V]) key() bool {
	// C++: extent_.at(nameLength_ - 1)
	return n.extent.At(uint32(n.nameLength - 1))
}

func (n *znode[V]) sizeChildren() uint32 {
	size := uint32(0)
	if n.leftChild != nil {
		size++
	}
	if n.rightChild != nil {
		size++
	}
	return size
}

func (n *znode[V]) insertChild(child *znode[V], lcpLength uint32) {
	// C++: child->extent_.at(lcpLength)
	if child.extent.At(lcpLength) {
		n.rightChild = child
	} else {
		n.leftChild = child
	}
}

func (n *znode[V]) eraseChild(key bool) {
	if key {
		n.rightChild = nil
	} else {
		n.leftChild = nil
	}
}

func (n *znode[V]) getChild() *znode[V] {
	if n.leftChild != nil {
		return n.leftChild
	}
	return n.rightChild
}

func (n *znode[V]) String() string {
	if n == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Node{value: %v, extent: \"%v\", nameLength: %d, extentLen: %d,  leftChild: %t, rightChild: %t}",
		n.value, n.extent, n.nameLength, n.extentLength(), n.leftChild != nil, n.rightChild != nil)
}
