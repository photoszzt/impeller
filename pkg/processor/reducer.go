package processor

type Reducer interface {
	Apply(value1 interface{}, value2 interface{}) interface{}
}

type ReducerFunc func(value1 interface{}, value2 interface{}) interface{}

func (fn ReducerFunc) Apply(value1 interface{}, value2 interface{}) interface{} {
	return fn(value1, value2)
}

type ReducerG[V any] interface {
	Apply(value1 V, value2 V) V
}

type ReducerFuncG[V any] func(value1 V, value2 V) V

func (fn ReducerFuncG[V]) Apply(value1 V, value2 V) V {
	return fn(value1, value2)
}
