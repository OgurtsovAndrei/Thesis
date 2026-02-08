package zfasttrie

import (
	"Thesis/bits"
	"Thesis/errutil"
	"fmt"
	"log"
	"strings"
)

import (
	"os"
)

var debug bool

func init() {
	if os.Getenv("DEBUG") == "1" {
		debug = true
	} else {
		debug = false
	}
}

type statistics struct {
	getExitNodeCnt          int
	getExitNodeInnerLoopCnt int
}

// ZFastTrie implementation in Go.
// Ported from ZFastTrie.hpp
type ZFastTrie[V comparable] struct {
	size           int32
	root           *znode[V]
	handle2NodeMap map[bits.BitString]*znode[V]
	emptyValue     V

	stat statistics
}

// NewZFastTrie creates a new ZFastTrie.
// emptyValue is the value designated as empty (e.g., nil or 0 in Go).
func NewZFastTrie[V comparable](emptyValue V) *ZFastTrie[V] {
	return &ZFastTrie[V]{
		size:           0,
		root:           nil,
		handle2NodeMap: make(map[bits.BitString]*znode[V]),
		emptyValue:     emptyValue,
	}
}

func (zt *ZFastTrie[V]) Insert(newText string, value V) {
	zt.checkTrie()
	zt.InsertBitString(bits.NewFromText(newText), value)
	zt.checkTrie()
}

func (zt *ZFastTrie[V]) Erase(targetText string) {
	zt.checkTrie()
	zt.EraseBitString(bits.NewFromText(targetText))
	zt.checkTrie()
}

func (zt *ZFastTrie[V]) InsertBitString(newText bits.BitString, value V) {
	if value == zt.emptyValue {
		panic("cannot insert empty value")
	}

	if zt.size == 0 {
		zt.root = newNode(value, newText)
		zt.insertHandle2NodeMap(zt.root)
	} else {
		exitNode := zt.getExitNode(newText)
		if exitNode == nil {
			panic("internal error: exitNode is nil")
		}

		lcpLength := exitNode.extent.GetLCPLength(newText)

		if lcpLength < exitNode.extentLength() {
			exitNodeExtent := exitNode.extent
			oldExitValue := exitNode.value

			var newExtent bits.BitString = bits.NewFromText("")
			if lcpLength > 0 {
				newExtent = exitNode.extent.Prefix(int(lcpLength))
			}

			if exitNode.isLeaf() {
				zt.eraseHandle2NodeMap(exitNodeExtent)
				exitNode.setExtent(newExtent)
				exitNode.value = zt.emptyValue
				zt.insertHandle2NodeMap(exitNode)
			} else {
				// C++: const uint &newHandleLength = Fast::twoFattest(lcpLength, newExtent.size());
				// newExtent.size() == lcpLength.
				// newHandleLength = twoFattest(lcpLength, lcpLength) == 0.
				// C++: bool isChangeHandle = (exitNode->handleLength() != newHandleLength);
				// isChangeHandle = (exitNode->handleLength() != 0)

				oldHandle := exitNode.handle()
				isChangeHandle := !oldHandle.IsEmpty()

				if isChangeHandle {
					zt.eraseHandle2NodeMap(oldHandle)
					exitNode.setExtent(newExtent)
					exitNode.value = zt.emptyValue
					zt.insertHandle2NodeMap(exitNode)
				} else {
					exitNode.setExtent(newExtent)
				}
			}

			// make new internal znode (with previous exit node value)
			newNode := newNodeWithNameLength(oldExitValue, exitNodeExtent, int32(lcpLength+1))
			swapChildren(exitNode, newNode)
			exitNode.insertChild(newNode, lcpLength)
			zt.insertHandle2NodeMap(newNode)

			// finally add node with newText
			if lcpLength < newText.Size() {
				// make new leaf znode
				newTextNode := newNodeWithNameLength(value, newText, int32(lcpLength+1))
				errutil.BugOn(exitNode.value != zt.emptyValue, "value already exists, but how?")
				exitNode.insertChild(newTextNode, lcpLength)
				zt.insertHandle2NodeMap(newTextNode)
			} else {
				errutil.BugOn(exitNode.value != zt.emptyValue, "value already exists, but how?")
				exitNode.value = value
			}

		} else { // lcpLength >= exitNode.extentLength()
			if lcpLength == newText.Size() { // lcpLength == exitNode.extentLength() == newText.Size()
				if exitNode.value == zt.emptyValue {
					exitNode.value = value
				} else {
					if debug {
						log.Println("Warning: new text already exist.")
					}
					exitNode.value = value
				}
			} else { // lcpLength == exitNode.extentLength() < newText.Size()
				if exitNode.isLeaf() {
					zt.insertHandle2NodeMap(exitNode)
				}
				newTextNode := newNodeWithNameLength(value, newText, int32(lcpLength+1))
				exitNode.insertChild(newTextNode, lcpLength)
				zt.insertHandle2NodeMap(newTextNode)
			}
		}
	}

	zt.size++
}

