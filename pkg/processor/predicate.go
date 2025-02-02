package processor

import "sharedlog-stream/pkg/optional"

var _ = (Predicate)(PredicateFunc(nil))

type PredicateFunc func(key interface{}, value interface{}) (bool, error)

type Predicate interface {
	Assert(key interface{}, value interface{}) (bool, error)
}

func (fn PredicateFunc) Assert(key interface{}, value interface{}) (bool, error) {
	return fn(key, value)
}

type PredicateG[K, V any] interface {
	Assert(key optional.Option[K], value optional.Option[V]) (bool, error)
}

type PredicateFuncG[K, V any] func(key optional.Option[K], value optional.Option[V]) (bool, error)

func (fn PredicateFuncG[K, V]) Assert(key optional.Option[K], value optional.Option[V]) (bool, error) {
	return fn(key, value)
}
