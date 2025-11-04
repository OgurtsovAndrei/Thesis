package main

import (
	"log"
	"math/bits"
)

// --- BitString Implementation ---
// (Based on usage in ZFastTrie.hpp, assuming string of '0's and '1's)

type BitString string

func NewBitString(s string) BitString {
	return BitString(s)
}

func (bs BitString) size() int {
	return len(bs)
}

func (bs BitString) at(i int) bool {
	if i < 0 || i >= len(bs) {
		log.Panicf("BitString.at: index %d out of bounds for len %d", i, len(bs))
	}
	return bs[i] == '1'
}

func (bs BitString) substring(length int) BitString {
	if length <= 0 {
		return BitString("")
	}
	if length > len(bs) {
		length = len(bs)
	}
	return bs[:length]
}

func (bs BitString) toString() string {
	return string(bs)
}

func getLCPLength(a, b BitString) int {
	lenA, lenB := a.size(), b.size()
	minLen := lenA
	if lenB < minLen {
		minLen = lenB
	}
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return minLen
}

func isPrefix(prefix, s BitString) bool {
	lenP, lenS := prefix.size(), s.size()
	if lenP > lenS {
		return false
	}
	return s.substring(lenP) == prefix
}

// --- Fast Utilities ---

func twoFattest(a, b int) int {
	if b <= a {
		if b < 0 {
			return 0
		}
		return b
	}
	return (a + b + 1) / 2
}

// --- Node Implementation ---

type Node[Value comparable] struct {
	value      Value
	extent     BitString
	nameLength int
	leftChild  *Node[Value]
	rightChild *Node[Value]
}

func NewNode[Value comparable](value Value, extent BitString) *Node[Value] {
	return &Node[Value]{
		value:      value,
		extent:     extent,
		nameLength: 1,
	}
}

func NewNodeWithNameLength[Value comparable](value Value, extent BitString, nameLength int) *Node[Value] {
	return &Node[Value]{
		value:      value,
		extent:     extent,
		nameLength: nameLength,
	}
}

func (n *Node[Value]) set(value Value, extent BitString, nameLength int) {
	n.value = value
	n.extent = extent
	n.nameLength = nameLength
}

func (n *Node[Value]) setExtent(extent BitString) {
	if extent.size() == 0 {
		n.nameLength = 0
	}
	n.extent = extent
}

func (n *Node[Value]) handleLength() int {
	return twoFattest(n.nameLength-1, n.extent.size())
}

func (n *Node[Value]) handle() BitString {
	if n.extent.size() <= 0 {
		return BitString("")
	}
	return n.extent.substring(n.handleLength())
}

func (n *Node[Value]) isLeaf() bool {
	return n.leftChild == nil && n.rightChild == nil
}

func (n *Node[Value]) key() bool {
	return n.extent.at(n.nameLength - 1)
}

func (n *Node[Value]) sizeChildren() int {
	size := 0
	if n.leftChild != nil {
		size++
	}
	if n.rightChild != nil {
		size++
	}
	return size
}

func (n *Node[Value]) insertChild(child *Node[Value], lcpLength int) {
	if child.extent.at(lcpLength) {
		n.rightChild = child
	} else {
		n.leftChild = child
	}
}

func (n *Node[Value]) eraseChild(key bool) {
	if key {
		n.rightChild = nil
	} else {
		n.leftChild = nil
	}
}

func (n *Node[Value]) getChild() *Node[Value] {
	if n.leftChild != nil {
		return n.leftChild
	}
	if n.rightChild != nil {
		return n.rightChild
	}
	return nil
}

func (n *Node[Value]) print(indent string) {
	log.Printf("%sZFastTrie::Node \"%s\", value: %v, nameLength: %d", indent, n.extent.toString(), n.value, n.nameLength)
	if n.leftChild != nil {
		log.Printf("%s  children Left", indent)
		n.leftChild.print(indent + "    ")
	}
	if n.rightChild != nil {
		log.Printf("%s  children Right", indent)
		n.rightChild.print(indent + "    ")
	}
}

// --- ZFastTrie Implementation ---

type ZFastTrie[Value comparable] struct {
	size           int
	root           *Node[Value]
	handle2NodeMap map[string]*Node[Value]
	EMPTY_VALUE    Value
}

func NewZFastTrie[Value comparable](emptyValue Value) *ZFastTrie[Value] {
	return &ZFastTrie[Value]{
		size:           0,
		root:           nil,
		handle2NodeMap: make(map[string]*Node[Value]),
		EMPTY_VALUE:    emptyValue,
	}
}

func (t *ZFastTrie[Value]) InsertString(newText string, value Value) {
	t.Insert(NewBitString(newText), value)
}

