//go:build debug
// +build debug

package debug

import (
	"fmt"
	"io"
)

// Assert will panic with msg if cond is false.
//
// msg must be a string, func() string or fmt.Stringer.
func Assert(cond bool, msg interface{}) {
	if !cond {
		panic(getStringValue(msg))
	}
}

func Fprintf(w io.Writer, format string, a ...interface{}) {
	fmt.Fprintf(w, format, a...)
}

func Fprint(w io.Writer, s string) {
	fmt.Fprint(w, s)
}
