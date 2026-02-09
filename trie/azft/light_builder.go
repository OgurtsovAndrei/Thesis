package azft

import (
	"Thesis/bits"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"Thesis/trie/zft"
	"sort"
)

// NewApproxZFastTrieFromIteratorLight creates AZFT by building a temporary heavy ZFT.
func NewApproxZFastTrieFromIteratorLight[E UNumber, S UNumber, I UNumber](
	iter bits.BitStringIterator,
	seed uint64,
) (*ApproxZFastTrie[E, S, I], error) {
	checkedIter := bits.NewCheckedSortedIterator(iter)
	trie, err := zft.BuildFromIterator(checkedIter)
	if err != nil {
		return nil, err
	}
	if trie == nil || trie.Root == nil {
		return &ApproxZFastTrie[E, S, I]{seed: seed}, nil
	}

	// Collect ALL handles (real and pseudo)
	handleMap := make(map[string]bits.BitString)
	
	var collectHandles func(n *zft.Node[bool])
	collectHandles = func(n *zft.Node[bool]) {
		if n == nil { return }
		
		a := uint64(0)
		if n.NameLength > 0 {
			a = uint64(n.NameLength - 1)
		}
		extentLen := uint64(n.ExtentLength())
		original := bits.TwoFattest(a, extentLen)
		
		desc := n.Extent.Prefix(int(original))
		handleMap[desc.PrettyString()] = desc

		if original > 0 {
			b_pseudo := original - 1
			for a < b_pseudo {
				ftst := bits.TwoFattest(a, b_pseudo)
				descPseudo := n.Extent.Prefix(int(ftst))
				handleMap[descPseudo.PrettyString()] = descPseudo
				b_pseudo = ftst - 1
			}
		}
		
		collectHandles(n.LeftChild)
		collectHandles(n.RightChild)
	}
	collectHandles(trie.Root)

	keysForMPH := make([]bits.BitString, 0, len(handleMap))
	for _, h := range handleMap {
		keysForMPH = append(keysForMPH, h)
	}
	sort.Slice(keysForMPH, func(i, j int) bool {
		return keysForMPH[i].Compare(keysForMPH[j]) < 0
	})

	mph := boomphf.New(boomphf.Gamma, keysForMPH)
	data := make([]NodeData[E, S, I], len(keysForMPH))
	maxI := I(^I(0))
	for i := range data {
		data[i] = NodeData[E, S, I]{
			extentLen:       E(^E(0)),
			parent:          maxI,
			minChild:        maxI,
			minGreaterChild: maxI,
			rightChild:      maxI,
			Rank:            maxI,
		}
	}

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

	for _, node := range trie.Handle2NodeMap {
		handle := node.Handle()
		idx := mph.Query(handle) - 1
		
		mostLeft := node
		for mostLeft.LeftChild != nil {
			mostLeft = mostLeft.LeftChild
		}
		minChildHandle := mostLeft.Handle()
		minChildIdx := mph.Query(minChildHandle) - 1
		minChild := I(minChildIdx)

		var minGreaterChild = maxI
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

		delimiterIdx := maxI
		if dIdx, exists := keyToDelimiterIdx[string(node.Extent.Data())]; exists {
			delimiterIdx = I(dIdx)
		}

		var rightChildIdx I = maxI
		if node.RightChild != nil {
			rcHandle := node.RightChild.Handle()
			rcIdx := mph.Query(rcHandle) - 1
			rightChildIdx = I(rcIdx)
		}

		data[idx] = NodeData[E, S, I]{
			extentLen:       extentLength,
			PSig:            sig,
			parent:          maxI,
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
	setParentRecursive(trie.Root, maxI)

	return &ApproxZFastTrie[E, S, I]{
		mph:    mph,
		data:   data,
		seed:   seed,
		rootId: rootIdx,
	}, nil
}
