package azft

import (
	"Thesis/bits"
	"Thesis/errutil"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"Thesis/trie/zft"
	"sort"
)

// NewApproxZFastTrieFromIteratorStreaming creates AZFT with reduced memory overhead.
// Instead of keeping the full ZFT in memory after building, we extract what we need
// and discard it immediately.
//
// The iterator MUST provide keys in sorted order.
//
// Note: This still builds a ZFT temporarily, but:
// 1. We don't keep debug references to nodes
// 2. We discard the ZFT as soon as we extract the data
//
// For truly streaming construction that avoids the ZFT entirely, see the HZFT
// streaming builder which only needs extentLen per node.
func NewApproxZFastTrieFromIteratorStreaming[E UNumber, S UNumber, I UNumber](
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

	// Extract handles for MPH
	keysForMPH := make([]bits.BitString, 0, len(trie.Handle2NodeMap))
	nodeToHandle := make(map[*zft.Node[bool]]bits.BitString)
	for handle, node := range trie.Handle2NodeMap {
		keysForMPH = append(keysForMPH, handle)
		nodeToHandle[node] = handle
	}
	// Sort keysForMPH to ensure deterministic order
	sort.Slice(keysForMPH, func(i, j int) bool {
		return keysForMPH[i].Compare(keysForMPH[j]) < 0
	})

	if len(keysForMPH) == 0 {
		return &ApproxZFastTrie[E, S, I]{seed: seed}, nil
	}

	mph := boomphf.New(boomphf.Gamma, keysForMPH)
	data := make([]NodeData[E, S, I], len(keysForMPH))

	// Reconstruct ranks by traversing Trie in-order
	nodeRanks := make([]I, len(keysForMPH))
	maxDelimiterIndex := I(^I(0))
	for i := range nodeRanks {
		nodeRanks[i] = maxDelimiterIndex
	}

	rank := 0
	if trie.Root != nil {
		stack := []*zft.Node[bool]{trie.Root}
		for len(stack) > 0 {
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if node.Value {
				handle := nodeToHandle[node]
				nodeIdx := mph.Query(handle) - 1
				errutil.BugOn(nodeIdx >= uint64(len(nodeRanks)), "MPH out of bounds")
				nodeRanks[nodeIdx] = I(rank)
				rank++
			}

			if node.RightChild != nil {
				stack = append(stack, node.RightChild)
			}
			if node.LeftChild != nil {
				stack = append(stack, node.LeftChild)
			}
		}
	}

	// Helper to find the node with the smallest rank in a subtree (leftmost with value)

	findSmallestRankNode := func(root *zft.Node[bool]) *zft.Node[bool] {

		if root == nil {

			return nil

		}

		// Pre-order traversal to find the first node with a value

		stack := []*zft.Node[bool]{root}

		for len(stack) > 0 {

			node := stack[len(stack)-1]

			stack = stack[:len(stack)-1]

			if node.Value {

				return node

			}

			if node.RightChild != nil {

				stack = append(stack, node.RightChild)

			}

			if node.LeftChild != nil {

				stack = append(stack, node.LeftChild)

			}

		}

		return nil

	}

	for _, node := range trie.Handle2NodeMap {

		handle := nodeToHandle[node]

		idx := mph.Query(handle) - 1

		errutil.BugOn(idx >= uint64(len(data)), "Out of bounds")

		// minChild points to the smallest rank in the LEFT subtree

		var minChild = maxDelimiterIndex

		if node.LeftChild != nil {

			smallest := findSmallestRankNode(node.LeftChild)

			if smallest != nil {

				sHandle := nodeToHandle[smallest]

				minChildIdx := mph.Query(sHandle) - 1

				minChild = I(minChildIdx)

			}

		}

		// minGreaterChild points to the smallest rank in the RIGHT subtree

		var minGreaterChild = maxDelimiterIndex

		if node.RightChild != nil {

			smallestGreater := findSmallestRankNode(node.RightChild)

			if smallestGreater != nil {

				lmcHandle := nodeToHandle[smallestGreater]

				lmcIdx := mph.Query(lmcHandle) - 1

				minGreaterChild = I(lmcIdx)

			}

		}

		sig := S(hashBitString(node.Extent, seed))

		extentLength := E(node.ExtentLength())

		errutil.BugOn(uint32(extentLength) != node.ExtentLength(), "Data loss")

		// Determine delimiter index for this node

		delimiterIdx := nodeRanks[idx]

		errutil.BugOn(delimiterIdx != maxDelimiterIndex && delimiterIdx > 255, "Rank overflow for uint8: %d", delimiterIdx)

		// Set up NodeData

		data[idx] = NodeData[E, S, I]{

			extentLen: extentLength,

			PSig: sig,

			parent: maxDelimiterIndex, // will be set correctly in the next loop

			minChild: minChild,

			minGreaterChild: minGreaterChild,

			Rank: uint8(delimiterIdx),
		}

	}

	// Set up parent relationships - find first ancestor where node is in left subtree
	var setParentRecursive func(*zft.Node[bool], I)
	setParentRecursive = func(node *zft.Node[bool], leftAncestor I) {
		if node == nil {
			return
		}

		nodeHandle := node.Handle()
		nodeIdx := mph.Query(nodeHandle) - 1
		if nodeIdx < uint64(len(data)) {
			data[nodeIdx].parent = leftAncestor
		}

		setParentRecursive(node.LeftChild, I(nodeIdx))
		setParentRecursive(node.RightChild, leftAncestor)
	}

	var rootIdx I
	rootHandle := trie.Root.Handle()
	rootQuery := mph.Query(rootHandle)
	if rootQuery == 0 {
		errutil.Bug("Root is empty")
	} else {
		rootIdx = I(rootQuery - 1)
		setParentRecursive(trie.Root, maxDelimiterIndex)
	}

	// Trie is discarded here (goes out of scope) - no debug reference kept
	return &ApproxZFastTrie[E, S, I]{
		mph:    mph,
		data:   data,
		seed:   seed,
		rootId: rootIdx,
	}, nil
}
