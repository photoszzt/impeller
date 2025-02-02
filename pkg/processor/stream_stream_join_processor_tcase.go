package processor

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"testing"
)

func StreamStreamJoin(ctx context.Context,
	oneJoinTwo func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	twoJoinOne func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	t testing.TB,
) {
	expected_keys := []int{0, 1, 2, 3}
	// push two items to the primary stream; the other window is empty
	// w1 = {}
	// w2 = {}
	// --> w1 = { 0:A0, 1:A1 }
	//     w2 = {}
	got := make([]commtypes.Message, 0)
	for i := 0; i < 2; i++ {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: expected_keys[i], Value: fmt.Sprintf("A%d", expected_keys[i])})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should produce no result")
	}

	got = make([]commtypes.Message, 0)
	// push two items to the other stream; this should produce two items
	// w1 = { 0:A0, 1:A1 }
	// w2 = {}
	// --> w1 = { 0:A0, 1:A1 }
	//     w2 = { 0:a0, 1:a1 }
	for i := 0; i < 2; i++ {
		ret := twoJoinOne(ctx, commtypes.Message{Key: expected_keys[i], Value: fmt.Sprintf("a%d", expected_keys[i])})
		got = append(got, ret...)
	}
	expected_join := []commtypes.Message{
		{Key: 0, Value: "A0+a0", Timestamp: 0},
		{Key: 1, Value: "A1+a1", Timestamp: 0},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.Message, 0)
	// push all four items to the primary stream; this should produce two items
	// w1 = { 0:A0, 1:A1 }
	// w2 = { 0:a0, 1:a1 }
	// --> w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3 }
	//     w2 = { 0:a0, 1:a1 }
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("B%d", k)})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "B0+a0", Timestamp: 0},
		{Key: 1, Value: "B1+a1", Timestamp: 0},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatal("should equal")
	}

	got = make([]commtypes.Message, 0)
	// push all items to the other stream; this should produce six items
	// w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3 }
	// w2 = { 0:a0, 1:a1 }
	// --> w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3 }
	//     w2 = { 0:a0, 1:a1, 0:b0, 1:b1, 2:b2, 3:b3 }
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("b%d", k)})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+b0", Timestamp: 0},
		{Key: 0, Value: "B0+b0", Timestamp: 0},
		{Key: 1, Value: "A1+b1", Timestamp: 0},
		{Key: 1, Value: "B1+b1", Timestamp: 0},
		{Key: 2, Value: "B2+b2", Timestamp: 0},
		{Key: 3, Value: "B3+b3", Timestamp: 0},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatal("should equal")
	}

	got = make([]commtypes.Message, 0)
	// push all four items to the primary stream; this should produce six items
	// w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3 }
	// w2 = { 0:a0, 1:a1, 0:b0, 1:b1, 2:b2, 3:b3 }
	// --> w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3, 0:C0, 1:C1, 2:C2, 3:C3 }
	//     w2 = { 0:a0, 1:a1, 0:b0, 1:b1, 2:b2, 3:b3 }
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("C%d", k)})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "C0+a0", Timestamp: 0},
		{Key: 0, Value: "C0+b0", Timestamp: 0},
		{Key: 1, Value: "C1+a1", Timestamp: 0},
		{Key: 1, Value: "C1+b1", Timestamp: 0},
		{Key: 2, Value: "C2+b2", Timestamp: 0},
		{Key: 3, Value: "C3+b3", Timestamp: 0},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.Message, 0)
	// push two items to the other stream; this should produce six items
	// w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3, 0:C0, 1:C1, 2:C2, 3:C3 }
	// w2 = { 0:a0, 1:a1, 0:b0, 1:b1, 2:b2, 3:b3 }
	// --> w1 = { 0:A0, 1:A1, 0:B0, 1:B1, 2:B2, 3:B3, 0:C0, 1:C1, 2:C2, 3:C3 }
	//     w2 = { 0:a0, 1:a1, 0:b0, 1:b1, 2:b2, 3:b3, 0:c0, 1:c1 }
	for i := 0; i < 2; i++ {
		ret := twoJoinOne(ctx, commtypes.Message{Key: expected_keys[i], Value: fmt.Sprintf("c%d", expected_keys[i])})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+c0", Timestamp: 0},
		{Key: 0, Value: "B0+c0", Timestamp: 0},
		{Key: 0, Value: "C0+c0", Timestamp: 0},
		{Key: 1, Value: "A1+c1", Timestamp: 0},
		{Key: 1, Value: "B1+c1", Timestamp: 0},
		{Key: 1, Value: "C1+c1", Timestamp: 0},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}
}

