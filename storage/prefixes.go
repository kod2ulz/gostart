package storage

import (
	"strings"
	"sync"

	"github.com/kod2ulz/gostart/collections"
)

type trieNode[K comparable, T any] struct {
	children collections.Map[K, *trieNode[K, T]]
	value    T
}

func (n *trieNode[K, T]) add(key K, value T) (ok bool) {
	if n.children == nil {
		n.children = collections.Map[K, *trieNode[K, T]]{}
	}
	if _, ok = n.children[key]; !ok {
		n.children[key] = &trieNode[K, T]{children: collections.Map[K, *trieNode[K, T]]{}}
	}
	n.children[key].value = value
	return
}

type Prefixes[T any] struct {
	mx        sync.RWMutex
	delimiter string
	root      *trieNode[string, T]
}

func NewPrefixMapper[T any](delimiter string, _default T) Prefixes[T] {
	return Prefixes[T]{
		delimiter: delimiter,
		root: &trieNode[string, T]{
			value:    _default,
			children: collections.Map[string, *trieNode[string, T]]{},
		},
	}
}

func (ts *Prefixes[T]) Set(key string, value T) (ok bool) {
	ts.mx.Lock()
	defer ts.mx.Unlock()
	var node = ts.root
	if !strings.Contains(key, ts.delimiter) {
		return node.add(key, value)
	}
	var namespace = strings.Split(key, ts.delimiter)
	var last = len(namespace) - 1
	for i, k := range namespace {
		if node.children != nil && node.children[k] != nil {
			node = node.children[k]
			continue
		}
		switch i {
		case 0:
			node.add(k, node.value)
		case last:
			ok = node.add(k, value)
			return
		default:
			node.add(k, node.value)
		}
		node = node.children[k]
	}
	return
}

func (ts *Prefixes[T]) Get(key string) (out T) {
	ts.mx.RLock()
	defer ts.mx.RUnlock()
	var node = ts.root
	if !strings.Contains(key, ts.delimiter) {
		if _, ok := node.children[key]; !ok {
			return node.value
		}
		return node.children[key].value
	}
	var namespace = strings.Split(key, ts.delimiter)
	var last = len(namespace) - 1
	for i, k := range namespace {
		if n, ok := node.children[k]; !ok {
			break
		} else if i == last {
			return n.value
		}
		node = node.children[k]
	}
	return node.value
}
