package zfasttrie

import (
	"Thesis/bits"
	"Thesis/errutil"
	boomphf "Thesis/mmph/go-boomphf-bs"
)

// see https://arxiv.org/abs/1804.04720

type HNodeData[E UNumber] struct {
	extentLen E // Length of the extent (prefix) represented by this node, should be at least log w, -1 stands for inf

	// Debug field - only populated when saveOriginalTrie is true
	originalNode *znode[bool]
}

type HZFastTrie[E UNumber] struct {
	mph    *boomphf.H     // Minimal Perfect Hash function mapping prefixes to indices.
	data   []HNodeData[E] // Flat array of node data, indexed via the MPH.
	rootId uint64

	// Debug field - only populated when saveOriginalTrie is true
	trie *ZFastTrie[bool]
}

func NewHZFastTrie[E UNumber](keys []bits.BitString) *HZFastTrie[E] {
	errutil.BugOn(!areSorted(keys), "Keys should be sorted")

	if len(keys) == 0 {
		return nil
	}
	trie := Build(keys)
	errutil.BugOn(trie == nil || trie.root == nil, "Trie should not be nil")

	kv := make(map[bits.BitString]HNodeData[E], 0)

	for handle, node := range trie.handle2NodeMap {
		a := uint64(node.nameLength - 1)
		if a == ^uint64(0) {
			a = 0
		}
		b := uint64(node.extentLength())

		original := bits.TwoFattest(a, b)
		b = original - 1
		errutil.BugOn(original != uint64(handle.Size()), "broken handle")

		extentLen := E(node.extentLength())
		errutil.BugOn(uint64(extentLen) != uint64(node.extentLength()), "Data loss on extent length")
		kv[node.extent.Prefix(int(original))] = HNodeData[E]{
			extentLen:    extentLen,
			originalNode: node,
		}

		for a < b {
			ftst := bits.TwoFattest(a, b)
			kv[node.extent.Prefix(int(ftst))] = HNodeData[E]{
				extentLen:    ^E(0), // inf
				originalNode: node,
			}
			b = ftst - 1
		}

	}

	keysForMPH := make([]bits.BitString, 0, len(kv))
	for handle := range kv {
		keysForMPH = append(keysForMPH, handle)
	}
	mph := boomphf.New(boomphf.Gamma, keysForMPH)

	data := make([]HNodeData[E], len(keysForMPH))
	for key, value := range kv {
		idx := mph.Query(key) - 1
		errutil.BugOn(idx >= uint64(len(data)), "Out of bounds")
		data[idx] = value
	}

	var rootIdx uint64
	rootHandle := trie.root.handle()
	rootQuery := mph.Query(rootHandle)
	if rootQuery == 0 {
		errutil.Bug("Root is empty")
	} else {
		rootIdx = rootQuery - 1
	}

	return &HZFastTrie[E]{
		mph:    mph,
		data:   data,
		rootId: rootIdx,
		trie:   trie,
	}
}

func (hzft *HZFastTrie[E]) GetExistingPrefix(pattern bits.BitString) *HNodeData[E] {
	if len(hzft.data) == 0 {
		return nil
	}
	//todo: hzft.stat.getExitNodeCnt++
	patternLength := int32(pattern.Size())
	a := int32(0)
	b := patternLength
	var result = &hzft.data[hzft.rootId]

	for 0 < (b - a) { // is <= ok?
		//todo: hzft.stat.getExitNodeInnerLoopCnt++
		fFast := bits.TwoFattest(uint64(a), uint64(b))

		handle := pattern.Prefix(int(fFast))
		node := hzft.getNodeData(handle)

		if node != nil && pattern.Size() >= uint32(node.extentLen) {
			if uint64(node.extentLen) < fFast {
				//collision
				//b = int32(fFast) - 1
				errutil.Bug("Extent length is too small")
			}
			if uint32(node.extentLen) > pattern.Size() {
				b = int32(fFast) - 1
			} else {
				a = int32(node.extentLen)
				result = node
			}
		} else {
			b = int32(fFast) - 1
		}
	}

	return result
}

func (hzft *HZFastTrie[E]) getNodeData(bitString bits.BitString) *HNodeData[E] {
	query := hzft.mph.Query(bitString)
	// Query return values from 1 to n, 0 used for no Entry
	if query == 0 {
		return nil
	}
	id := query - 1
	return &hzft.data[id]
}
