package lfu

import (
	"errors"
	"iter"
	"lfucache/internal/linkedlist"
)

var ErrKeyNotFound = errors.New("key not found")

const DefaultCapacity = 5

// Cache
// O(capacity) memory
type Cache[K comparable, V any] interface {
	// Get returns the value of the key if the key exists in the cache,
	// otherwise, returns ErrKeyNotFound.
	//
	// O(1)
	Get(key K) (V, error)

	// Put updates the value of the key if present, or inserts the key if not already present.
	//
	// When the cache reaches its capacity, it should invalidate and remove the least frequently used key
	// before inserting a new item. For this problem, when there is a tie
	// (i.e., two or more keys with the same frequency), the least recently used key would be invalidated.
	//
	// O(1)
	Put(key K, value V)

	// All returns the iterator in descending order of frequency.
	// If two or more keys have the same frequency, the most recently used key will be listed first.
	//
	// O(capacity)
	All() iter.Seq2[K, V]

	// Size returns the cache size.
	//
	// O(1)
	Size() int

	// Capacity returns the cache capacity.
	//
	// O(1)
	Capacity() int

	// GetKeyFrequency returns the element's frequency if the key exists in the cache,
	// otherwise, returns ErrKeyNotFound.
	//
	// O(1)
	GetKeyFrequency(key K) (int, error)
}

// cacheImpl represents LFU cache implementation
type cacheImpl[K comparable, V any] struct {
	frequencyClasses internal.LinkedList[*classFrequency[K, V]] // linkedlist where every node is linked list of keys that have the same frequency of use
	keyToElements    map[K]*internal.Node[valOfKey[K, V]]       // Map from key to node with value
	capacity         int
	size             int
}

// Auxiliary structure that stores keys in the form of a list in the order of their use history
// and the frequency of the stored keys
type classFrequency[K comparable, V any] struct {
	lst       internal.LinkedList[valOfKey[K, V]]
	frequency int
}

// Factory of classFrequency, which initializes new class by values
func newClassFrequency[K comparable, V any](frequency int) *classFrequency[K, V] {
	cls := new(classFrequency[K, V])
	cls.lst = internal.NewLinkedList[valOfKey[K, V]]()
	cls.frequency = frequency
	return cls
}

// Structure that is contained by the node. Structure contains key, value and pointer to node of frequencyClasses
// in cacheImpl
type valOfKey[K comparable, V any] struct {
	key           K
	val           V
	nodeFreqClass *internal.Node[*classFrequency[K, V]] // is needed to have opportunity of accessing to nodes of frequencyClasses
}

// New initializes the cache with the given capacity.
// If no capacity is provided, the cache will use DefaultCapacity.
func New[K comparable, V any](capacity ...int) *cacheImpl[K, V] {
	newCap := DefaultCapacity
	if len(capacity) > 0 { // if capacity was given
		newCap = capacity[0] // reads the value
		if newCap < 0 {      // if capacity is incorrect, New panics
			panic("The capacity must be greater than zero")
		}
	}
	classes := internal.NewLinkedList[*classFrequency[K, V]]() // Creates frequencyClasses
	classes.PushBack(newClassFrequency[K, V](1))               // and pushes in it default frequency class with frequency 1

	return &cacheImpl[K, V]{classes, make(map[K]*internal.Node[valOfKey[K, V]], newCap), newCap, 0}
}

// Function increases frequency of key in valNode and moves valNode to frequency class of new frequency
// or if there is no this class, create it, and moves valNode in its
func (l *cacheImpl[K, V]) increaseFreqOfKey(keyNode *internal.Node[valOfKey[K, V]]) {
	// reads values from pointer of Node to won't have long access to memory in method
	curFreqNode := keyNode.Data.nodeFreqClass // node of frequencyClasses containing keyNode
	curClass := curFreqNode.Data              // classFrequency of curFreqNode
	curLst := curClass.lst                    // linkedList of keys with one frequency
	curFraq := curClass.frequency             // current frequency

	if nextClass := curFreqNode.Prev(l.frequencyClasses); nextClass != nil && nextClass.Data.frequency == curFraq+1 { // if there is frequency class of new frequency (nextClass)
		nextClass.Data.lst.MoveToFront(keyNode, curLst) // moves keyNode to this nextClass
		keyNode.Data.nodeFreqClass = nextClass
	} else if curLst.Size() == 1 { // if nextClass doesn't exist, but in curClass there is only keyNode
		curClass.frequency++ // changes frequency of curClass
	} else { // if nextClass doesn't exist and in curClass there are another nodes
		newClass := newClassFrequency[K, V](curFraq + 1) // creates new class
		newClass.lst.MoveToFront(keyNode, curLst)        // moves keyNode to new class
		keyNode.Data.nodeFreqClass = l.frequencyClasses.PushBefore(curFreqNode, newClass)
	}
	if curLst.Size() == 0 { // if after moving node its last class (curClass) has become empty
		// remove curClass
		l.frequencyClasses.Remove(curFreqNode)
	}
}

func (l *cacheImpl[K, V]) Get(key K) (V, error) {
	valueOfKey, ok := l.keyToElements[key] // attempt to read key from keyToElements
	if !ok {                               // if there isn't given key
		var zeroVal V
		return zeroVal, ErrKeyNotFound // returns error
	}
	l.increaseFreqOfKey(valueOfKey) // increases frequency of this key
	return valueOfKey.Data.val, nil
}

func (l *cacheImpl[K, V]) Put(key K, value V) {
	if valNode, ok := l.keyToElements[key]; ok { // if already there is given key in cacheImpl
		valNode.Data.val = value     // changes value of key
		l.increaseFreqOfKey(valNode) // increases frequency of key
		return
	}
	// reads values from pointer of Node, to won't have long access to memory in method
	freqClasses := l.frequencyClasses
	leastFreqClass := freqClasses.Back().Data
	leastFreqLst := leastFreqClass.lst

	if l.Size() == l.Capacity() { // if cache is filled
		// removes key with least frequency and the oldest time of using
		delete(l.keyToElements, leastFreqLst.PopBack().Data.key) // from leastFreqClass and map
	} else {
		l.size++ // increments size
	}
	if leastFreqClass.frequency > 1 { // if class of least frequency (leastFreqClass) has frequency bigger then 1
		if leastFreqLst.Size() > 0 { // if leastFreqClass is not empty, a new one is created
			leastFreqClass = newClassFrequency[K, V](1)
			freqClasses.PushBack(leastFreqClass)
		} else { // if this class is empty
			leastFreqClass.frequency = 1 // reusing this class with changing its frequency
		}
	}
	// adds this key in map and in leastFreqClass
	l.keyToElements[key] = leastFreqClass.lst.PushFront(valOfKey[K, V]{key, value, freqClasses.Back()})
}

func (l *cacheImpl[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for class := range l.frequencyClasses.All() { // iterates by classes of frequency
			for el := range class.lst.All() { // iterates by keys in current class of frequency
				if !yield(el.key, el.val) { // checks that user wants next value
					return
				}
			}
		}
	}
}

func (l *cacheImpl[K, V]) Size() int {
	return l.size
}

func (l *cacheImpl[K, V]) Capacity() int {
	return l.capacity
}

func (l *cacheImpl[K, V]) GetKeyFrequency(key K) (int, error) {
	val, ok := l.keyToElements[key] // attempt to read key from keyToElements
	if !ok {                        // if there isn't given key
		return 0, ErrKeyNotFound // returns error
	}
	return val.Data.nodeFreqClass.Data.frequency, nil
}
