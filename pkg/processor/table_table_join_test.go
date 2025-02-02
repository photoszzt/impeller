package processor

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/store"
	"testing"
)

func getJoinTable(t *testing.T) (
	func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, commtypes.ChangeG[string]],
	func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, commtypes.ChangeG[string]],
) {
	tab1 := store.NewInMemorySkipmapKeyValueStoreG[int, commtypes.ValueTimestampG[string]]("tab1", store.IntLessFunc)
	toTab1 := NewTableSourceProcessorWithTableG[int, string](tab1)
	tab2 := store.NewInMemorySkipmapKeyValueStoreG[int, commtypes.ValueTimestampG[string]]("tab2", store.IntLessFunc)
	toTab2 := NewTableSourceProcessorWithTableG[int, string](tab2)
	joiner := ValueJoinerWithKeyFuncG[int, string, string, string](
		func(readOnlyKey int,
			leftValue string, rightValue string,
		) optional.Option[string] {
			debug.Fprintf(os.Stderr, "left val: %v, right val: %v\n", leftValue, rightValue)
			return optional.Some(fmt.Sprintf("%s+%s", leftValue, rightValue))
		})
	oneJoinTwo := NewTableTableJoinProcessorG[int, string, string, string](tab2.Name(), tab2, joiner)
	twoJoinOne := NewTableTableJoinProcessorG[int, string, string, string](tab1.Name(), tab1, ReverseValueJoinerWithKeyG(joiner))
	oneJoinTwoFunc := func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, commtypes.ChangeG[string]] {
		outMsgs, err := toTab1.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		if outMsgs != nil {
			joinedMsgs, err := oneJoinTwo.ProcessAndReturn(ctx, outMsgs[0])
			if err != nil {
				t.Fatal(err.Error())
			}
			return joinedMsgs
		}
		return nil
	}
	twoJoinOneFunc := func(ctx context.Context, m commtypes.MessageG[int, string]) []commtypes.MessageG[int, commtypes.ChangeG[string]] {
		ret, err := toTab2.ProcessAndReturn(ctx, m)
		if err != nil {
			t.Fatal(err.Error())
		}
		if ret != nil {
			joinedMsgs, err := twoJoinOne.ProcessAndReturn(ctx, ret[0])
			if err != nil {
				t.Fatal(err.Error())
			}
			return joinedMsgs
		}
		return nil
	}
	return oneJoinTwoFunc, twoJoinOneFunc
}

