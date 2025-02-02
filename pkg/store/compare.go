package store

import (
	"strings"

	"golang.org/x/exp/constraints"
)

type CompareFunc func(lhs, rhs interface{}) int

func (f CompareFunc) Compare(lhs, rhs interface{}) int {
	return f(lhs, rhs)
}

type CompareFuncG[K any] func(lhs, rhs K) int

func Int64IntrCompare(l, r interface{}) int {
	return IntegerCompare(l.(int64), r.(int64))
}

func Uint64IntrCompare(lhs, rhs interface{}) int {
	l := lhs.(uint64)
	r := rhs.(uint64)
	return IntegerCompare(l, r)
}

func Uint32IntrCompare(lhs, rhs interface{}) int {
	l := lhs.(uint32)
	r := rhs.(uint32)
	return IntegerCompare(l, r)
}

func IntIntrCompare(lhs, rhs interface{}) int {
	l := lhs.(int)
	r := rhs.(int)
	return IntegerCompare(l, r)
}

func IntegerCompare[K constraints.Integer](l, r K) int {
	if l < r {
		return -1
	} else if l == r {
		return 0
	} else {
		return 1
	}
}

func StringCompare(lhs, rhs string) int {
	return strings.Compare(lhs, rhs)
}

func CompareWithVersionedKey[K any](lv, rv VersionedKeyG[K], baseCompare CompareFuncG[K]) int {
	lvk := lv.Key
	rvk := rv.Key
	kCompare := baseCompare(lvk, rvk)
	if kCompare == 0 {
		if lv.Version < rv.Version {
			return -1
		} else if lv.Version == rv.Version {
			return 0
		} else {
			return 1
		}
	} else {
		return kCompare
	}
}
