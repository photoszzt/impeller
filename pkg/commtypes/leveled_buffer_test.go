package commtypes

import (
	"fmt"
	"testing"
	"time"
)

func SetLength[T any](a []T, newLen int) []T {
	if n := newLen - cap(a); n > 0 {
		a = append(a[:cap(a)], make([]T, n)...)
	}
	return a[:newLen]
}

func TestGetPutConcurrent(t *testing.T) {
	const concurrency = 10
	doneCh := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for capacity := -1; capacity < 100; capacity++ {
				bb := getBuf(capacity)
				buf := *bb
				if len(buf) > 0 {
					panic(fmt.Errorf("len(bb) must be zero; got %d", len(buf)))
				}
				if capacity < 0 {
					capacity = 0
				}
				buf = SetLength(buf, len(buf)+capacity)
				*bb = buf
				putBuf(bb)
			}
			doneCh <- struct{}{}
		}()
	}
	tc := time.After(10 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case <-tc:
			t.Fatalf("timeout")
		case <-doneCh:
		}
	}
}
