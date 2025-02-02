package data_structure

type Uint16Set map[uint16]struct{}

func NewUint16Set() Uint16Set {
	return make(Uint16Set)
}

func NewUint16SetSized(size int) Uint16Set {
	return make(Uint16Set, size)
}

func (s Uint16Set) Has(val uint16) bool {
	_, ok := s[val]
	return ok
}

func (s Uint16Set) Add(val uint16) {
	s[val] = struct{}{}
}

func (s Uint16Set) Remove(val uint16) {
	delete(s, val)
}
