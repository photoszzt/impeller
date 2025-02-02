package processor

/*
func byte_slice_increment(input []byte) ([]byte, error) {
	out := make([]byte, len(input))
	carry := 1
	for i := len(input) - 1; i >= 0; i-- {
		if input[i] == byte(0xFF) && carry == 1 {
			out[i] = byte(0x00)
		} else {
			out[i] = byte(input[i] + byte(carry))
			carry = 0
		}
	}
	if carry == 0 {
		return out, nil
	} else {
		return nil, xerrors.New("index out of bound")
	}
}
*/
