package zfasttrie

import (
	"Thesis/bits"
	"Thesis/errutil"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"fmt"
	"math/rand"
)

type UNumber interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// NodeData contains the packed data for a trie node.
type NodeData[E UNumber, S UNumber, I UNumber] struct {
	extentLen E // Length of the extent (prefix) represented by this node,  should be (log w)
	// should be at ((log log n) + (log log w) - (log eps)) bits
	PSig            S // Hash signature for path verification in the probabilistic structure.
	parent          I // Index of the first ancestor where this node is in the left subtree
	minChild        I
	minGreaterChild I

	// todo: remove after testing
	originalNode *znode[bool]
}

// ApproxZFastTrie is a compact probabilistic implementation of a Z-Fast Trie,
// utilizing Minimal Perfect Hashing (MPH) instead of explicit pointers.
type ApproxZFastTrie[E UNumber, S UNumber, I UNumber] struct {
	mph  *boomphf.H          // Minimal Perfect Hash function mapping prefixes to indices.
	data []NodeData[E, S, I] // Flat array of node data, indexed via the MPH.
	seed uint64              // Seed used for computing PSig signatures.

	// will be removed, used ONLY for verification
	trie *ZFastTrie[bool]
}

// Build creates a standard Z-Fast Trie from a set of bit strings.
// It serves as an intermediate step during the construction of the compact version.
func Build(keys []bits.BitString) *ZFastTrie[bool] {
	trie := NewZFastTrie[bool](false)
	for i := 0; i < len(keys); i++ {
		trie.InsertBitString(keys[i], true)
	}
	return trie
}

// NewApproxZFastTrie initializes a compact trie.
// The process involves building a standard trie, generating an MPH, and packing NodeData.
func NewApproxZFastTrie[E UNumber, S UNumber, I UNumber](keys []bits.BitString) (*ApproxZFastTrie[E, S, I], error) {
	errutil.BugOn(!areSorted(keys), "Keys should be sorted")
	trie := Build(keys)
	if trie == nil || trie.root == nil {
		return &ApproxZFastTrie[E, S, I]{seed: rand.Uint64()}, nil
	}

	keysForMPH := make([]bits.BitString, 0, len(trie.handle2NodeMap))
	for handle := range trie.handle2NodeMap {
		keysForMPH = append(keysForMPH, handle)
	}

	if len(keysForMPH) == 0 {
		return &ApproxZFastTrie[E, S, I]{seed: rand.Uint64()}, nil
	}

	mph := boomphf.New(boomphf.Gamma, keysForMPH)
	//fmt.Println("Root node from handle:", trie.handle2NodeMap[trie.root.handle()])

	data := make([]NodeData[E, S, I], len(keysForMPH))
	seed := rand.Uint64()

	for handle, node := range trie.handle2NodeMap {
		idx := mph.Query(handle) - 1
		errutil.BugOn(idx >= uint64(len(data)), "Out of bounds")

		mostLeft := node
		// todo: optimize perf, can be done in O(n) instead of O(n log n)
		for mostLeft.rightChild != nil && mostLeft.leftChild != nil {
			errutil.BugOn(mostLeft.leftChild == nil, "Branch with only one child")
			mostLeft = mostLeft.leftChild
		}
		errutil.BugOn(!mostLeft.value || mostLeft.leftChild != nil || mostLeft.rightChild != nil, "mostLeft is not a leaf")
		minChildHandle := mostLeft.handle()
		minChildIdx := mph.Query(minChildHandle) - 1 // Query return values from 1 to n, 0 used for no Entry
		errutil.BugOn(minChildIdx >= uint64(len(data)), "Out of bounds")
		minChild := I(minChildIdx)
		errutil.BugOn(uint64(minChild) != minChildIdx, "Data loss on minChild index")

		var minGreaterChild = I(idx)
		errutil.BugOn(uint64(minGreaterChild) != idx, "Data loss on minChild index")
		if node.rightChild != nil {
			mostLeft := node.rightChild
			for mostLeft.rightChild != nil && mostLeft.leftChild != nil {
				errutil.BugOn(mostLeft.leftChild == nil, "LeftChild")
				mostLeft = mostLeft.leftChild
			}
			errutil.BugOn(!mostLeft.value || mostLeft.leftChild != nil || mostLeft.rightChild != nil, "mostLeft is not a leaf")
			lmcHandle := mostLeft.handle()
			lmcIdx := mph.Query(lmcHandle) - 1
			errutil.BugOn(lmcIdx >= uint64(len(data)), "Out of bounds")
			minGreaterChild = I(lmcIdx)
			errutil.BugOn(uint64(minGreaterChild) != lmcIdx, "Data loss on minGreaterChild index")
		}

		sig := S(hashBitString(node.extent, seed))
		extentLength := E(node.extentLength())
		errutil.BugOn(uint32(extentLength) != node.extentLength(), "Data loss")

		data[idx] = NodeData[E, S, I]{
			extentLen:       extentLength,
			PSig:            sig,
			parent:          I(idx), // will be set correctly in the next loop
			minChild:        minChild,
			minGreaterChild: minGreaterChild,
			originalNode:    node,
		}
	}

	// Set up parent relationships - find first ancestor where node is in left subtree
	var setParentRecursive func(*znode[bool], I)
	setParentRecursive = func(node *znode[bool], leftAncestor I) {
		if node == nil {
			return
		}

		nodeHandle := node.handle()
		nodeIdx := mph.Query(nodeHandle) - 1
		if nodeIdx < uint64(len(data)) {
			data[nodeIdx].parent = leftAncestor
		}

		// For left child, current node becomes the leftAncestor
		// For right child, leftAncestor stays the same
		setParentRecursive(node.leftChild, I(nodeIdx))
		setParentRecursive(node.rightChild, leftAncestor)
	}

	// Start from root with itself as the left ancestor
	if trie.root != nil {
		rootHandle := trie.root.handle()
		rootQuery := mph.Query(rootHandle)
		if rootQuery == 0 {
			errutil.Bug("Root is empty")
		} else {
			rootIdx := I(rootQuery - 1)
			setParentRecursive(trie.root, rootIdx)
		}
	}

	return &ApproxZFastTrie[E, S, I]{
		mph:  mph,
		data: data,
		seed: seed,
		trie: trie,
	}, nil
}

