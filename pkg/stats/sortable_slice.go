package stats

import (
	"sort"

	"golang.org/x/exp/constraints"
)

type SortableSlice interface {
	sort.Interface
	At(i int) interface{}
}

func P(t SortableSlice, percent float64) interface{} {
	return t.At(int(float64(t.Len())*percent+0.5) - 1)
}

func POf[E constraints.Ordered](t []E, percent float64) E {
	return t[(int(float64(len(t))*percent+0.5) - 1)]
}

type IntSlice []int

func (t IntSlice) Len() int {
	return len(t)
}

func (t IntSlice) Less(i, j int) bool {
	return t[i] < t[j]
}

func (t IntSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t IntSlice) At(i int) interface{} {
	return t[i]
}

type Int32Slice []int32

func (t Int32Slice) Len() int {
	return len(t)
}

func (t Int32Slice) Less(i, j int) bool {
	return t[i] < t[j]
}

func (t Int32Slice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Int32Slice) At(i int) interface{} {
	return t[i]
}

type Int64Slice []int64

func (t Int64Slice) Len() int {
	return len(t)
}

func (t Int64Slice) Less(i, j int) bool {
	return t[i] < t[j]
}

func (t Int64Slice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Int64Slice) At(i int) interface{} {
	return t[i]
}

type UintSlice []uint

func (t UintSlice) Len() int {
	return len(t)
}

func (t UintSlice) Less(i, j int) bool {
	return t[i] < t[j]
}

func (t UintSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t UintSlice) At(i int) interface{} {
	return t[i]
}

type Uint32Slice []uint32

func (t Uint32Slice) Len() int {
	return len(t)
}

func (t Uint32Slice) Less(i, j int) bool {
	return t[i] < t[j]
}

func (t Uint32Slice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Uint32Slice) At(i int) interface{} {
	return t[i]
}

type Uint64Slice []int64

func (t Uint64Slice) Len() int {
	return len(t)
}

func (t Uint64Slice) Less(i, j int) bool {
	return t[i] < t[j]
}

func (t Uint64Slice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Uint64Slice) At(i int) interface{} {
	return t[i]
}