func (t *ZFastTrie[Value]) Insert(newText BitString, value Value) {
	log.Printf("ZFastTrie::insert(%s, value: %v)", newText, value)
	if value == t.EMPTY_VALUE {
		log.Panic("insert value cannot be EMPTY_VALUE")
	}

	if t.size == 0 {
		t.root = NewNode(value, newText)
	} else {
		exitNode := t.getExitNode(newText)
		lcpLength := getLCPLength(exitNode.extent, newText)

		if lcpLength < exitNode.extent.size() {
			exitNodeExtent := exitNode.extent

			var newExtent BitString
			if lcpLength == 0 {
				newExtent = BitString("")
			} else {
				newExtent = exitNode.extent.substring(lcpLength)
			}

			if exitNode.isLeaf() {
				exitNode.setExtent(newExtent)
				t.insertHandle2NodeMap(exitNode)
			} else {
				newHandleLength := twoFattest(lcpLength, newExtent.size())
				isChangeHandle := (exitNode.handleLength() != newHandleLength)
				if isChangeHandle {
					t.eraseHandle2NodeMap(exitNode.handle())
					exitNode.setExtent(newExtent)
					t.insertHandle2NodeMap(exitNode)
				} else {
					exitNode.setExtent(newExtent)
				}
			}

			newNode := NewNodeWithNameLength(exitNode.value, exitNodeExtent, lcpLength+1)
			swapChildren(exitNode, newNode)
			exitNode.insertChild(newNode, lcpLength)

			if !newNode.isLeaf() {
				t.insertHandle2NodeMap(newNode)
			}

			if lcpLength < newText.size() {
				newTextNode := NewNodeWithNameLength(value, newText, lcpLength+1)
				exitNode.value = t.EMPTY_VALUE
				exitNode.insertChild(newTextNode, lcpLength)
			} else {
				exitNode.value = value
			}
		} else {
			if lcpLength == newText.size() {
				if exitNode.value == t.EMPTY_VALUE {
					exitNode.value = value
				} else {
					log.Println("warn: new text already exist.")
				}
			} else {
				if exitNode.isLeaf() {
					t.insertHandle2NodeMap(exitNode)
				}
				newTextNode := NewNodeWithNameLength(value, newText, lcpLength+1)
				exitNode.insertChild(newTextNode, lcpLength)
			}
		}
	}

	t.size++
}

func (t *ZFastTrie[Value]) EraseString(targetText string) {
	t.Erase(NewBitString(targetText))
}

func (t *ZFastTrie[Value]) Erase(targetText BitString) {
	log.Printf("ZFastTrie::erase %s", targetText.toString())
	if !t.Contains(targetText) {
		log.Println("warn: attempting to erase non-existent key")
		return
	}

	targetNode := t.getExitNode(targetText)
	if targetNode == nil {
		log.Println("warn: targetNode not found during erase, though Contains was true.")
		return
	}

	if t.size <= 1 {
		if t.root != nil {
			t.eraseHandle2NodeMap(t.root.handle())
		}
		t.root = nil
	} else {
		lcpLength := getLCPLength(targetNode.extent, targetText)
		if lcpLength != targetText.size() || targetText.size() > targetNode.extent.size() {
			log.Panic("erase: invariant violation")
		}

		if targetNode.isLeaf() {
			if targetNode == t.root {
				t.eraseHandle2NodeMap(targetNode.handle())
				t.root = nil
			} else {
				var parentNode *Node[Value]
				if targetNode.nameLength <= 1 {
					parentNode = t.root
				} else {
					parentNode = t.getExitNode(targetNode.extent.substring(targetNode.nameLength - 1))
				}

				parentNode.eraseChild(targetNode.key())
				t.eraseHandle2NodeMap(targetNode.handle())

				if parentNode.isLeaf() {
					t.eraseHandle2NodeMap(parentNode.handle())
				} else if parentNode.sizeChildren() == 1 && parentNode.value == t.EMPTY_VALUE {
					log.Println("swap parent and child node")
					t.eraseHandle2NodeMap(parentNode.handle())
					childNode := parentNode.getChild()

					parentNode.eraseChild(childNode.key())
					swapChildren(parentNode, childNode)

					parentNode.set(childNode.value, childNode.extent, parentNode.nameLength)

					if !parentNode.isLeaf() {
						t.eraseHandle2NodeMap(childNode.handle())
						t.insertHandle2NodeMap(parentNode)
					}
				}
			}
		} else if targetNode.sizeChildren() == 1 {
			childNode := targetNode.getChild()
			targetNode.eraseChild(childNode.key())
			t.eraseHandle2NodeMap(targetNode.handle())
			targetNode.set(childNode.value, childNode.extent, targetNode.nameLength)

			swapChildren(targetNode, childNode)
			if !targetNode.isLeaf() {
				t.eraseHandle2NodeMap(childNode.handle())
				t.insertHandle2NodeMap(targetNode)
			}
		} else {
			targetNode.value = t.EMPTY_VALUE
		}
	}
	t.size--
}

