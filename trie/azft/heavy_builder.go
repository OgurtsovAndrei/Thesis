package azft

import (
	"Thesis/bits"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"Thesis/trie/zft"
	"math/rand"
	"sort"
)

// NewApproxZFastTrieHeavy creates an AZFT by keeping the original ZFT in memory.
//
// Deprecated: This represents the "old" implementation used for performance comparison.
func NewApproxZFastTrieHeavy[E UNumber, S UNumber, I UNumber](keys []bits.BitString) (*ApproxZFastTrie[E, S, I], error) {
	seed := rand.Uint64()
	iter := bits.NewSliceBitStringIterator(keys)
	checkedIter := bits.NewCheckedSortedIterator(iter)
	trie, err := zft.BuildFromIterator(checkedIter)
	if err != nil {
		return nil, err
	}
	if trie == nil || trie.Root == nil {
		return &ApproxZFastTrie[E, S, I]{seed: seed}, nil
	}

	keysForMPH := make([]bits.BitString, 0, len(trie.Handle2NodeMap))
	for handle := range trie.Handle2NodeMap {
		keysForMPH = append(keysForMPH, handle)
	}
	sort.Slice(keysForMPH, func(i, j int) bool {
		return keysForMPH[i].Compare(keysForMPH[j]) < 0
	})

	mph := boomphf.New(boomphf.Gamma, keysForMPH)
	data := make([]NodeData[E, S, I], len(keysForMPH))

	keyToDelimiterIdx := make(map[string]int)
	trieIter := zft.NewSortedIterator(trie)
	rank := 0
	for trieIter.Next() {
		node := trieIter.Node()
		if node.Value {
			keyToDelimiterIdx[string(node.Extent.Data())] = rank
			rank++
		}
	}

	maxDelimiterIndex := I(^I(0))

	for handle, node := range trie.Handle2NodeMap {
		idx := mph.Query(handle) - 1
		
		mostLeft := node
		for mostLeft.LeftChild != nil {
			mostLeft = mostLeft.LeftChild
		}
		minChildHandle := mostLeft.Handle()
		minChildIdx := mph.Query(minChildHandle) - 1
		minChild := I(minChildIdx)

		var minGreaterChild = maxDelimiterIndex
		if node.RightChild != nil {
			mostLeft := node.RightChild
			for mostLeft.LeftChild != nil {
				mostLeft = mostLeft.LeftChild
			}
			lmcHandle := mostLeft.Handle()
			lmcIdx := mph.Query(lmcHandle) - 1
			minGreaterChild = I(lmcIdx)
		}

		sig := S(hashBitString(node.Extent, seed))
		extentLength := E(node.ExtentLength())

		delimiterIdx := maxDelimiterIndex
		if dIdx, exists := keyToDelimiterIdx[string(node.Extent.Data())]; exists {
			delimiterIdx = I(dIdx)
		}

		var rightChildIdx I = maxDelimiterIndex
		if node.RightChild != nil {
			rcHandle := node.RightChild.Handle()
			rcIdx := mph.Query(rcHandle) - 1
			rightChildIdx = I(rcIdx)
		}

		data[idx] = NodeData[E, S, I]{
			extentLen:       extentLength,
			PSig:            sig,
			parent:          maxDelimiterIndex,
			minChild:        minChild,
			minGreaterChild: minGreaterChild,
			rightChild:      rightChildIdx,
			Rank:            delimiterIdx,
		}
	}

	var setParentRecursive func(*zft.Node[bool], I)
	setParentRecursive = func(node *zft.Node[bool], leftAncestor I) {
		if node == nil {
			return
		}
		nodeHandle := node.Handle()
		nodeIdx := mph.Query(nodeHandle) - 1
		data[nodeIdx].parent = leftAncestor
		setParentRecursive(node.LeftChild, I(nodeIdx))
		setParentRecursive(node.RightChild, leftAncestor)
	}

	rootHandle := trie.Root.Handle()
	rootIdx := I(mph.Query(rootHandle) - 1)
	setParentRecursive(trie.Root, maxDelimiterIndex)

	return &ApproxZFastTrie[E, S, I]{
		mph:    mph,
		data:   data,
		seed:   seed,
		rootId: rootIdx,
	}, nil
}
