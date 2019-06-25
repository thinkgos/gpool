package gpool

import (
	"time"
)

type item struct {
	task TaskFunc
	arg  interface{}
}

// work is an element of a linked list.
type work struct {
	// fot list
	next, prev *work
	list       *list

	// The value stored
	// pool who owns this worker.
	pool *Pool
	// itm is a time with task and it's arg should be done.
	itm chan item
	// markTime mark when coroutine begin in idle
	markTime time.Time
}

// Next returns the next list element or nil.
func (this *work) Next() *work {
	if p := this.next; this.list != nil && p != &this.list.root {
		return p
	}
	return nil
}

// Prev returns the previous list element or nil.
func (this *work) Prev() *work {
	if p := this.prev; this.list != nil && p != &this.list.root {
		return p
	}
	return nil
}

// list represents a doubly linked list.
type list struct {
	root   work // sentinel list element, only &root, root.prev, and root.next are used
	length int  // current list length excluding (this) sentinel element
}

// newList returns an initialized list.
func newList() *list {
	l := &list{length: 0}
	l.root.next = &l.root
	l.root.prev = &l.root
	return l
}

// len returns the number of elements of list l.
// The complexity is O(1).
func (l *list) Len() int {
	return l.length
}

// Front returns the first element of list l or nil if the list is empty.
func (l *list) Front() *work {
	if l.length == 0 {
		return nil
	}
	return l.root.next
}

// Back returns the last element of list l or nil if the list is empty.
func (l *list) Back() *work {
	if l.length == 0 {
		return nil
	}
	return l.root.prev
}

// insert inserts this after at, increments l.length, and returns this.
func (l *list) insert(this, at *work) *work {
	n := at.next
	at.next = this
	this.prev = at
	this.next = n
	n.prev = this
	this.list = l
	l.length++
	return this
}

// remove removes this from its list, decrements l.length, and returns this.
func (l *list) remove(this *work) *work {
	this.prev.next = this.next
	this.next.prev = this.prev
	this.next = nil // avoid memory leaks
	this.prev = nil // avoid memory leaks
	this.list = nil
	l.length--
	return this
}

// Remove removes this from l if this is an element of list l.
// The element must not be nil.
func (l *list) Remove(this *work) *work {
	if this.list == l {
		// if this.list == l, l must have been initialized when this was inserted
		// in l or l == nil (this is a zero Element) and l.remove will crash
		l.remove(this)
	}
	return this
}

// PushFront inserts a new element at the front of list l and returns this.
func (l *list) PushFront(this *work) *work {
	return l.insert(this, &l.root)
}

// PushBack inserts a new element at the back of list l and returns this.
func (l *list) PushBack(this *work) *work {
	return l.insert(this, l.root.prev)
}
