package conc_bst

import "sync"

type Comparable[T any] interface {
	Compare(other T) int
}

type ConcAvlTree[K Comparable[K], V any] struct {
	// sentinel nodes per [DVY18] section 5
	// rationale is to have all snapshots be of the same length, for simplicity.
	// For AVL, that means instead of leftSnapshot size being 0, or 1, its always 1.
	leftSentinel  *avlNode[K, V]
	rightSentinel *avlNode[K, V]
	root          *avlNode[K, V]
}

type avlNode[K Comparable[K], V any] struct {
	key   K
	value V

	// height = 1 + number of layers underneath this node. So a leaf has 1
	height uint8
	marked bool

	// TODO: consider benchmarking removing parent.
	// Saves us RAM, we have to complicate code for rebalance.
	parent *avlNode[K, V]
	left   *avlNode[K, V]
	right  *avlNode[K, V]

	// Needed to validate the PaVT condition
	// using the BST optimization of just linking to other nodes directly,
	// due to snapshots being length 1 & always present
	// TODO: Evaluate is this actually better than just having another copy of the key.
	leftSnapshot  *avlNode[K, V]
	rightSnapshot *avlNode[K, V]

	mtx sync.Mutex
}

func NewConcAvlTree[K Comparable[K], V any](min K, max K, compareFn func(K, K) int) ConcAvlTree[K, V] {
	leftSentinel := &avlNode[K, V]{key: min}
	rightSentinel := &avlNode[K, V]{key: max}
	// make leftSentinel the root
	leftSentinel.right = rightSentinel
	rightSentinel.parent = leftSentinel
	leftSentinel.leftSnapshot = rightSentinel
	rightSentinel.rightSnapshot = leftSentinel

	return ConcAvlTree[K, V]{
		leftSentinel:  leftSentinel,
		rightSentinel: rightSentinel,
		root:          leftSentinel,
	}
}

func newNode[K Comparable[K], V any](key K, value V, parent *avlNode[K, V]) *avlNode[K, V] {
	return &avlNode[K, V]{key: key, value: value, parent: parent}
}

func (t *ConcAvlTree[K, V]) Contains(key K) bool {
	_, found := t.Get(key)
	return found
}

func invalidSnapshot[K Comparable[K], V any](
	lastNode *avlNode[K, V], key K, compRes int) bool {
	return (compRes < 0 && lastNode.leftSnapshot.key.Compare(key) <= 0) ||
		(compRes > 0 && lastNode.rightSnapshot.key.Compare(key) >= 0)
}

func (t *ConcAvlTree[K, V]) Get(key K) (value V, found bool) {
	// [DVY18] PaVT-Traverse_T(k)
	// this explicitly does not require any synchronization primitives,
	// due to the design of it.
GET_TAILCALLSTART:
	lastNode, compRes := t.searchForKey(key)
	if compRes == 0 {
		return lastNode.value, true
	}
	// if it wasn't in the tree, we need to ensure we didn't get re-organized out.
	// Recall that the goal of [DVY18] was that this can be constructed with no locks!
	if lastNode.marked || invalidSnapshot(lastNode, key, compRes) {
		// WE use a goto here because golang doesn't have tail-call recursion
		goto GET_TAILCALLSTART
	}
	// value is set to unitialized value
	return value, false
}

// searchForKey looks for a node
// this explicitly does not require any synchronization primitives,
// due to the design of [DVY18]. Subcomponent of PaVT-traverse
func (t *ConcAvlTree[K, V]) searchForKey(key K) (lastNode *avlNode[K, V], compRes int) {
	curNode := t.root
	// child is assigned a value in first loop iteration
	var child *avlNode[K, V]
	compRes = -2
	for compRes != 0 {
		if compRes < 0 {
			child = curNode.left
		} else {
			child = curNode.right
		}
		// if nil, key not found, curNode was the last node,
		// which has `compRes` relation to `key`
		if child == nil {
			break
		}
		curNode = child
		compRes = curNode.key.Compare(key)
	}
	return curNode, compRes
}

// Updates the tree, to have K.
// returns if the key already existed in the tree.
func (t *ConcAvlTree[K, V]) Insert(key K, value V) bool {
INSERT_TAILCALLSTART:
	node, compRes := t.searchForKey(key)
	// compRes = 0, implies we update the value at an existing node in tree.
	if compRes == 0 {
		node.value = value
		return true
	}

	// we insert this as a child for node
	retry := node.addChild(key, value, compRes)
	if retry {
		goto INSERT_TAILCALLSTART
	}
	return false
}

func (n *avlNode[K, V]) addChild(key K, value V, compRes int) (retry bool) {
	n.mtx.Lock()
	// We need to check this under lock. We check that:
	// * Has this node been marked is the true check. (AKA deleted)
	// * No child has already been added where were trying to add it
	// * This node is snapshot invalidated
	// in single threaded mode, this check is unnecessary.
	if n.marked ||
		(n.left != nil && compRes < 0) || (n.right != nil && compRes > 0) ||
		invalidSnapshot(n, key, compRes) {
		n.mtx.Unlock()
		return true
	}
	newChild := newNode(key, value, n)

	n.mtx.Unlock()
	return false
}

type iterator[K Comparable[K], V any] struct {
	t         *ConcAvlTree[K, V]
	ascending bool
	nodeList  []*avlNode[K, V]
}

func (t *ConcAvlTree[K, V]) Iter(k1 K, k2 K) iterator[K, V] {
	return iterator[K, V]{t, true, nil}
}
