package linkedhashset

import "sharedlog-stream/pkg/data_structure/genericlist"

type LinkedHashSet[T comparable] struct {
	table    map[T]struct{}
	ordering *genericlist.List[T]
}

func New[T comparable]() *LinkedHashSet[T] {
	s := &LinkedHashSet[T]{
		table:    make(map[T]struct{}),
		ordering: genericlist.New[T](),
	}
	return s
}

func (s *LinkedHashSet[T]) Add(v T) {
	if _, ok := s.table[v]; !ok {
		s.table[v] = struct{}{}
		s.ordering.PushBack(v)
	}
}

func (s *LinkedHashSet[T]) Remove(v T) {
	if _, ok := s.table[v]; ok {
		delete(s.table, v)
		s.ordering.RemoveValue(v, func(a, b T) bool {
			return a == b
		})
	}
}

func (s *LinkedHashSet[T]) Contains(v T) bool {
	_, ok := s.table[v]
	return ok
}

func (s *LinkedHashSet[T]) Len() int {
	return len(s.table)
}

func (s *LinkedHashSet[T]) Clear() {
	s.table = make(map[T]struct{})
	s.ordering.Init()
}

func (s *LinkedHashSet[T]) Copy() *LinkedHashSet[T] {
	sCpy := New[T]()
	for k := range s.table {
		sCpy.Add(k)
	}
	return sCpy
}

func (s *LinkedHashSet[T]) IterateCb(cb func(v T) bool) {
	for e := s.ordering.Front(); e != nil; e = e.Next() {
		shouldIter := cb(e.Value)
		if !shouldIter {
			return
		}
	}
}