func TestTableTableInnerJoin(t *testing.T) {
	ctx := context.Background()
	oneJoinTwo, twoJoinOne := getJoinTable(t)
	expected_keys := []int{0, 1, 2, 3}
	expected_keysOp := []optional.Option[int]{
		optional.Some(0),
		optional.Some(1),
		optional.Some(2),
		optional.Some(3),
	}
	got := make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	for i := 0; i < 2; i++ {
		ret := oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: optional.Some(expected_keys[i]),
			Value: optional.Some(fmt.Sprintf("X%d", expected_keys[i])), TimestampMs: 5 + int64(i)})
		got = append(got, ret...)
	}
	ret := oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: optional.None[int](),
		Value: optional.Some("SomeVal"), TimestampMs: 42})
	// left: X0:0 (ts: 5), X1:1 (ts: 6)
	// right:
	got = append(got, ret...)
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push two items to the other stream. this should produce two items.
	for i := 0; i < 2; i++ {
		ret := twoJoinOne(ctx, commtypes.MessageG[int, string]{Key: optional.Some(expected_keys[i]),
			Value: optional.Some(fmt.Sprintf("Y%d", expected_keys[i])), TimestampMs: 10 * int64(i)})
		got = append(got, ret...)
	}
	ret = twoJoinOne(ctx, commtypes.MessageG[int, string]{Key: optional.None[int](),
		Value: optional.Some("AnotherVal"), TimestampMs: 73})
	got = append(got, ret...)
	// left: X0:0 (ts: 5), X1:1 (ts: 6)
	// right: Y0:0 (ts: 0), Y1:1 (ts: 10)
	expected_join := commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 0, Value: commtypes.NewChangeOnlyNewValG[string]("X0+Y0"), Timestamp: 5},
		{Key: 1, Value: commtypes.NewChangeOnlyNewValG[string]("X1+Y1"), Timestamp: 10},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.MessageToMessageG[int, string](commtypes.Message{Key: k,
			Value: fmt.Sprintf("XX%d", k), Timestamp: 7}))
		got = append(got, ret...)
	}
	// left: XX0:0 (ts: 7), XX1:1 (ts: 7), XX2:2 (ts: 7), XX3:3 (ts: 7)
	// right: Y0:0 (ts: 0), Y1:1 (ts: 10)
	expected_join = commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 0, Value: commtypes.NewChangeG[string]("XX0+Y0", "X0+Y0"), Timestamp: 7},
		{Key: 1, Value: commtypes.NewChangeG[string]("XX1+Y1", "X1+Y1"), Timestamp: 10},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.MessageToMessageG[int, string](commtypes.Message{Key: k,
			Value: fmt.Sprintf("YY%d", k), Timestamp: int64(k * 5)}))
		got = append(got, ret...)
	}
	// left: XX0:0 (ts: 7), XX1:1 (ts: 7), XX2:2 (ts: 7), XX3:3 (ts: 7)
	// right: YY0:0 (ts: 0), YY1:1 (ts: 5), YY2:2 (ts: 10), YY3:3 (ts: 15)
	expected_join = commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 0, Value: commtypes.NewChangeG("XX0+YY0", "XX0+Y0"), Timestamp: 7},
		{Key: 1, Value: commtypes.NewChangeG("XX1+YY1", "XX1+Y1"), Timestamp: 7},
		{Key: 2, Value: commtypes.NewChangeOnlyNewValG("XX2+YY2"), Timestamp: 10},
		{Key: 3, Value: commtypes.NewChangeOnlyNewValG("XX3+YY3"), Timestamp: 15},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.MessageToMessageG[int, string](commtypes.Message{Key: k,
			Value: fmt.Sprintf("XXX%d", k), Timestamp: 6}))
		got = append(got, ret...)
	}
	// left: XXX0:0 (ts: 6), XXX1:1 (ts: 6), XXX2:2 (ts: 6), XXX3:3 (ts: 6)
	// right: YY0:0 (ts: 0), YY1:1 (ts: 5), YY2:2 (ts: 10), YY3:3 (ts: 15)
	expected_join = commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 0, Value: commtypes.NewChangeG("XXX0+YY0", "XX0+YY0"), Timestamp: 6},
		{Key: 1, Value: commtypes.NewChangeG("XXX1+YY1", "XX1+YY1"), Timestamp: 6},
		{Key: 2, Value: commtypes.NewChangeG("XXX2+YY2", "XX2+YY2"), Timestamp: 10},
		{Key: 3, Value: commtypes.NewChangeG("XXX3+YY3", "XX3+YY3"), Timestamp: 15},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	noneStr := optional.None[string]()
	debug.Fprintf(os.Stderr, "####### removing YY0:0 and YY1:1")
	got = make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	// push two items with null to the other stream as deletes. this should produce two item.
	ret = twoJoinOne(ctx, commtypes.MessageG[int, string]{Key: expected_keysOp[0], Value: noneStr, TimestampMs: 5})
	got = append(got, ret...)
	ret = twoJoinOne(ctx, commtypes.MessageG[int, string]{Key: expected_keysOp[1], Value: noneStr, TimestampMs: 7})
	got = append(got, ret...)
	// left: XXX0:0 (ts: 6), XXX1:1 (ts: 6), XXX2:2 (ts: 6), XXX3:3 (ts: 6)
	// right: YY2:2 (ts: 10), YY3:3 (ts: 15)
	expected_join = commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 0, Value: commtypes.NewChangeOnlyOldValG("XXX0+YY0"), Timestamp: 6},
		{Key: 1, Value: commtypes.NewChangeOnlyOldValG("XXX1+YY1"), Timestamp: 7},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push all four items to the primary stream. this should produce two items.
	got = make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: optional.Some(k),
			Value: optional.Some(fmt.Sprintf("XXXX%d", k)), TimestampMs: 13})
		got = append(got, ret...)
	}
	// left: XXXX0:0 (ts: 13), XXXX1:1 (ts: 13), XXXX2:2 (ts: 13), XXXX3:3 (ts: 13)
	// right: YY2:2 (ts: 10), YY3:3 (ts: 15)
	expected_join = commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 2, Value: commtypes.NewChangeG("XXXX2+YY2", "XXX2+YY2"), Timestamp: 13},
		{Key: 3, Value: commtypes.NewChangeG("XXXX3+YY3", "XXX3+YY3"), Timestamp: 15},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.MessageG[int, commtypes.ChangeG[string]], 0)
	// push four items to the primary stream with null. this should produce two items.
	ret = oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: expected_keysOp[0], Value: noneStr, TimestampMs: 0})
	got = append(got, ret...)
	ret = oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: expected_keysOp[1], Value: noneStr, TimestampMs: 42})
	got = append(got, ret...)
	ret = oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: expected_keysOp[2], Value: noneStr, TimestampMs: 5})
	got = append(got, ret...)
	ret = oneJoinTwo(ctx, commtypes.MessageG[int, string]{Key: expected_keysOp[3], Value: noneStr, TimestampMs: 20})
	got = append(got, ret...)
	// left:
	// right: YY2:2 (ts: 10), YY3:3 (ts: 15)
	expected_join = commtypes.MsgArrToMsgGArr[int, commtypes.ChangeG[string]]([]commtypes.Message{
		{Key: 2, Value: commtypes.NewChangeOnlyOldValG("XXXX2+YY2"), Timestamp: 10},
		{Key: 3, Value: commtypes.NewChangeOnlyOldValG("XXXX3+YY3"), Timestamp: 20},
	})
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}
}
