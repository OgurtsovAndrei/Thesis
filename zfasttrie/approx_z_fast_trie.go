package zfasttrie

import (
	"Thesis/bits"
	"Thesis/errutil"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"math/rand"
	"sort"
	"unsafe"
)

type UNumber interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// NodeData contains the packed data for a Trie node.
type NodeData[E UNumber, S UNumber, I UNumber] struct {
	extentLen E // Length of the extent (prefix) represented by this node,  should be (log w)
	// should be at ((log log n) + (log log w) - (log eps)) bits
	PSig            S // Hash signature for path verification in the probabilistic structure.
	parent          I // Index of the first ancestor where this node is in the left subtree
	minChild        I
	minGreaterChild I
	rightChild      I
	Rank            I // Index of this key in the original array (or max value if not a delimiter)

	// Debug field - only populated when saveOriginalTrie is true
	originalNode *znode[bool]
}

// ApproxZFastTrie is a compact probabilistic implementation of a Z-Fast Trie,
// utilizing Minimal Perfect Hashing (MPH) instead of explicit pointers.
type ApproxZFastTrie[E UNumber, S UNumber, I UNumber] struct {
	mph    *boomphf.H          // Minimal Perfect Hash function mapping prefixes to indices.
	data   []NodeData[E, S, I] // Flat array of node data, indexed via the MPH.
	seed   uint64              // Seed used for computing PSig signatures.
	rootId I

	// Debug field - only populated when saveOriginalTrie is true
	Trie *ZFastTrie[bool]
}

// NewApproxZFastTrie initializes a compact Trie with delimiter index information.
// This is used for bucketing where we need to map Trie nodes to bucket indices.
// saveOriginalTrie controls whether to keep debug information (original Trie and node references).
// Uses a random seed from the global rand package.
func NewApproxZFastTrie[E UNumber, S UNumber, I UNumber](keys []bits.BitString, saveOriginalTrie bool) (*ApproxZFastTrie[E, S, I], error) {
	return NewApproxZFastTrieWithSeed[E, S, I](keys, saveOriginalTrie, rand.Uint64())
}

