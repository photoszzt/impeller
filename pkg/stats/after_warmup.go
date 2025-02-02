package stats

import (
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/utils/syncutils"
	"sync/atomic"
	"time"
)

type Warmup struct {
	initial          time.Time
	afterWarmupStart time.Time
	warmupTime       time.Duration
	afterWarmup      bool
}

func NewWarmupChecker(warmupTime time.Duration) Warmup {
	afterWarmup := false
	if warmupTime == 0 {
		afterWarmup = true
	}
	return Warmup{
		warmupTime:  warmupTime,
		afterWarmup: afterWarmup,
	}
}

func (w *Warmup) Check() {
	if !w.afterWarmup && time.Since(w.initial) >= w.warmupTime {
		w.afterWarmup = true
		w.afterWarmupStart = time.Now()
	}
}

func (w *Warmup) AfterWarmup() bool {
	return w.afterWarmup
}

func (w *Warmup) StartWarmup() {
	w.initial = time.Now()
	if w.afterWarmup {
		w.afterWarmupStart = w.initial
	}
}

func (w *Warmup) GetStartTime() time.Time {
	return w.initial
}

func (w *Warmup) ElapsedAfterWarmup() time.Duration {
	return time.Since(w.afterWarmupStart)
}

func (w *Warmup) ElapsedSinceInitial() time.Duration {
	debug.Assert(!w.initial.IsZero(), "Warmup not started")
	return time.Since(w.initial)
}

type WarmupGoroutineSafe struct {
	syncutils.Mutex
	afterWarmup      uint32
	initial          time.Time
	afterWarmupStart time.Time
	warmupTime       time.Duration
}

func NewWarmupGoroutineSafeChecker(warmupTime time.Duration) WarmupGoroutineSafe {
	afterWarmup := 0
	if warmupTime == 0 {
		afterWarmup = 1
	}
	return WarmupGoroutineSafe{
		warmupTime:  warmupTime,
		afterWarmup: uint32(afterWarmup),
	}
}

func (w *WarmupGoroutineSafe) Check() {
	if atomic.LoadUint32(&w.afterWarmup) == 0 && time.Since(w.initial) >= w.warmupTime {
		w.Lock()
		atomic.StoreUint32(&w.afterWarmup, 1)
		w.afterWarmupStart = time.Now()
		w.Unlock()
	}
}

func (w *WarmupGoroutineSafe) AfterWarmup() bool {
	return atomic.LoadUint32(&w.afterWarmup) == 1
}

func (w *WarmupGoroutineSafe) StartWarmup() {
	w.Lock()
	w.initial = time.Now()
	if atomic.LoadUint32(&w.afterWarmup) == 1 {
		w.afterWarmupStart = w.initial
	}
	w.Unlock()
}

func (w *WarmupGoroutineSafe) GetStartTime() time.Time {
	w.Lock()
	ret := w.initial
	w.Unlock()
	return ret
}

func (w *WarmupGoroutineSafe) ElapsedAfterWarmup() time.Duration {
	w.Lock()
	ret := time.Since(w.afterWarmupStart)
	w.Unlock()
	return ret
}

func (w *WarmupGoroutineSafe) ElapsedSinceInitial() time.Duration {
	w.Lock()
	ret := time.Since(w.initial)
	w.Unlock()
	return ret
}