func (zt *ZFastTrie[V]) EraseBitString(targetText bits.BitString) {
	targetNode := zt.getExitNode(targetText)
	if targetNode == nil || !targetNode.extent.Equal(targetText) || targetNode.value == zt.emptyValue {
		if debug {
			log.Println("Warning: trying to erase non-existent key")
		}
		return
	}

	if zt.size <= 1 {
		if zt.root == targetNode {
			zt.eraseHandle2NodeMap(targetNode.handle())
			zt.root = nil
		}
	} else {
		if targetNode.isLeaf() {
			if targetNode == zt.root {
				zt.eraseHandle2NodeMap(targetNode.handle())
				zt.root = nil
			} else {
				var parentNode *znode[V]
				if targetNode.nameLength <= 1 {
					parentNode = zt.root
				} else {
					parentPrefix := targetNode.extent.Prefix(int(targetNode.nameLength - 1))
					parentNode = zt.getExitNode(parentPrefix)
				}

				if parentNode == nil {
					panic("internal error: parentNode not found during erase")
				}
				errutil.BugOn(parentNode.leftChild != targetNode && parentNode.rightChild != targetNode, "wrong parent:\n%s\n%s", parentNode, targetNode)

				parentNode.eraseChild(targetNode.key())
				zt.eraseHandle2NodeMap(targetNode.handle())

				if parentNode.sizeChildren() == 1 && parentNode.value == zt.emptyValue {
					// swap parent and child znode
					zt.eraseHandle2NodeMap(parentNode.handle())
					childNode := parentNode.getChild()

					parentNode.eraseChild(childNode.key())
					swapChildren(parentNode, childNode)

					parentNode.set(childNode.value, childNode.extent, parentNode.nameLength)

					zt.eraseHandle2NodeMap(childNode.handle())
					zt.insertHandle2NodeMap(parentNode)

				}
			}
		} else if targetNode.sizeChildren() == 1 {
			// delete internal znode
			childNode := targetNode.getChild()
			targetNode.eraseChild(childNode.key())
			zt.eraseHandle2NodeMap(targetNode.handle())
			//zt.eraseHandle2NodeMap(childNode.handle())

			targetNode.set(childNode.value, childNode.extent, targetNode.nameLength)
			swapChildren(targetNode, childNode)

			zt.eraseHandle2NodeMap(childNode.handle())
			zt.insertHandle2NodeMap(targetNode)
		} else {
			targetNode.value = zt.emptyValue
		}
	}
	zt.size--
}

func (zt *ZFastTrie[V]) Contains(pattern string) bool {
	return zt.ContainsBitString(bits.NewFromText(pattern))
}

func (zt *ZFastTrie[V]) ContainsBitString(pattern bits.BitString) bool {
	exitNode := zt.getExitNode(pattern)
	if exitNode == nil {
		return false
	}
	return exitNode.extent.Equal(pattern) && exitNode.value != zt.emptyValue
}

func (zt *ZFastTrie[V]) GetBitString(pattern bits.BitString) (value V) {
	exitNode := zt.getExitNode(pattern)

	if exitNode == nil {
		return zt.emptyValue
	}
	if exitNode.extent.Equal(pattern) {
		return exitNode.value
	}
	return zt.emptyValue
}

func swapChildren[V comparable](a, b *znode[V]) {
	a.leftChild, b.leftChild = b.leftChild, a.leftChild
	a.rightChild, b.rightChild = b.rightChild, a.rightChild
}

func (zt *ZFastTrie[V]) checkTrie() {
	if debug {
		cnt := zt.checkTrieRec(zt.root)
		errutil.BugOn(cnt != len(zt.handle2NodeMap), "%d != %d\n%s", cnt, len(zt.handle2NodeMap), zt)
	}
}

func (zt *ZFastTrie[V]) checkTrieRec(node *znode[V]) (notEmptyNodesInTrie int) {
	if node == nil {
		return 0
	}

	if node.nameLength != 0 {
		fFast := node.handleLength()
		f := int32(fFast)
		handle := node.extent.Prefix(int(f))
		handleNode, ok := zt.handle2NodeMap[handle]
		errutil.BugOn(!ok, "on %q, %d != %d\n%s\n%s\n%s", handle, zt.size, f, node, handleNode, zt)
		errutil.BugOn(node != handleNode, "%s\n%s\n%s\n%s", handle, node, handleNode, zt)
	}

	if !node.extent.IsEmpty() {
		notEmptyNodesInTrie = 1
	}

	return notEmptyNodesInTrie + zt.checkTrieRec(node.leftChild) + zt.checkTrieRec(node.rightChild)
}