// NewApproxZFastTrieWithSeed initializes a compact Trie with delimiter index information using a specified seed.
// This is used for bucketing where we need to map Trie nodes to bucket indices.
// saveOriginalTrie controls whether to keep debug information (original Trie and node references).
// seed is the value used for computing PSig signatures, allowing deterministic construction.
func NewApproxZFastTrieWithSeed[E UNumber, S UNumber, I UNumber](keys []bits.BitString, saveOriginalTrie bool, seed uint64) (*ApproxZFastTrie[E, S, I], error) {
	errutil.BugOn(!areSorted(keys), "Keys should be sorted")

	trie := Build(keys)
	if trie == nil || trie.root == nil {
		result := &ApproxZFastTrie[E, S, I]{seed: seed}
		if saveOriginalTrie {
			result.Trie = trie
		}
		return result, nil
	}

	keysForMPH := make([]bits.BitString, 0, len(trie.handle2NodeMap))
	for handle := range trie.handle2NodeMap {
		keysForMPH = append(keysForMPH, handle)
	}
	// Sort keysForMPH to ensure deterministic order (map iteration is non-deterministic)
	sort.Slice(keysForMPH, func(i, j int) bool {
		return keysForMPH[i].Compare(keysForMPH[j]) < 0
	})

	if len(keysForMPH) == 0 {
		result := &ApproxZFastTrie[E, S, I]{seed: seed}
		if saveOriginalTrie {
			result.Trie = trie
		}
		return result, nil
	}

	mph := boomphf.New(boomphf.Gamma, keysForMPH)

	data := make([]NodeData[E, S, I], len(keysForMPH))

	// Create mapping from keys to their delimiter indices using hash for efficiency
	keyToDelimiterIdx := make(map[bits.BitString]int)
	for i, key := range keys {
		keyToDelimiterIdx[key] = i
	}

	maxDelimiterIndex := I(^I(0)) // Maximum value for I type (means "not a delimiter")

	for handle, node := range trie.handle2NodeMap {
		idx := mph.Query(handle) - 1
		errutil.BugOn(idx >= uint64(len(data)), "Out of bounds")

		mostLeft := node
		// Find the leftmost node by always going left when possible
		for mostLeft.leftChild != nil {
			mostLeft = mostLeft.leftChild
		}
		// With mixed-size strings, mostLeft might have a right child (unbalanced tree)
		// We only require that mostLeft has a value (is a valid node)
		errutil.BugOn(!mostLeft.value, "mostLeft should have a value")
		minChildHandle := mostLeft.handle()
		minChildIdx := mph.Query(minChildHandle) - 1
		errutil.BugOn(minChildIdx >= uint64(len(data)), "Out of bounds")
		minChild := I(minChildIdx)
		errutil.BugOn(uint64(minChild) != minChildIdx, "Data loss on minChild index")

		var minGreaterChild = maxDelimiterIndex
		if node.rightChild != nil {
			mostLeft := node.rightChild
			// Find the leftmost node in the right subtree
			for mostLeft.leftChild != nil {
				mostLeft = mostLeft.leftChild
			}
			// With mixed-size strings, mostLeft might have a right child (unbalanced tree)
			// We only require that mostLeft has a value (is a valid node)
			errutil.BugOn(!mostLeft.value, "mostLeft in right subtree should have a value")
			lmcHandle := mostLeft.handle()
			lmcIdx := mph.Query(lmcHandle) - 1
			errutil.BugOn(lmcIdx >= uint64(len(data)), "Out of bounds")
			minGreaterChild = I(lmcIdx)
			errutil.BugOn(uint64(minGreaterChild) != lmcIdx, "Data loss on minGreaterChild index")
		}

		sig := S(hashBitString(node.extent, seed))
		extentLength := E(node.extentLength())
		errutil.BugOn(uint32(extentLength) != node.extentLength(), "Data loss")

		// Determine delimiter index for this node
		delimiterIdx := maxDelimiterIndex
		if delimIdx, exists := keyToDelimiterIdx[node.extent]; exists {
			delimiterIdx = I(delimIdx)
		}

		// Set rightChild index
		var rightChildIdx I = maxDelimiterIndex // default: no right child
		if node.rightChild != nil {
			rcHandle := node.rightChild.handle()
			rcIdx := mph.Query(rcHandle) - 1
			errutil.BugOn(rcIdx >= uint64(len(data)), "Out of bounds")
			rightChildIdx = I(rcIdx)
			errutil.BugOn(uint64(rightChildIdx) != rcIdx, "Data loss on rightChild index")
		}

		nodeData := NodeData[E, S, I]{
			extentLen:       extentLength,
			PSig:            sig,
			parent:          maxDelimiterIndex, // will be set correctly in the next loop
			minChild:        minChild,
			minGreaterChild: minGreaterChild,
			rightChild:      rightChildIdx,
			Rank:            delimiterIdx,
		}
		if saveOriginalTrie {
			nodeData.originalNode = node
		}
		data[idx] = nodeData
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

		setParentRecursive(node.leftChild, I(nodeIdx))
		setParentRecursive(node.rightChild, leftAncestor)
	}

	var rootIdx I
	rootHandle := trie.root.handle()
	rootQuery := mph.Query(rootHandle)
	if rootQuery == 0 {
		errutil.Bug("Root is empty")
	} else {
		rootIdx = I(rootQuery - 1)
		setParentRecursive(trie.root, maxDelimiterIndex)
	}

	result := &ApproxZFastTrie[E, S, I]{
		mph:    mph,
		data:   data,
		seed:   seed,
		rootId: rootIdx,
	}
	if saveOriginalTrie {
		result.Trie = trie
	}
	return result, nil
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
	var result = &azft.data[azft.rootId]

	for 0 < (b - a) { // is <= ok?
		//todo: azft.stat.getExitNodeInnerLoopCnt++
		fFast := bits.TwoFattest(uint64(a), uint64(b))

		handle := pattern.Prefix(int(fFast))
		node := azft.getNodeData(handle)

		if node != nil && pattern.Size() >= uint32(node.extentLen) && S(hashBitString(pattern.Prefix(int(node.extentLen)), azft.seed)) == node.PSig {
			if uint64(node.extentLen) < fFast {
				// collision
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

func (azft *ApproxZFastTrie[E, S, I]) LowerBound(pattern bits.BitString) (*NodeData[E, S, I], *NodeData[E, S, I], *NodeData[E, S, I], *NodeData[E, S, I], *NodeData[E, S, I], *NodeData[E, S, I]) {
	// HERE WE HAVE SOME MAGIC
	// todo: !!! DOC is REALLY REQUIRED !!!

	node := azft.GetExistingPrefix(pattern)
	if node == nil {
		return nil, nil, nil, nil, nil, nil
	}

	parentNode := (*NodeData[E, S, I])(nil)
	if node.parent != I(^I(0)) {
		parentNode = &azft.data[node.parent]
	}

	cand2 := (*NodeData[E, S, I])(nil)
	if node.minGreaterChild != I(^I(0)) {
		cand2 = &azft.data[node.minGreaterChild]
	}
	cand5 := (*NodeData[E, S, I])(nil)
	if node.rightChild != I(^I(0)) {
		cand5 = &azft.data[node.rightChild]
	}

	// work with parents
	cand3 := azft.getMinGreaterFromParent(parentNode)
	cand4 := azft.getGreaterFromParent(parentNode)

	cand6 := (*NodeData[E, S, I])(node)
	return &azft.data[node.minChild], cand2, cand3, cand4, cand5, cand6
}

func (azft *ApproxZFastTrie[E, S, I]) getMinGreaterFromParent(parentNode *NodeData[E, S, I]) *NodeData[E, S, I] {
	if parentNode == nil {
		return nil
	}
	for parentNode.minGreaterChild == I(^I(0)) && parentNode.parent != I(^I(0)) {
		parentId := parentNode.parent
		parentNode = &azft.data[parentId]
	}
	cand := (*NodeData[E, S, I])(nil)
	if parentNode.minGreaterChild != I(^I(0)) {
		cand = &azft.data[parentNode.minGreaterChild]
	}
	return cand
}

func (azft *ApproxZFastTrie[E, S, I]) getGreaterFromParent(parentNode *NodeData[E, S, I]) *NodeData[E, S, I] {
	if parentNode == nil {
		return nil
	}
	for parentNode.rightChild == I(^I(0)) && parentNode.parent != I(^I(0)) {
		parentId := parentNode.parent
		parentNode = &azft.data[parentId]
	}
	cand4 := (*NodeData[E, S, I])(nil)
	if parentNode.rightChild != I(^I(0)) {
		cand4 = &azft.data[parentNode.rightChild]
	}
	return cand4
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

// ByteSize returns the resident size estimate of ApproxZFastTrie in bytes.
//
// Approximate memory model (bits):
//   - Let u be the number of trie nodes materialized in data[] (typically u=O(m),
//     and for a binary trie with m leaves, u <= 2m-1).
//   - AZFT_bits ~= O(1) + MPH_u_bits + u*(E + S + 5*I).
//   - Current implementation keeps originalNode pointer in NodeData layout, so
//     practical model is:
//     AZFT_bits ~= O(1) + MPH_u_bits + u*(E + S + 5*I + 64).
//
// It includes:
//   - top-level struct header (pointers/slice header/scalars),
//   - backing storage of MPH (via mph.Size()),
//   - backing storage of data[] (len * sizeof(NodeData)).
//
// Note: NodeData always includes originalNode pointer field in layout (nil in
// production mode). The debug Trie object itself is not included.
func (azft *ApproxZFastTrie[E, S, I]) ByteSize() int {
	if azft == nil {
		return 0
	}

	// Include struct header: mph pointer, data slice header, seed, rootId, Trie pointer.
	size := int(unsafe.Sizeof(*azft))

	// Size of the MPH (Minimal Perfect Hash function)
	if azft.mph != nil {
		size += azft.mph.Size()
	}

	// Size of data backing array (real NodeData layout including padding/all fields).
	size += len(azft.data) * int(unsafe.Sizeof(NodeData[E, S, I]{}))

	return size
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
