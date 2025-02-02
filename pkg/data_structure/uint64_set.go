package data_structure

type Uint64Set map[uint64]struct{}

func NewUint64Set() Uint64Set {
	return make(Uint64Set)
}

func (s Uint64Set) Has(val uint64) bool {
	_, ok := s[val]
	return ok
}

func (s Uint64Set) Add(val uint64) {
	s[val] = struct{}{}
}

func (s Uint64Set) Remove(val uint64) {
	delete(s, val)
}
