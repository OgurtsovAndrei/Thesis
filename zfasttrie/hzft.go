package zfasttrie

import (
	"Thesis/bits"
	"Thesis/errutil"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"fmt"
	"strings"
	"unsafe"
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

func NewHZFastTrieFromIterator[E UNumber](iter bits.BitStringIterator) (*HZFastTrie[E], error) {
	checkedIter := bits.NewCheckedSortedIterator(iter)
	trie, err := BuildFromIterator(checkedIter)
	if err != nil {
		return nil, err
	}
	if trie == nil || trie.root == nil {
		return nil, nil
	}

	kv := make(map[bits.BitString]HNodeData[E], 0)

	for handle, node := range trie.handle2NodeMap {
		errutil.BugOn(node.handle() != handle, "handle")
		a := uint64(node.nameLength - 1)
		if a == ^uint64(0) {
			a = 0
		}
		b := uint64(node.extentLength())

		original := bits.TwoFattest(a, b) // TwoFattest on (a, b]
		errutil.BugOn(original != uint64(handle.Size()), "broken handle")

		extentLen := E(node.extentLength())
		errutil.BugOn(uint64(extentLen) != uint64(node.extentLength()), "Data loss on extent length")
		errutil.BugOn(node.extent.Prefix(int(original)) != handle, "handle")
		kv[node.extent.Prefix(int(original))] = HNodeData[E]{
			extentLen:    extentLen,
			originalNode: node,
		}
		if original == 0 {
			continue
		}
		b = original - 1
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
	}, nil
}

func NewHZFastTrie[E UNumber](keys []bits.BitString) *HZFastTrie[E] {
	errutil.BugOn(!areSorted(keys), "Keys should be sorted")

	hzft, err := NewHZFastTrieFromIterator[E](bits.NewSliceBitStringIterator(keys))
	if err != nil {
		panic(err)
	}
	return hzft
}

/*
Algorithm 1
Input: a prefix p of some string in S.

i ← floor(log|p|)
l,r ← 0, |p|
while r - l > 1 do

	if exists b such that 2^i * b in (l..r) then
	    // 2^i * b is 2-fattest number in (l..r)
	    g ← T(p[0..2^i * b))
	    if g >= |p| then
	        r ← 2^i * b
	    else
	        l ← g
	    end if
	end if
	i ← i - 1

end while
if l = 0 then

	return ε

else

	return p[0..l+1)

end if
*/

func (hzft *HZFastTrie[E]) GetExistingPrefix(pattern bits.BitString) int64 {
	if len(hzft.data) == 0 || pattern.IsEmpty() {
		return 0
	}

	patternLength := uint32(pattern.Size())
	rootData := hzft.data[hzft.rootId]
	rootExtentLen := uint32(rootData.extentLen)

	if rootExtentLen >= patternLength {
		return 0
	}

	l := uint64(rootExtentLen)
	r := uint64(patternLength)
	maxI := bits.MostSignificantBit(uint64(patternLength))

	for i := maxI; i >= 0; i-- {
		if r-l <= 1 {
			break
		}
		f := uint64((r-1)>>uint(i)) << uint(i)

		if f > l && f < r {
			g := hzft.queryT(pattern.Prefix(int(f)))
			if g >= patternLength {
				r = f
			} else {
				l = uint64(g)
			}
		}
	}

	return int64(l + 1)
}

// queryT implements the function T from the paper
// Maps descriptors to extent lengths, pseudo-descriptors to infinity
func (hzft *HZFastTrie[E]) queryT(prefix bits.BitString) uint32 {
	nodeData := hzft.getNodeData(prefix)
	if nodeData == nil {
		return ^uint32(0) // infinity - not found
	}
	if uint64(nodeData.extentLen) == uint64(^E(0)) {
		return ^uint32(0) // infinity - pseudo-descriptor
	}
	return uint32(nodeData.extentLen)
}

func (hzft *HZFastTrie[E]) getNodeData(bitString bits.BitString) *HNodeData[E] {
	// Query return values from 1 to n, 0 used for no Entry
	query := hzft.mph.Query(bitString)
	if query == 0 {
		return nil
	}
	id := query - 1
	return &hzft.data[id]
}

func (hzft *HZFastTrie[E]) String() string {
	var sb strings.Builder
	sb.WriteString("HZFastTrie:\n")
	sb.WriteString(fmt.Sprintf("| rootId: %d\n", hzft.rootId))
	sb.WriteString(fmt.Sprintf("| data length: %d\n", len(hzft.data)))

	sb.WriteString("| MPH mappings:\n")
	for i, nodeData := range hzft.data {
		sb.WriteString(fmt.Sprintf("  [%d] extentLen=%d", i, nodeData.extentLen))
		if nodeData.originalNode != nil {
			sb.WriteString(fmt.Sprintf(" extent=%q", nodeData.originalNode.extent.PrettyString()))
		}
		if uint64(nodeData.extentLen) == uint64(^E(0)) {
			sb.WriteString(" (pseudo-descriptor)")
		}
		sb.WriteString("\n")
	}

	if hzft.trie != nil {
		sb.WriteString("| Original Trie:\n")
		sb.WriteString(strings.ReplaceAll(hzft.trie.String(), "\n", "\n  "))
	}

	return sb.String()
}

// ByteSize returns the total size of the structure in bytes.
func (hzft *HZFastTrie[E]) ByteSize() int {
	if hzft == nil {
		return 0
	}

	size := 0

	// Size of the MPH (Minimal Perfect Hash function)
	if hzft.mph != nil {
		size += hzft.mph.Size()
	}

	// Size of the data array (HNodeData entries)
	// Each HNodeData contains: extentLen(E) + originalNode pointer
	nodeDataSize := len(hzft.data) * (int(unsafe.Sizeof(*new(E))) /* + int(unsafe.Sizeof((*znode[bool])(nil)))*/)
	size += nodeDataSize

	// Size of rootId (uint64)
	size += 8

	// Size of debug Trie pointer (always present in struct, but nil in production)
	size += int(unsafe.Sizeof((*ZFastTrie[bool])(nil)))

	return size
}