func (t *ZFastTrie[Value]) Update(targetText BitString, value Value) {
	exitNode := t.getExitNode(targetText)
	if exitNode == nil {
		log.Println("warn: ZFastTrie::update")
		log.Println("warn: Not Found key")
	} else {
		if exitNode.value == t.EMPTY_VALUE {
			log.Println("warn: ZFastTrie::update")
			log.Println("warn: Not Found key")
		} else {
			exitNode.value = value
		}
	}
}

func (t *ZFastTrie[Value]) insertHandle2NodeMap(node *Node[Value]) {
	if node.extent.size() == 0 {
		return
	}
	handle := node.handle()
	if handle.size() == 0 {
		return
	}
	handleStr := handle.toString()
	if _, ok := t.handle2NodeMap[handleStr]; ok {
		log.Panicf("handle %s already in map", handleStr)
	}
	t.handle2NodeMap[handleStr] = node
}

func (t *ZFastTrie[Value]) eraseHandle2NodeMap(handle BitString) {
	if handle.size() == 0 {
		return
	}
	delete(t.handle2NodeMap, handle.toString())
}

func (t *ZFastTrie[Value]) ContainsString(pattern string) bool {
	return t.Contains(NewBitString(pattern))
}

func (t *ZFastTrie[Value]) Contains(pattern BitString) bool {
	exitNode := t.getExitNode(pattern)
	if exitNode == nil {
		return false
	} else {
		return (exitNode.extent == pattern) && (exitNode.value != t.EMPTY_VALUE)
	}
}

func (t *ZFastTrie[Value]) ContainsPrefixString(pattern string) bool {
	return t.ContainsPrefix(NewBitString(pattern))
}

func (t *ZFastTrie[Value]) ContainsPrefix(pattern BitString) bool {
	node := t.getExitNode(pattern)
	return node != nil && isPrefix(node.extent, pattern)
}

func (t *ZFastTrie[Value]) getExitNode(pattern BitString) *Node[Value] {
	if t.root == nil {
		return nil
	}

	patternLength := pattern.size()
	a := 0
	b := patternLength
	var f int
	var node *Node[Value]
	result := t.root

	for 0 < (b - a) {
		f = twoFattest(a, b)
		node = t.getNode(pattern.substring(f))

		if node != nil {
			a = node.extent.size()
			result = node
		} else {
			b = f - 1
		}
	}

	if result != nil {
		lcpLength := getLCPLength(result.extent, pattern)
		if lcpLength == result.extent.size() && lcpLength < pattern.size() {
			var next *Node[Value]
			if pattern.at(lcpLength) {
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

func (t *ZFastTrie[Value]) getNode(handle BitString) *Node[Value] {
	node, ok := t.handle2NodeMap[handle.toString()]
	if !ok {
		return nil
	}
	return node
}

func swapChildren[Value comparable](a, b *Node[Value]) {
	a.leftChild, b.leftChild = b.leftChild, a.leftChild
	a.rightChild, b.rightChild = b.rightChild, a.rightChild
}

func (t *ZFastTrie[Value]) Print(indent string) {
	log.Printf("%sprint ZFastTrie", indent)
	for handle, node := range t.handle2NodeMap {
		log.Printf("%s    \"%s\"\t-> \"%s\", %d -> %d", indent, handle, node.extent.toString(), len(handle), node.extent.size())
	}
	if t.root != nil {
		t.root.print(indent)
	}
}

// Size returns the number of values stored in the trie.
func (t *ZFastTrie[Value]) Size() int {
	return t.size
}

// --- math/bits helper (for Go < 1.9 or clarity) ---
// This is built-in, but just to be clear what `bits.Len` does:
var _ = bits.Len

func main() {
	log.Println("--- Initializing ZFastTrie ---")

	// Инициализируем trie с типом Value = int и EMPTY_VALUE = -1
	trie := NewZFastTrie[int](-1)

	log.Println("--- Inserting values ---")
	trie.InsertString("101", 1)
	trie.InsertString("1011", 2)
	trie.InsertString("100", 3)
	trie.InsertString("0", 4)

	log.Println("--- Printing trie state after inserts ---")
	trie.Print("")

	log.Printf("Trie size: %d", trie.Size())

	log.Println("--- Checking 'Contains' ---")
	key1 := "1011"
	log.Printf("Contains('%s'): %t", key1, trie.ContainsString(key1))

	key2 := "111"
	log.Printf("Contains('%s'): %t", key2, trie.ContainsString(key2))

	log.Println("--- Erasing '101' ---")
	trie.EraseString("101")

	log.Println("--- Printing trie state after erase ---")
	trie.Print("")

	log.Println("--- Checking 'Contains' after erase ---")
	log.Printf("Contains('101'): %t", trie.ContainsString("101"))
	log.Printf("Contains('1011'): %t", trie.ContainsString("1011"))
	log.Printf("Trie size: %d", trie.Size())

	log.Println("--- Inserting value that creates internal node split ---")
	trie.InsertString("1010", 5)

	log.Println("--- Printing trie state after final insert ---")
	trie.Print("")
}