// GetExistingPrefix finds node the longest existing extent, which is a prefix for a given pattern.
// It implements the "Fat Binary Search" algorithm using 2-fattest numbers.
func (azft *ApproxZFastTrie[E, S, I]) GetExistingPrefix(pattern bits.BitString) *NodeData[E, S, I] {
	if len(azft.data) == 0 {
		return nil
	}
	//todo: azft.stat.getExitNodeCnt++
	patternLength := int32(pattern.Size())
	a := int32(0)
	b := patternLength
	var result *NodeData[E, S, I] = azft.getNodeData(bits.NewBitString(""))

	for 0 < (b - a) { // is <= ok?
		//todo: azft.stat.getExitNodeInnerLoopCnt++
		fFast := bits.TwoFattest(uint64(a), uint64(b))

		handle := pattern.Prefix(int(fFast))
		trie_node := azft.trie.getNode(handle)
		node := azft.getNodeData(handle)
		errutil.BugOn(trie_node != nil && node == nil, "Trie node is nil")
		if node != nil && trie_node != nil {
			errutil.BugOn(uint32(node.extentLen) != trie_node.extentLength(), "Illegal extent length")
		}

		if node != nil && pattern.Size() >= uint32(node.extentLen) && S(hashBitString(pattern.Prefix(int(node.extentLen)), azft.seed)) == node.PSig {
			if uint64(node.extentLen) < fFast {
				// collision
				if debug {
					fmt.Println("Collision detected")
				}
				errutil.BugOn(trie_node != nil, "Not a collision?")
				b = int32(fFast) - 1
			} else {
				ref_node := azft.trie.getNode(handle)
				if ref_node != nil /* collision */ && ref_node.extentLength() != uint32(node.extentLen) /* bug */ {
					fmt.Printf("Undetectable (in prod) collision")
				}
				if node.originalNode.extent.Prefix(int(fFast)) != handle {
					//fmt.Println(azft.trie.getExitNode(pattern))
					//fmt.Println("FN Collision detected", node.originalNode.extent.String())
				}
				a = int32(node.extentLen)
				result = node
			}
		} else {
			b = int32(fFast) - 1
		}
	}

	return result
}

// LowerBound returns candidates for being lower-bound
func (azft *ApproxZFastTrie[E, S, I]) LowerBound(pattern bits.BitString) (*NodeData[E, S, I], *NodeData[E, S, I], *NodeData[E, S, I]) {
	node := azft.GetExistingPrefix(pattern)
	if node == nil {
		return nil, nil, nil
	}
	parentId := node.parent
	parentNode := &azft.data[parentId]
	return &azft.data[node.minChild], &azft.data[node.minGreaterChild], &azft.data[parentNode.minGreaterChild]
}

func (azft *ApproxZFastTrie[E, S, I]) getNodeData(bitString bits.BitString) *NodeData[E, S, I] {
	query := azft.mph.Query(bitString)
	// Query return values from 1 to n, 0 used for no Entry
	if query == 0 {
		return nil
	}
	id := query - 1
	return &azft.data[id]
}

func hashBitString(bs bits.BitString, seed uint64) uint64 {
	return bs.HashWithSeed(seed)
}

func areSorted(keys []bits.BitString) bool {
	for i := 1; i < len(keys); i++ {
		if keys[i-1].Compare(keys[i]) > 0 {
			return false
		}
	}
	return true
}
