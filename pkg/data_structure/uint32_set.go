package data_structure

type Uint32Set map[uint32]struct{}

func NewUint32Set() Uint32Set {
	return make(Uint32Set)
}

func NewUint32SetSized(size int) Uint32Set {
	return make(Uint32Set, size)
}

func (s Uint32Set) Has(val uint32) bool {
	_, ok := s[val]
	return ok
}

func (s Uint32Set) Add(val uint32) {
	s[val] = struct{}{}
}

func (s Uint32Set) Remove(val uint32) {
	delete(s, val)
}
