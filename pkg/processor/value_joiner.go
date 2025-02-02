package processor

import "sharedlog-stream/pkg/optional"

/*
type ValueJoiner interface {
	Apply(value1 interface{}, value2 interface{}) interface{}
}

type ValueJoinerFunc func(value1 interface{}, value2 interface{}) interface{}

func (fn ValueJoinerFunc) Apply(value1 interface{}, value2 interface{}) interface{} {
	return fn(value1, value2)
}
*/

type ValueJoinerG[V1, V2, VR any] interface {
	Apply(value1 V1, value2 V2) VR
}

type ValueJoinerFuncG[V1, V2, VR any] func(value1 V1, value2 V2) VR

func (fn ValueJoinerFuncG[V1, V2, VR]) Apply(value1 V1, value2 V2) VR {
	return fn(value1, value2)
}

/*
type ValueJoinerWithKey interface {
	Apply(readOnlyKey interface{}, value1 interface{}, value2 interface{}) interface{}
}

type ValueJoinerWithKeyFunc func(readOnlyKey interface{}, value1 interface{}, value2 interface{}) interface{}

func (fn ValueJoinerWithKeyFunc) Apply(readOnlyKey interface{}, value1 interface{}, value2 interface{}) interface{} {
	return fn(readOnlyKey, value1, value2)
}
*/

type ValueJoinerWithKeyG[K, V1, V2, VR any] interface {
	Apply(readOnlyKey K, value1 V1, value2 V2) optional.Option[VR]
}

type ValueJoinerWithKeyFuncG[K, V1, V2, VR any] func(readOnlyKey K, value1 V1, value2 V2) optional.Option[VR]

func (fn ValueJoinerWithKeyFuncG[K, V1, V2, VR]) Apply(readOnlyKey K, value1 V1, value2 V2) optional.Option[VR] {
	return fn(readOnlyKey, value1, value2)
}

// func ReverseValueJoinerWithKey(f ValueJoinerWithKeyFunc) ValueJoinerWithKeyFunc {
// 	return func(readOnlyKey, value1, value2 interface{}) interface{} {
// 		return f(readOnlyKey, value2, value1)
// 	}
// }

func ReverseValueJoinerWithKeyG[K, V1, V2, VR any](f ValueJoinerWithKeyFuncG[K, V1, V2, VR]) ValueJoinerWithKeyFuncG[K, V2, V1, VR] {
	return func(readOnlyKey K, value2 V2, value1 V1) optional.Option[VR] {
		return f(readOnlyKey, value1, value2)
	}
}

/*
type ValueJoinerWithKeyTs interface {
	Apply(readOnlyKey interface{}, value1 interface{}, value2 interface{},
		leftTs int64, otherTs int64) interface{}
}

type ValueJoinerWithKeyTsFunc func(readOnlyKey interface{}, value1 interface{}, value2 interface{},
	leftTs int64, otherTs int64) interface{}

func (fn ValueJoinerWithKeyTsFunc) Apply(readOnlyKey interface{}, value1 interface{}, value2 interface{},
	leftTs int64, otherTs int64,
) interface{} {
	return fn(readOnlyKey, value1, value2, leftTs, otherTs)
}
*/

type ValueJoinerWithKeyTsG[K, V1, V2, VR any] interface {
	Apply(readOnlyKey K, value1 V1, value2 V2, leftTs int64, otherTs int64) optional.Option[VR]
}

type ValueJoinerWithKeyTsFuncG[K, V1, V2, VR any] func(readOnlyKey K, value1 V1, value2 V2,
	leftTs int64, otherTs int64) optional.Option[VR]

func (fn ValueJoinerWithKeyTsFuncG[K, V1, V2, VR]) Apply(readOnlyKey K, value1 V1, value2 V2,
	leftTs int64, otherTs int64,
) optional.Option[VR] {
	return fn(readOnlyKey, value1, value2, leftTs, otherTs)
}

func ReverseValueJoinerWithKeyTsG[K, V1, V2, VR any](f ValueJoinerWithKeyTsFuncG[K, V1, V2, VR]) ValueJoinerWithKeyTsFuncG[K, V2, V1, VR] {
	return func(readOnlyKey K, value2 V2, value1 V1, leftTs, otherTs int64) optional.Option[VR] {
		return f(readOnlyKey, value1, value2, otherTs, leftTs)
	}
}

// func ReverseValueJoinerWithKeyTs(f ValueJoinerWithKeyTsFunc) ValueJoinerWithKeyTsFunc {
// 	return func(readOnlyKey, value1, value2 interface{}, leftTs, otherTs int64) interface{} {
// 		return f(readOnlyKey, value2, value1, otherTs, leftTs)
// 	}
// }
