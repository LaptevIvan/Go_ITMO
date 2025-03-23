package internal

import (
	"iter"
)

type LinkedList[T any] interface {

	// Adds value to end of list.
	PushBack(value T) *Node[T]

	// Adds value to start of list.
	PushFront(value T) *Node[T]

	// Moves node from listNode to this list after head without memmory allocation.
	// This list can be ListNode.
	// If given node doesn't belong listNode, behaviour of listNode and its real list is undefined.
	// Given node and listNode must not be nil.
	MoveToFront(node *Node[T], listNode LinkedList[T])

	// Adds value before given node and returns new node containings given value,
	// If given node doesn't belong this list, behaviour of its real list is undefined.
	// Given node must not be nil.
	PushBefore(node *Node[T], value T) *Node[T]

	// Removes given node from this list.
	// If given node doesn't belong this list, behaviour of this and its real list is undefined.
	// Given node must not be nil.
	Remove(node *Node[T])

	// Removes value from end of list and returns deleted node or nil, if list is empty.
	PopBack() *Node[T]

	// Returns count of value in list.
	Size() int

	// Returns first node of list or nil, if list is empty.
	Front() *Node[T]

	// Returns last node of list or nil, if list is empty.
	Back() *Node[T]

	// Returns iterator, which goes from Front() to Back() of list.
	All() iter.Seq[T]
}

// Node - element of LinkedList
type Node[T any] struct {
	prev *Node[T] // pointer to previous node
	next *Node[T] // pointer to next node
	Data T        // Data witch node contains
}

// Factory of nodes. Returns a node containing the specified data, with prev and next pointing to it
func NewNode[T any](data T) *Node[T] {
	ans := &Node[T]{Data: data}
	ans.prev = ans
	ans.next = ans
	return ans
}

// Returns previous node of current node or nil if node is first in listNode
// If node doesn't belong listNode, behaviour of real its list is undefined.
func (curNode *Node[T]) Prev(listNode LinkedList[T]) *Node[T] {
	if listNode.Front() != curNode { // if previous node is not head of this list
		return curNode.prev // returns previous node
	}
	return nil
}

// Returns next node of current node or nil if node is last in listNode
// If node doesn't belong listNode, behaviour of real its list is undefined.
func (curNode *Node[T]) Next(listNode LinkedList[T]) *Node[T] {
	if listNode.Back() != curNode { // if next node is not head of this list
		return curNode.next // returns previous node
	}
	return nil
}

// Connects given nodes together so that, node1 is previous node of node2 and node2 is next node of node1
func connectNodes[T any](node1 *Node[T], node2 *Node[T]) {
	if node1 != nil { // checking that given pointer of node points to really node
		node1.next = node2
	}
	if node2 != nil {
		node2.prev = node1
	}
}

// Realization of interface linkedlist. Сircular list
type linkedListImpl[T any] struct {
	head *Node[T] // pointer to first technical node
	size int      // count of nodes in list without head
}

// Factory of linkedlists. Returns the empty linkedListImpl with initialized head
func NewLinkedList[T any]() *linkedListImpl[T] {
	var zeroVal T
	lnkList := &linkedListImpl[T]{NewNode(zeroVal), 0}
	return lnkList
}

func (lst *linkedListImpl[T]) PushBefore(beforeNode *Node[T], value T) *Node[T] {
	newNode := NewNode(value)
	connectNodes(beforeNode.prev, newNode)
	connectNodes(newNode, beforeNode)
	lst.size++

	return newNode
}

func (lst *linkedListImpl[T]) PushBack(value T) *Node[T] {
	return lst.PushBefore(lst.head, value) // reusing PushBefore due to the fact that the list is circular
}

func (lst *linkedListImpl[T]) PushFront(value T) *Node[T] {
	return lst.PushBefore(lst.head.next, value) // reusing PushBefore due to the fact that the list is circular
}

func (lst *linkedListImpl[T]) MoveToFront(node *Node[T], listNode LinkedList[T]) {
	head := lst.head
	headNext := head.next
	if headNext == node { // if given node is already been after head
		return // There's nothing to do
	}

	listNode.Remove(node)    // removes node from its list
	connectNodes(head, node) // connects node with head and node after head
	connectNodes(node, headNext)
	lst.size++
}

func (lst *linkedListImpl[T]) Remove(node *Node[T]) {
	connectNodes(node.prev, node.next) // connects the neighbours nodes of given node
	lst.size--
}

func (lst *linkedListImpl[T]) PopBack() *Node[T] {
	if lst.Size() == 0 { // if there isn't node, which can be removed
		return nil // returns nil
	}
	remBack := lst.Back() // takes node for removing
	lst.Remove(remBack)
	return remBack // returns removed node
}

func (lst *linkedListImpl[T]) Size() int {
	return lst.size
}

func (lst *linkedListImpl[T]) Front() *Node[T] {
	if lst.Size() > 0 { // if list contains any nodes
		return lst.head.next // returns node after head
	}
	return nil
}

func (lst *linkedListImpl[T]) Back() *Node[T] {
	if lst.Size() > 0 { // if list contains any nodes
		return lst.head.prev // returns node before head (due to the circular sheet, the last node)
	}
	return nil
}

func (lst *linkedListImpl[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		cur := lst.head // head as starting node
		n := lst.Size()
		for range n { // n times
			cur = cur.Next(lst)   // goes to next node
			if !yield(cur.Data) { // checks that user wants next value
				return
			}
		}
	}
}
