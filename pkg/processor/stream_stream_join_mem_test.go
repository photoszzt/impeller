package processor

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/store"
	"testing"
	"time"
)

/*
func getStreamJoinMem(joinWindows *commtypes.JoinWindows, t *testing.T) (
	func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	func(ctx context.Context, m commtypes.Message) []commtypes.Message,
) {
	toWinTab1, winTab1, err := ToInMemWindowTable("tab1", joinWindows, store.IntIntrCompare)
	if err != nil {
		t.Fatal(err.Error())
	}
	toWinTab2, winTab2, err := ToInMemWindowTable("tab2", joinWindows, store.IntIntrCompare)
	if err != nil {
		t.Fatal(err.Error())
	}
	joiner := ValueJoinerWithKeyTsFunc(
		func(readOnlyKey interface{},
			leftValue interface{}, rightValue interface{}, leftTs int64, rightTs int64,
		) interface{} {
			debug.Fprintf(os.Stderr, "left val: %v, ts: %d, right val: %v, ts: %d\n", leftValue, leftTs, rightValue, rightTs)
			return fmt.Sprintf("%s+%s", leftValue.(string), rightValue.(string))
		})
	sharedTimeTracker := NewTimeTracker()
	oneJoinTwoProc := NewStreamStreamJoinProcessor("oneJoinTwo", winTab2, joinWindows, joiner, false, true, sharedTimeTracker)
	twoJoinOneProc := NewStreamStreamJoinProcessor("twoJoinOne", winTab1, joinWindows, ReverseValueJoinerWithKeyTs(joiner), false, false, sharedTimeTracker)
	oneJoinTwo := func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		_, err := toWinTab1.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		joinedMsgs, err := oneJoinTwoProc.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		return joinedMsgs
	}
	twoJoinOne := func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		_, err := toWinTab2.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		joinedMsgs, err := twoJoinOneProc.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		return joinedMsgs
	}
	return oneJoinTwo, twoJoinOne
}
*/

func getSkipMapStreamJoinMem(joinWindows *commtypes.JoinWindows, t *testing.T) (
	func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, string],
	func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, string],
) {
	toWinTab1, winTab1, err := ToInMemSkipMapWindowTable[int, string]("tab1", joinWindows, store.IntegerCompare[int])
	if err != nil {
		t.Fatal(err.Error())
	}
	toWinTab2, winTab2, err := ToInMemSkipMapWindowTable[int, string]("tab2", joinWindows, store.IntegerCompare[int])
	if err != nil {
		t.Fatal(err.Error())
	}
	joiner := ValueJoinerWithKeyTsFuncG[int, string, string, string](
		func(readOnlyKey int,
			leftValue string, rightValue string, leftTs int64, rightTs int64,
		) optional.Option[string] {
			debug.Fprintf(os.Stderr, "left val: %v, ts: %d, right val: %v, ts: %d\n", leftValue, leftTs, rightValue, rightTs)
			return optional.Some(fmt.Sprintf("%s+%s", leftValue, rightValue))
		})
	sharedTimeTracker := NewTimeTracker()
	oneJoinTwoProc := NewStreamStreamJoinProcessorG[int, string, string, string](
		"oneJoinTwo", winTab2, joinWindows, joiner, false, true, sharedTimeTracker)
	twoJoinOneProc := NewStreamStreamJoinProcessorG[int, string, string, string](
		"twoJoinOne", winTab1, joinWindows, ReverseValueJoinerWithKeyTsG(joiner), false, false, sharedTimeTracker)
	oneJoinTwo := func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, string] {
		_, err := toWinTab1.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		joinedMsgs, err := oneJoinTwoProc.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		return joinedMsgs
	}
	twoJoinOne := func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, string] {
		_, err := toWinTab2.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		joinedMsgs, err := twoJoinOneProc.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		return joinedMsgs
	}
	return oneJoinTwo, twoJoinOne
}

// func TestStreamStreamJoinMem(t *testing.T) {
// 	ctx := context.Background()
// 	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(50)*time.Millisecond,
// 		time.Duration(50)*time.Millisecond)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	oneJoinTwo, twoJoinOne := getStreamJoinMem(joinWindows, t)
// 	StreamStreamJoin(ctx, oneJoinTwo, twoJoinOne, t)
// }

func TestSkipMapStreamStreamJoinMem(t *testing.T) {
	ctx := context.Background()
	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(50)*time.Millisecond,
		time.Duration(50)*time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	oneJoinTwo, twoJoinOne := getSkipMapStreamJoinMem(joinWindows, t)
	StreamStreamJoin(ctx, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := oneJoinTwo(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := twoJoinOne(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, t)
}

// func TestWindowingMem(t *testing.T) {
// 	ctx := context.Background()
// 	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(100)*time.Millisecond, time.Duration(100)*time.Millisecond)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	oneJoinTwo, twoJoinOne := getStreamJoinMem(joinWindows, t)
// 	Windowing(ctx, oneJoinTwo, twoJoinOne, t)
// }

func TestSkipMapWindowingMem(t *testing.T) {
	ctx := context.Background()
	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(100)*time.Millisecond, time.Duration(100)*time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	oneJoinTwo, twoJoinOne := getSkipMapStreamJoinMem(joinWindows, t)
	Windowing(ctx, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := oneJoinTwo(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := twoJoinOne(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, t)
}

// func TestAsymmetricWindowingAfterMem(t *testing.T) {
// 	ctx := context.Background()
// 	joinWindows, err := commtypes.NewJoinWindowsNoGrace(time.Duration(0) * time.Millisecond)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	joinWindows, err = joinWindows.After(time.Duration(100) * time.Millisecond)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	oneJoinTwo, twoJoinOne := getStreamJoinMem(joinWindows, t)
// 	AsymmetricWindowingAfter(ctx, oneJoinTwo, twoJoinOne, t)
// }

func TestSkipMapAsymmetricWindowingAfterMem(t *testing.T) {
	ctx := context.Background()
	joinWindows, err := commtypes.NewJoinWindowsNoGrace(time.Duration(0) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	joinWindows, err = joinWindows.After(time.Duration(100) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	oneJoinTwo, twoJoinOne := getSkipMapStreamJoinMem(joinWindows, t)
	AsymmetricWindowingAfter(ctx, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := oneJoinTwo(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := twoJoinOne(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, t)
}

// func TestAsymmetricWindowingBeforeMem(t *testing.T) {
// 	ctx := context.Background()
// 	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(0)*time.Millisecond,
// 		time.Duration(86400000)*time.Millisecond)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	joinWindows, err = joinWindows.Before(time.Duration(100) * time.Millisecond)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	oneJoinTwo, twoJoinOne := getStreamJoinMem(joinWindows, t)
// 	AsymmetricWindowingBefore(ctx, oneJoinTwo, twoJoinOne, t)
// }

func TestSkipMapAsymmetricWindowingBeforeMem(t *testing.T) {
	ctx := context.Background()
	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(0)*time.Millisecond,
		time.Duration(86400000)*time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	joinWindows, err = joinWindows.Before(time.Duration(100) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	oneJoinTwo, twoJoinOne := getSkipMapStreamJoinMem(joinWindows, t)
	AsymmetricWindowingBefore(ctx, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := oneJoinTwo(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, func(ctx context.Context, m commtypes.Message) []commtypes.Message {
		ret := twoJoinOne(ctx, commtypes.MessageToMessageG[int, string](m))
		var retMsgs []commtypes.Message
		for _, msg := range ret {
			retMsgs = append(retMsgs, commtypes.MessageGToMessage(msg))
		}
		return retMsgs
	}, t)
}
