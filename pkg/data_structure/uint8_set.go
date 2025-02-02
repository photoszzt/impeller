package data_structure

type Uint8Set map[uint8]struct{}

func NewUint8Set() Uint8Set {
	return make(Uint8Set)
}

func (s Uint8Set) Has(val uint8) bool {
	_, ok := s[val]
	return ok
}

func (s Uint8Set) Add(val uint8) {
	s[val] = struct{}{}
}

func (s Uint8Set) Remove(val uint8) {
	delete(s, val)
}