func (zt *ZFastTrie[V]) insertHandle2NodeMap(n *znode[V]) {
	//if n.extentLength() == 0 {
	//	return
	//}
	handle := n.handle()
	if handle.IsEmpty() {
		errutil.BugOn(zt.handle2NodeMap[handle] != nil, "root already in handle2NodeMap")
		//return
	}
	zt.handle2NodeMap[handle] = n
}

func (zt *ZFastTrie[V]) eraseHandle2NodeMap(handle bits.BitString) {
	//if handle.IsEmpty() {
	//	return
	//}
	delete(zt.handle2NodeMap, handle)
}

// ContainsPrefix checks if the string is a prefix of any entry in the Trie.
func (zt *ZFastTrie[V]) ContainsPrefix(pattern string) bool {
	return zt.containsPrefixBitString(bits.NewFromText(pattern))
}

func (zt *ZFastTrie[V]) containsPrefixBitString(pattern bits.BitString) bool {
	node := zt.getExitNode(pattern)
	if node == nil {
		return false
	}
	return node.extent.HasPrefix(pattern)
}

func (zt *ZFastTrie[V]) getExitNode(pattern bits.BitString) *znode[V] {
	result := zt.getExistingPrefix(pattern)

	if result != nil {
		lcpLength := result.extent.GetLCPLength(pattern)
		if lcpLength == result.extentLength() && lcpLength < pattern.Size() {
			var next *znode[V]
			if pattern.At(lcpLength) {
				next = result.rightChild
			} else {
				next = result.leftChild
			}
			if next != nil {
				result = next
			}
		}
	}

	return result
}

func (zt *ZFastTrie[V]) getExistingPrefix(pattern bits.BitString) *znode[V] {
	zt.stat.getExitNodeCnt++
	patternLength := int32(pattern.Size())
	a := int32(0)
	b := patternLength
	var node *znode[V]
	result := zt.root

	for 0 < (b - a) {
		zt.stat.getExitNodeInnerLoopCnt++
		// C++: f = Fast::twoFattest(a, b);
		fFast := bits.TwoFattest(uint64(a), uint64(b))

		handle := pattern.Prefix(int(fFast))
		node = zt.getNode(handle)

		if node != nil {
			a = int32(node.extentLength())
			result = node
		} else {
			b = int32(fFast) - 1
		}
	}
	return result
}

func (zt *ZFastTrie[V]) getNode(handle bits.BitString) *znode[V] {
	node, _ := zt.handle2NodeMap[handle]
	return node
}

func (zt *ZFastTrie[V]) String() string {
	var sb strings.Builder
	sb.WriteString("ZFastTrie:\n")
	sb.WriteString(fmt.Sprintf("| size: %d\n", zt.size))
	sb.WriteString(fmt.Sprintf("| emptyValue: %v\n", zt.emptyValue))

	sb.WriteString("| root:")
	if zt.root == nil {
		sb.WriteString("nil")
	} else {
		// Pass 2 for the initial indentation of the root node's fields (value, extent, etc.)
		sb.WriteString(zt.stringNode(zt.root, "| | "))
	}

	for bitString, z := range zt.handle2NodeMap {
		sb.WriteString(bitString.PrettyString())
		sb.WriteString(": ")
		sb.WriteString(z.String())
		sb.WriteString("\n")
	}
	if len(zt.handle2NodeMap) == 0 {
		sb.WriteString("<empty handle>")
	}
	sb.WriteString("\n")

	return sb.String()
}

func (zt *ZFastTrie[V]) stringNode(node *znode[V], prefix string) string {
	if node == nil {
		return " nil\n"
	}

	var sb strings.Builder

	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("%svalue: %v\n", prefix, node.value))
	sb.WriteString(fmt.Sprintf("%sextent: %q\n", prefix, node.extent.PrettyString()))
	sb.WriteString(fmt.Sprintf("%snameLength: %d\n", prefix, node.nameLength))
	sb.WriteString(fmt.Sprintf("%sleftChild:", prefix))
	sb.WriteString(zt.stringNode(node.leftChild, prefix+"| "))
	sb.WriteString(fmt.Sprintf("%srightChild:", prefix))
	sb.WriteString(zt.stringNode(node.rightChild, prefix+"| "))

	return sb.String()
}

// BuildFromIterator creates a standard Z-Fast Trie from a sequence of bit strings.
// It serves as an intermediate step during the construction of the compact version.
func BuildFromIterator(iter bits.BitStringIterator) (*ZFastTrie[bool], error) {
	trie := NewZFastTrie[bool](false)
	for iter.Next() {
		trie.InsertBitString(iter.Value(), true)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return trie, nil
}

// Build creates a standard Z-Fast Trie from a set of bit strings.
// It serves as an intermediate step during the construction of the compact version.
func Build(keys []bits.BitString) *ZFastTrie[bool] {
	trie, err := BuildFromIterator(bits.NewSliceBitStringIterator(keys))
	if err != nil {
		panic(err) // Should not happen with slice iterator
	}
	return trie
}
