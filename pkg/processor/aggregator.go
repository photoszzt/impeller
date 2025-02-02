package processor

import "sharedlog-stream/pkg/optional"

type AggregatorG[K, V, VA any] interface {
	Apply(key K, value V, aggregate optional.Option[VA]) optional.Option[VA]
}

type AggregatorFunc func(key interface{}, value interface{}, aggregate interface{}) interface{}

func (fn AggregatorFunc) Apply(key interface{}, value interface{}, aggregate interface{}) interface{} {
	return fn(key, value, aggregate)
}

type AggregatorFuncG[K, V, VA any] func(key K, value V, agg optional.Option[VA]) optional.Option[VA]

func (fn AggregatorFuncG[K, V, VA]) Apply(key K, value V, aggregate optional.Option[VA]) optional.Option[VA] {
	return fn(key, value, aggregate)
}

type Initializer interface {
	Apply() interface{}
}

type InitializerFunc func() interface{}

func (fn InitializerFunc) Apply() interface{} {
	return fn()
}

type InitializerG[VA any] interface {
	Apply() optional.Option[VA]
}

type InitializerFuncG[VA any] func() optional.Option[VA]

func (fn InitializerFuncG[VA]) Apply() optional.Option[VA] {
	return fn()
}
