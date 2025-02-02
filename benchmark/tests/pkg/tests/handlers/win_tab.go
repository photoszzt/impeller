package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/tests/pkg/tests"
	"sharedlog-stream/benchmark/tests/pkg/tests/test_types"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_with_changelog"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type winTabTestsHandler struct {
	env types.Environment
}

func NewWinTabTestsHandler(env types.Environment) types.FuncHandler {
	return &winTabTestsHandler{
		env: env,
	}
}

func getWindowStoreWithChangelog(retainDuplicates bool) *store_with_changelog.InMemoryWindowStoreWithChangelogG[uint32, string] {
	msgSerde := commtypes.MessageGJSONSerdeG[uint32, string]{
		KeySerde: commtypes.Uint32SerdeG{},
		ValSerde: commtypes.StringSerdeG{},
	}
	storeName := "test1"
	mp := store_with_changelog.NewMaterializeParamBuilder[uint32, string]().
		MessageSerde(msgSerde).
		StoreName(storeName).
		ParNum(0).
		SerdeFormat(commtypes.JSON).
		ChangelogManagerParam(commtypes.CreateChangelogManagerParam{
			NumPartition:  1,
			TimeOut:       common.SrcConsumeTimeout,
			FlushDuration: time.Duration(5) * time.Millisecond,
		}).
		BufMaxSize(32 * 1024).
		Build()
	winTab := store.NewInMemorySkipMapWindowStore[uint32, string](mp.StoreName(),
		store.TEST_RETENTION_PERIOD, store.TEST_WINDOW_SIZE, retainDuplicates, store.IntegerCompare[uint32],
	)
	store, err := store_with_changelog.NewInMemoryWindowStoreWithChangelogG[uint32, string](
		winTab, mp,
	)
	if err != nil {
		panic(err)
	}
	return store
}

func (wt *winTabTestsHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &test_types.TestInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output := wt.WinTests(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (wt *winTabTestsHandler) WinTests(ctx context.Context, sp *test_types.TestInput) *common.FnOutput {
	t := &tests.MockTesting{}
	var winstore *store_with_changelog.InMemoryWindowStoreWithChangelogG[uint32, string]
	if sp.TestName != "TestPutSameKeyTs" {
		winstore = getWindowStoreWithChangelog(false)
	} else {
		winstore = getWindowStoreWithChangelog(true)
	}
	switch sp.TestName {
	case "TestGetAndRange":
		store.GetAndRangeTestG(ctx, winstore, t)
	case "TestShouldGetAllNonDeletedMsgs":
		store.ShouldGetAllNonDeletedMsgsTestG(ctx, winstore, t)
	case "TestExpiration":
		store.ExpirationTestG(ctx, winstore, t)
	case "TestShouldGetAll":
		store.ShouldGetAllTestG(ctx, winstore, t)
	case "TestShouldGetAllReturnTimestampOrdered":
		store.ShouldGetAllReturnTimestampOrderedTestG(ctx, winstore, t)
	case "TestFetchRange":
		store.FetchRangeTestG(ctx, winstore, t)
	case "TestPutAndFetchBefore":
		store.PutAndFetchBeforeTestG(ctx, winstore, t)
	case "TestPutAndFetchAfter":
		store.PutAndFetchAfterTestG(ctx, winstore, t)
	case "TestPutSameKeyTs":
		store.PutSameKeyTsTestG(ctx, winstore, t)
	default:
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("test not recognized: %s\n", sp.TestName),
		}
	}
	return &common.FnOutput{
		Success: true,
		Message: fmt.Sprintf("%s done", sp.TestName),
	}
}