func Windowing(ctx context.Context,
	oneJoinTwo func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	twoJoinOne func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	t testing.TB,
) {
	expected_keys := []int{0, 1, 2, 3}
	time := int64(0)

	// push two items to the primary stream; the other window is empty; this should produce no items
	// w1 = {}
	// w2 = {}
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0) }
	//     w2 = {}
	got := make([]commtypes.Message, 0)
	for i := 0; i < 2; i++ {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: expected_keys[i],
			Value:     fmt.Sprintf("A%d", expected_keys[i]),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should produce no result")
	}

	got = make([]commtypes.Message, 0)
	// push two items to the other stream; this should produce two items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0) }
	// w2 = {}
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0) }
	for i := 0; i < 2; i++ {
		ret := twoJoinOne(ctx, commtypes.Message{
			Key: expected_keys[i], Value: fmt.Sprintf("a%d", expected_keys[i]), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join := []commtypes.Message{
		{Key: 0, Value: "A0+a0", Timestamp: 0},
		{Key: 1, Value: "A1+a1", Timestamp: 0},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	got = make([]commtypes.Message, 0)
	time = int64(1000)
	// push four items to the primary stream with larger and increasing timestamp; this should produce no items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0) }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0) }
	for idx, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("B%d", k), Timestamp: time + int64(idx)})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should produce no result")
	}

	// push four items to the other stream with fixed larger timestamp; this should produce four items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100) }
	got = make([]commtypes.Message, 0)
	time += int64(100)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("b%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "B0+b0", Timestamp: 1100},
		{Key: 1, Value: "B1+b1", Timestamp: 1100},
		{Key: 2, Value: "B2+b2", Timestamp: 1100},
		{Key: 3, Value: "B3+b3", Timestamp: 1100},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with incremented timestamp; this should produce three items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("c%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 1, Value: "B1+c1", Timestamp: 1101},
		{Key: 2, Value: "B2+c2", Timestamp: 1101},
		{Key: 3, Value: "B3+c3", Timestamp: 1101},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with incremented timestamp; this should produce two items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("d%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 2, Value: "B2+d2", Timestamp: 1102},
		{Key: 3, Value: "B3+d3", Timestamp: 1102},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with incremented timestamp; this should produce one item
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("e%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 3, Value: "B3+e3", Timestamp: 1103},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with incremented timestamp; this should produce no items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//        0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//            0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("f%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items to the other stream with timestamp before the window bound; this should produce no items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//        0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//        0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//            0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//            0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899) }
	time = int64(1000 - 100 - 1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("g%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items to the other stream with with incremented timestamp; this should produce one item
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//        0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//        0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//        0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//            0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//            0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//            0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("h%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "B0+h0", Timestamp: 1000},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with with incremented timestamp; this should produce two items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//        0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//        0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//        0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//        0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//            0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//            0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//            0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900),
	//            0:i0 (ts: 901), 1:i1 (ts: 901), 2:i2 (ts: 901), 3:i3 (ts: 901) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("i%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "B0+i0", Timestamp: 1000},
		{Key: 1, Value: "B1+i1", Timestamp: 1001},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with with incremented timestamp; this should produce three items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//        0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//        0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//        0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//        0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900),
	//        0:i0 (ts: 901), 1:i1 (ts: 901), 2:i2 (ts: 901), 3:i3 (ts: 901) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//            0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//            0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//            0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900),
	//            0:i0 (ts: 901), 1:i1 (ts: 901), 2:i2 (ts: 901), 3:i3 (ts: 901),
	//            0:j0 (ts: 902), 1:j1 (ts: 902), 2:j2 (ts: 902), 3:j3 (ts: 902) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("j%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "B0+j0", Timestamp: 1000},
		{Key: 1, Value: "B1+j1", Timestamp: 1001},
		{Key: 2, Value: "B2+j2", Timestamp: 1002},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items to the other stream with with incremented timestamp; this should produce four items
	// w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//        0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003)  }
	// w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//        0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//        0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//        0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//        0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//        0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//        0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//        0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900),
	//        0:i0 (ts: 901), 1:i1 (ts: 901), 2:i2 (ts: 901), 3:i3 (ts: 901),
	//        0:j0 (ts: 902), 1:j1 (ts: 902), 2:j2 (ts: 902), 3:j3 (ts: 902) }
	// --> w1 = { 0:A0 (ts: 0), 1:A1 (ts: 0),
	//            0:B0 (ts: 1000), 1:B1 (ts: 1001), 2:B2 (ts: 1002), 3:B3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 0), 1:a1 (ts: 0),
	//            0:b0 (ts: 1100), 1:b1 (ts: 1100), 2:b2 (ts: 1100), 3:b3 (ts: 1100),
	//            0:c0 (ts: 1101), 1:c1 (ts: 1101), 2:c2 (ts: 1101), 3:c3 (ts: 1101),
	//            0:d0 (ts: 1102), 1:d1 (ts: 1102), 2:d2 (ts: 1102), 3:d3 (ts: 1102),
	//            0:e0 (ts: 1103), 1:e1 (ts: 1103), 2:e2 (ts: 1103), 3:e3 (ts: 1103),
	//            0:f0 (ts: 1104), 1:f1 (ts: 1104), 2:f2 (ts: 1104), 3:f3 (ts: 1104),
	//            0:g0 (ts: 899), 1:g1 (ts: 899), 2:g2 (ts: 899), 3:g3 (ts: 899),
	//            0:h0 (ts: 900), 1:h1 (ts: 900), 2:h2 (ts: 900), 3:h3 (ts: 900),
	//            0:i0 (ts: 901), 1:i1 (ts: 901), 2:i2 (ts: 901), 3:i3 (ts: 901),
	//            0:j0 (ts: 902), 1:j1 (ts: 902), 2:j2 (ts: 902), 3:j3 (ts: 902) }
	//            0:k0 (ts: 903), 1:k1 (ts: 903), 2:k2 (ts: 903), 3:k3 (ts: 903) }
	time += int64(1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("k%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "B0+k0", Timestamp: 1000},
		{Key: 1, Value: "B1+k1", Timestamp: 1001},
		{Key: 2, Value: "B2+k2", Timestamp: 1002},
		{Key: 3, Value: "B3+k3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// advance time to not join with existing data
	// we omit above exiting data, even if it's still in the window
	//
	// push four items with increasing timestamps to the other stream. the primary window is empty; this should produce no items
	// w1 = {}
	// w2 = {}
	// --> w1 = {}
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time = 2000
	got = make([]commtypes.Message, 0)
	for idx, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("l%d", k), Timestamp: time + int64(idx)})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items with larger timestamps to the primary stream; this should produce four items
	// w1 = {}
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	got = make([]commtypes.Message, 0)
	time = 2000 + 100
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("C%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "C0+l0", Timestamp: 2100},
		{Key: 1, Value: "C1+l1", Timestamp: 2100},
		{Key: 2, Value: "C2+l2", Timestamp: 2100},
		{Key: 3, Value: "C3+l3", Timestamp: 2100},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamps to the primary stream; this should produce three items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("D%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 1, Value: "D1+l1", Timestamp: 2101},
		{Key: 2, Value: "D2+l2", Timestamp: 2101},
		{Key: 3, Value: "D3+l3", Timestamp: 2101},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamps to the primary stream; this should produce two items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("E%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 2, Value: "E2+l2", Timestamp: 2102},
		{Key: 3, Value: "E3+l3", Timestamp: 2102},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamps to the primary stream; this should produce one item
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("F%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 3, Value: "F3+l3", Timestamp: 2103},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamps (now out of window) to the primary stream; this should produce no items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//        0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//            0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("G%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items with smaller timestamps (before window) to the primary stream; this should produce no items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//        0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//        0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//            0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//            0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time = 2000 - 100 - 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("H%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items with increased timestamps to the primary stream; this should produce one item
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//        0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//        0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//        0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//            0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//            0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//            0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("I%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "I0+l0", Timestamp: 2000},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the primary stream; this should produce two items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//        0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//        0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//        0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//        0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//            0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//            0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//            0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900),
	//            0:J0 (ts: 1901), 1:J1 (ts: 1901), 2:J2 (ts: 1901), 3:J3 (ts: 1901) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("J%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "J0+l0", Timestamp: 2000},
		{Key: 1, Value: "J1+l1", Timestamp: 2001},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the primary stream; this should produce three items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//        0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//        0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//        0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//        0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900),
	//        0:J0 (ts: 1901), 1:J1 (ts: 1901), 2:J2 (ts: 1901), 3:J3 (ts: 1901) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//            0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//            0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//            0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900),
	//            0:J0 (ts: 1901), 1:J1 (ts: 1901), 2:J2 (ts: 1901), 3:J3 (ts: 1901),
	//            0:K0 (ts: 1902), 1:K1 (ts: 1902), 2:K2 (ts: 1902), 3:K3 (ts: 1902) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("K%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "K0+l0", Timestamp: 2000},
		{Key: 1, Value: "K1+l1", Timestamp: 2001},
		{Key: 2, Value: "K2+l2", Timestamp: 2002},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the primary stream; this should produce four items
	// w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//        0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//        0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//        0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//        0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//        0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//        0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900),
	//        0:J0 (ts: 1901), 1:J1 (ts: 1901), 2:J2 (ts: 1901), 3:J3 (ts: 1901) }
	//        0:K0 (ts: 1902), 1:K1 (ts: 1902), 2:K2 (ts: 1902), 3:K3 (ts: 1902) }
	// w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	// --> w1 = { 0:C0 (ts: 2100), 1:C1 (ts: 2100), 2:C2 (ts: 2100), 3:C3 (ts: 2100),
	//            0:D0 (ts: 2101), 1:D1 (ts: 2101), 2:D2 (ts: 2101), 3:D3 (ts: 2101),
	//            0:E0 (ts: 2102), 1:E1 (ts: 2102), 2:E2 (ts: 2102), 3:E3 (ts: 2102),
	//            0:F0 (ts: 2103), 1:F1 (ts: 2103), 2:F2 (ts: 2103), 3:F3 (ts: 2103),
	//            0:G0 (ts: 2104), 1:G1 (ts: 2104), 2:G2 (ts: 2104), 3:G3 (ts: 2104),
	//            0:H0 (ts: 1899), 1:H1 (ts: 1899), 2:H2 (ts: 1899), 3:H3 (ts: 1899),
	//            0:I0 (ts: 1900), 1:I1 (ts: 1900), 2:I2 (ts: 1900), 3:I3 (ts: 1900),
	//            0:J0 (ts: 1901), 1:J1 (ts: 1901), 2:J2 (ts: 1901), 3:J3 (ts: 1901),
	//            0:K0 (ts: 1902), 1:K1 (ts: 1902), 2:K2 (ts: 1902), 3:K3 (ts: 1902),
	//            0:L0 (ts: 1903), 1:L1 (ts: 1903), 2:L2 (ts: 1903), 3:L3 (ts: 1903) }
	//     w2 = { 0:l0 (ts: 2000), 1:l1 (ts: 2001), 2:l2 (ts: 2002), 3:l3 (ts: 2003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k, Value: fmt.Sprintf("L%d", k), Timestamp: time})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "L0+l0", Timestamp: 2000},
		{Key: 1, Value: "L1+l1", Timestamp: 2001},
		{Key: 2, Value: "L2+l2", Timestamp: 2002},
		{Key: 3, Value: "L3+l3", Timestamp: 2003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}
}

func AsymmetricWindowingAfter(ctx context.Context,
	oneJoinTwo func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	twoJoinOne func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	t testing.TB,
) {
	expected_keys := []int{0, 1, 2, 3}
	time := int64(1000)

	// push four items with increasing timestamps to the primary stream; the other window is empty; this should produce no items
	// w1 = {}
	// w2 = {}
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = {}
	got := make([]commtypes.Message, 0)
	for idx, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("A%d", k),
			Timestamp: time + int64(idx),
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items smaller timestamps (out of window) to the secondary stream; this should produce no items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = {}
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999) }
	time = int64(1000 - 1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("a%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items with increased timestamps to the secondary stream; this should produce one item
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("b%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join := []commtypes.Message{
		{Key: 0, Value: "A0+b0", Timestamp: 1000},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the secondary stream; this should produce two items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("c%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+c0", Timestamp: 1001},
		{Key: 1, Value: "A1+c1", Timestamp: 1001},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the secondary stream; this should produce three items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("d%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+d0", Timestamp: 1002},
		{Key: 1, Value: "A1+d1", Timestamp: 1002},
		{Key: 2, Value: "A2+d2", Timestamp: 1002},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the secondary stream; this should produce four items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//        0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//            0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("e%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+e0", Timestamp: 1003},
		{Key: 1, Value: "A1+e1", Timestamp: 1003},
		{Key: 2, Value: "A2+e2", Timestamp: 1003},
		{Key: 3, Value: "A3+e3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with larger timestamps to the secondary stream; this should produce four items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//        0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//        0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//            0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//            0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100) }
	time = 1000 + 100
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("f%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+f0", Timestamp: 1100},
		{Key: 1, Value: "A1+f1", Timestamp: 1100},
		{Key: 2, Value: "A2+f2", Timestamp: 1100},
		{Key: 3, Value: "A3+f3", Timestamp: 1100},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the secondary stream; this should produce three items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//        0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//        0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//        0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//            0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//            0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//            0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("g%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 1, Value: "A1+g1", Timestamp: 1101},
		{Key: 2, Value: "A2+g2", Timestamp: 1101},
		{Key: 3, Value: "A3+g3", Timestamp: 1101},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the secondary stream; this should produce two items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//        0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//        0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//        0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//        0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//            0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//            0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//            0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101),
	//            0:h0 (ts: 1102), 1:h1 (ts: 1102), 2:h2 (ts: 1102), 3:h3 (ts: 1102) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("g%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 2, Value: "A2+g2", Timestamp: 1102},
		{Key: 3, Value: "A3+g3", Timestamp: 1102},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps to the secondary stream; this should produce one item
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//        0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//        0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//        0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//        0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101),
	//        0:h0 (ts: 1102), 1:h1 (ts: 1102), 2:h2 (ts: 1102), 3:h3 (ts: 1102) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//            0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//            0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//            0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101),
	//            0:h0 (ts: 1102), 1:h1 (ts: 1102), 2:h2 (ts: 1102), 3:h3 (ts: 1102),
	//            0:i0 (ts: 1103), 1:i1 (ts: 1103), 2:i2 (ts: 1103), 3:i3 (ts: 1103) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("i%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 3, Value: "A3+i3", Timestamp: 1103},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamps (no out of window) to the secondary stream; this should produce no items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//        0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//        0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//        0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//        0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//        0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//        0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101),
	//        0:h0 (ts: 1102), 1:h1 (ts: 1102), 2:h2 (ts: 1102), 3:h3 (ts: 1102),
	//        0:i0 (ts: 1103), 1:i1 (ts: 1103), 2:i2 (ts: 1103), 3:i3 (ts: 1103) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 999), 1:a1 (ts: 999), 2:a2 (ts: 999), 3:a3 (ts: 999),
	//            0:b0 (ts: 1000), 1:b1 (ts: 1000), 2:b2 (ts: 1000), 3:b3 (ts: 1000),
	//            0:c0 (ts: 1001), 1:c1 (ts: 1001), 2:c2 (ts: 1001), 3:c3 (ts: 1001),
	//            0:d0 (ts: 1002), 1:d1 (ts: 1002), 2:d2 (ts: 1002), 3:d3 (ts: 1002),
	//            0:e0 (ts: 1003), 1:e1 (ts: 1003), 2:e2 (ts: 1003), 3:e3 (ts: 1003),
	//            0:f0 (ts: 1100), 1:f1 (ts: 1100), 2:f2 (ts: 1100), 3:f3 (ts: 1100),
	//            0:g0 (ts: 1101), 1:g1 (ts: 1101), 2:g2 (ts: 1101), 3:g3 (ts: 1101),
	//            0:h0 (ts: 1102), 1:h1 (ts: 1102), 2:h2 (ts: 1102), 3:h3 (ts: 1102),
	//            0:i0 (ts: 1103), 1:i1 (ts: 1103), 2:i2 (ts: 1103), 3:i3 (ts: 1103),
	//            0:j0 (ts: 1104), 1:j1 (ts: 1104), 2:j2 (ts: 1104), 3:j3 (ts: 1104) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("j%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}
}

func AsymmetricWindowingBefore(ctx context.Context,
	oneJoinTwo func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	twoJoinOne func(ctx context.Context, m commtypes.Message) []commtypes.Message,
	t testing.TB,
) {
	expected_keys := []int{0, 1, 2, 3}
	time := int64(1000)

	// push four items with increasing timestamps to the primary stream; the other window is empty; this should produce no items
	// w1 = {}
	// w2 = {}
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = {}
	got := make([]commtypes.Message, 0)
	for idx, k := range expected_keys {
		ret := oneJoinTwo(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("A%d", k),
			Timestamp: time + int64(idx),
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items with smaller timestamps (before the window) to the other stream; this should produce no items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = {}
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899) }
	time = int64(1000 - 100 - 1)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("a%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}

	// push four items with increased timestamp to the other stream; this should produce one item
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("b%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join := []commtypes.Message{
		{Key: 0, Value: "A0+b0", Timestamp: 1000},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamp to the other stream; this should produce two items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901) }
	time += 1
	debug.Fprintf(os.Stderr, "current time: %d\n", time)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("c%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+c0", Timestamp: 1000},
		{Key: 1, Value: "A1+c1", Timestamp: 1001},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamp to the other stream; this should produce three items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902) }
	time += 1
	debug.Fprintf(os.Stderr, "current time: %d\n", time)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("d%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+d0", Timestamp: 1000},
		{Key: 1, Value: "A1+d1", Timestamp: 1001},
		{Key: 2, Value: "A2+d2", Timestamp: 1002},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increased timestamp to the other stream; this should produce four items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//        0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//            0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903) }
	time += 1
	debug.Fprintf(os.Stderr, "current time: %d\n", time)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("e%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+e0", Timestamp: 1000},
		{Key: 1, Value: "A1+e1", Timestamp: 1001},
		{Key: 2, Value: "A2+e2", Timestamp: 1002},
		{Key: 3, Value: "A3+e3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with larger timestamp to the other stream; this should produce four items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//        0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//        0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//            0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//            0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000) }
	time = 1000
	debug.Fprintf(os.Stderr, "current time: %d\n", time)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("f%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 0, Value: "A0+f0", Timestamp: 1000},
		{Key: 1, Value: "A1+f1", Timestamp: 1001},
		{Key: 2, Value: "A2+f2", Timestamp: 1002},
		{Key: 3, Value: "A3+f3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamp to the other stream; this should produce three items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//        0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//        0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//        0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//            0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//            0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//            0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001) }
	time += 1
	debug.Fprintf(os.Stderr, "current time: %d\n", time)
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("g%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 1, Value: "A1+g1", Timestamp: 1001},
		{Key: 2, Value: "A2+g2", Timestamp: 1002},
		{Key: 3, Value: "A3+g3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamp to the other stream; this should produce two items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//        0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//        0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//        0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//        0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//            0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//            0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//            0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001),
	//            0:h0 (ts: 1002), 1:h1 (ts: 1002), 2:h2 (ts: 1002), 3:h3 (ts: 1002) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("h%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 2, Value: "A2+h2", Timestamp: 1002},
		{Key: 3, Value: "A3+h3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamp to the other stream; this should produce one item
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//        0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//        0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//        0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//        0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001),
	//        0:h0 (ts: 1002), 1:h1 (ts: 1002), 2:h2 (ts: 1002), 3:h3 (ts: 1002) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//            0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//            0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//            0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001),
	//            0:h0 (ts: 1002), 1:h1 (ts: 1002), 2:h2 (ts: 1002), 3:h3 (ts: 1002),
	//            0:i0 (ts: 1003), 1:i1 (ts: 1003), 2:i2 (ts: 1003), 3:i3 (ts: 1003) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("i%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	expected_join = []commtypes.Message{
		{Key: 3, Value: "A3+i3", Timestamp: 1003},
	}
	if !reflect.DeepEqual(expected_join, got) {
		t.Fatalf("should equal. expected: %v, got: %v", expected_join, got)
	}

	// push four items with increase timestamp (no out of window) to the other stream; this should produce no items
	// w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	// w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//        0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//        0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//        0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//        0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//        0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//        0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001),
	//        0:h0 (ts: 1002), 1:h1 (ts: 1002), 2:h2 (ts: 1002), 3:h3 (ts: 1002),
	//        0:i0 (ts: 1003), 1:i1 (ts: 1003), 2:i2 (ts: 1003), 3:i3 (ts: 1003) }
	// --> w1 = { 0:A0 (ts: 1000), 1:A1 (ts: 1001), 2:A2 (ts: 1002), 3:A3 (ts: 1003) }
	//     w2 = { 0:a0 (ts: 899), 1:a1 (ts: 899), 2:a2 (ts: 899), 3:a3 (ts: 899),
	//            0:b0 (ts: 900), 1:b1 (ts: 900), 2:b2 (ts: 900), 3:b3 (ts: 900),
	//            0:c0 (ts: 901), 1:c1 (ts: 901), 2:c2 (ts: 901), 3:c3 (ts: 901),
	//            0:d0 (ts: 902), 1:d1 (ts: 902), 2:d2 (ts: 902), 3:d3 (ts: 902),
	//            0:e0 (ts: 903), 1:e1 (ts: 903), 2:e2 (ts: 903), 3:e3 (ts: 903),
	//            0:f0 (ts: 1000), 1:f1 (ts: 1000), 2:f2 (ts: 1000), 3:f3 (ts: 1000),
	//            0:g0 (ts: 1001), 1:g1 (ts: 1001), 2:g2 (ts: 1001), 3:g3 (ts: 1001),
	//            0:h0 (ts: 1002), 1:h1 (ts: 1002), 2:h2 (ts: 1002), 3:h3 (ts: 1002),
	//            0:i0 (ts: 1003), 1:i1 (ts: 1003), 2:i2 (ts: 1003), 3:i3 (ts: 1003),
	//            0:j0 (ts: 1004), 1:j1 (ts: 1004), 2:j2 (ts: 1004), 3:j3 (ts: 1004) }
	time += 1
	got = make([]commtypes.Message, 0)
	for _, k := range expected_keys {
		ret := twoJoinOne(ctx, commtypes.Message{Key: k,
			Value:     fmt.Sprintf("i%d", k),
			Timestamp: time,
		})
		got = append(got, ret...)
	}
	if len(got) != 0 {
		t.Fatal("should be empty")
	}
}
