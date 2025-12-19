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
type NodeData[E UNumber, S UNumber] struct {
	extentLen E // Length of the extent (prefix) represented by this node,  should be (log w)
	// should be at ((log log n) + (log log w) - (log eps)) bits
	PSig         S // Hash signature for path verification in the probabilistic structure.
	originalNode *znode[bool]
}

// ApproxZFastTrie is a compact probabilistic implementation of a Z-Fast Trie,
// utilizing Minimal Perfect Hashing (MPH) instead of explicit pointers.
type ApproxZFastTrie[E UNumber, S UNumber] struct {
	mph  *boomphf.H       // Minimal Perfect Hash function mapping prefixes to indices.
	data []NodeData[E, S] // Flat array of node data, indexed via the MPH.
	seed uint64           // Seed used for computing PSig signatures.

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
func NewApproxZFastTrie[E UNumber, S UNumber](keys []bits.BitString) (*ApproxZFastTrie[E, S], error) {
	trie := Build(keys)
	if trie == nil || trie.root == nil {
		return &ApproxZFastTrie[E, S]{seed: rand.Uint64()}, nil
	}

	type pair struct {
		key bits.BitString
		val NodeData[E, S]
	}

	seed := rand.Uint64()
	pairs := make([]pair, 0, len(trie.handle2NodeMap))

	for handleBitString, node := range trie.handle2NodeMap {
		sig := S(hashBitString(node.extent, seed))
		extentLength := E(node.extentLength())
		errutil.BugOn(uint32(extentLength) != node.extentLength(), "Data loss")
		pairs = append(pairs, pair{
			key: handleBitString,
			val: NodeData[E, S]{
				extentLen:    extentLength,
				PSig:         sig,
				originalNode: node,
			},
		})
	}

	if len(pairs) == 0 {
		return &ApproxZFastTrie[E, S]{seed: seed}, nil
	}

	keysForMPH := make([]bits.BitString, len(pairs))
	for i, p := range pairs {
		keysForMPH[i] = p.key
	}

	mph := boomphf.New(boomphf.Gamma, keysForMPH)

	data := make([]NodeData[E, S], len(pairs))
	for _, p := range pairs {
		idx := mph.Query(p.key) - 1 // Query return values from 1 to n, 0 used for no Entry
		if idx < uint64(len(data)) {
			data[idx] = p.val
		}
	}

	return &ApproxZFastTrie[E, S]{
		mph:  mph,
		data: data,
		seed: seed,
		trie: trie,
	}, nil
}

// GetExistingPrefix finds node the longest existing extent, which is a prefix for a given pattern.
// It implements the "Fat Binary Search" algorithm using 2-fattest numbers.
func (azft *ApproxZFastTrie[E, S]) GetExistingPrefix(pattern bits.BitString) *NodeData[E, S] {
	if len(azft.data) == 0 {
		return nil
	}
	//todo: azft.stat.getExitNodeCnt++
	patternLength := int32(pattern.Size())
	a := int32(0)
	b := patternLength
	var result *NodeData[E, S]

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
					fmt.Println(azft.trie.getExitNode(pattern))
					fmt.Println("FN Collision detected", node.originalNode.extent.String())
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

func (azft *ApproxZFastTrie[E, S]) getNodeData(bitString bits.BitString) *NodeData[E, S] {
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
