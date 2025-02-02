package commtypes

import (
	"time"

	"golang.org/x/xerrors"
)

/**
 * The window specifications used for joins.
 *
 * A {@code JoinWindows} instance defines a maximum time difference for a {@link KStream#join(KStream, ValueJoiner,
 * JoinWindows) join over two streams} on the same key.
 * In SQL-style you would express this join as
 * {@code
 *     SELECT * FROM stream1, stream2
 *     WHERE
 *       stream1.key = stream2.key
 *       AND
 *       stream1.ts - before <= stream2.ts AND stream2.ts <= stream1.ts + after
 * }
 * There are three different window configuration supported:
 *
 *     - before = after = time-difference
 *     - before = 0 and after = time-difference
 *     - before = time-difference and after = 0
 *
 * A join is symmetric in the sense, that a join specification on the first stream returns the same result record as
 * a join specification on the second stream with flipped before and after values.
 *
 * Both values (before and after) must not result in an "inverse" window, i.e., upper-interval bound cannot be smaller
 * than lower-interval bound.
 *
 * {@code JoinWindows} are sliding windows, thus, they are aligned to the actual record timestamps.
 * This implies, that each input record defines its own window with start and end time being relative to the record's
 * timestamp.
 */
type JoinWindows struct {
	beforeMs int64
	afterMs  int64
	graceMs  int64
}

var _ = EnumerableWindowDefinition(&JoinWindows{})

func getJoinWindows(beforeMs int64, afterMs int64, graceMs int64) (*JoinWindows, error) {
	if beforeMs+afterMs < 0 {
		return nil, DurationLeqZero
	}
	if graceMs < 0 {
		return nil, DurationLeqZero
	}
	return &JoinWindows{
		beforeMs: beforeMs,
		afterMs:  afterMs,
		graceMs:  graceMs,
	}, nil
}

func NewJoinWindowsNoGrace(timeDifference time.Duration) (*JoinWindows, error) {
	timeDifferenceMs := timeDifference.Milliseconds()
	return getJoinWindows(timeDifferenceMs, timeDifferenceMs, 0)
}

func NewJoinWindowsWithGrace(timeDifference time.Duration, afterWindowEnd time.Duration) (*JoinWindows, error) {
	timeDifferenceMs := timeDifference.Milliseconds()
	afterWindowEndMs := afterWindowEnd.Milliseconds()
	return getJoinWindows(timeDifferenceMs, timeDifferenceMs, afterWindowEndMs)
}

/**
 * Changes the start window boundary to {@code timeDifference} but keep the end window boundary as is.
 * Thus, records of the same key are joinable if the timestamp of a record from the secondary stream is at most
 * {@code timeDifference} earlier than the timestamp of the record from the primary stream.
 * {@code timeDifference} can be negative but its absolute value must not be larger than current window "after"
 * value (which would result in a negative window size).
 *
 * @param timeDifference relative window start time
 * @panic if the resulting window size is negative or {@code timeDifference} can't be represented as {@code uint64 milliseconds}
 */
func (w *JoinWindows) Before(timeDifference time.Duration) (*JoinWindows, error) {
	timeDifferenceMs := timeDifference.Milliseconds()
	return getJoinWindows(timeDifferenceMs, w.afterMs, w.graceMs)
}

func (w *JoinWindows) After(timeDifference time.Duration) (*JoinWindows, error) {
	timeDifferenceMs := timeDifference.Milliseconds()
	return getJoinWindows(w.beforeMs, timeDifferenceMs, w.graceMs)
}

func (w *JoinWindows) BeforeMs() int64 {
	return w.beforeMs
}

func (w *JoinWindows) AfterMs() int64 {
	return w.afterMs
}

func (w *JoinWindows) WindowsFor(timestamp int64) (map[int64]Window, []int64, error) {
	return nil, nil, xerrors.New("WindowsFor is not supported by JoinWindows")
}

func (w *JoinWindows) MaxSize() int64 {
	return w.beforeMs + w.afterMs
}

func (w *JoinWindows) GracePeriodMs() int64 {
	return w.graceMs
}
