//go:build !debug
// +build !debug

package debug

import "io"

// Assert will panic with msg if cond is false.
//
// msg must be a string, func() string or fmt.Stringer.
func Assert(cond bool, msg interface{}) {
}

func Fprintf(w io.Writer, format string, a ...interface{}) {
}

func Fprint(w io.Writer, s string) {
}
