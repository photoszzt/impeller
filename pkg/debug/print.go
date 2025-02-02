package debug

import (
	"os"
)

func PrintByteSlice(data []byte) {
	for i := 0; i < len(data); i++ {
		Fprintf(os.Stderr, "%x ", data[i])
	}
	Fprintf(os.Stderr, "\n")
}
