package zfasttrie

import "Thesis/bits"

type NodeInfo struct {
	Extent bits.BitString
	IsLeaf bool
}

type Iterator struct {
	trie     *ZFastTrie[bool]
	stack    []*znode[bool]
	finished bool
}

func (it *Iterator) Next() bool {
	if it.finished {
		return false
	}

	if len(it.stack) == 0 {
		if it.trie.root == nil {
			it.finished = true
			return false
		}
		it.stack = append(it.stack, it.trie.root)
		it.stack = append(it.stack, mostLeft(it.trie.root)...)
		return true
	}

	node := it.stack[len(it.stack)-1]
	if node.rightChild != nil {
		it.stack = append(it.stack, node.rightChild)
		it.stack = append(it.stack, mostLeft(node.rightChild)...)
		return true
	}

	for len(it.stack) > 1 && it.stack[len(it.stack)-1] == it.stack[len(it.stack)-2].rightChild {
		it.stack = it.stack[:len(it.stack)-1]
	}
	if len(it.stack) > 1 {
		it.stack = it.stack[:len(it.stack)-1]
		return true
	}
	it.finished = true
	return false
}

func mostLeft(node *znode[bool]) (path []*znode[bool]) {
	path = make([]*znode[bool], 0)
	for node.leftChild != nil {
		node = node.leftChild
		path = append(path, node)
	}
	return path
}

func (it *Iterator) Node() *NodeInfo {
	if len(it.stack) == 0 {
		return nil
	}
	node := it.stack[len(it.stack)-1]
	return &NodeInfo{
		Extent: node.extent,
		IsLeaf: node.isLeaf(),
	}
}

func NewIterator(zt *ZFastTrie[bool]) *Iterator {
	return &Iterator{
		trie:     zt,
		stack:    []*znode[bool]{},
		finished: false,
	}
}
