package handlers

import (
	"context"
	"encoding/json"
	"os"
	"sharedlog-stream/benchmark/common"
	datatype "sharedlog-stream/benchmark/lat_tp/pkg/data_type"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/sharedlog_stream"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type sharedlogProduceBenchHandler struct {
	env types.Environment
}

func NewSharedlogProduceBenchHandler(env types.Environment) types.FuncHandler {
	return &sharedlogProduceBenchHandler{
		env: env,
	}
}

func (h *sharedlogProduceBenchHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.BenchSourceParam{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output := h.sharedlogProduceBench(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *sharedlogProduceBenchHandler) sharedlogProduceBench(ctx context.Context, sp *common.BenchSourceParam) *common.FnOutput {
	content, err := os.ReadFile(sp.FileName)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	// latencies := make([]int, 0, 128)
	numEvents := sp.NumEvents
	duration := time.Duration(sp.Duration) * time.Second
	nEmitEvent := uint32(0)
	stream, err := sharedlog_stream.NewShardedSharedLogStream(sp.TopicName,
		sp.NumOutPartition, commtypes.SerdeFormat(sp.SerdeFormat), sp.BufMaxSize)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	var ptSerde datatype.PayloadTsMsgpSerde
	timeGapUs := time.Duration(1000000/sp.Tps) * time.Microsecond
	startTime := time.Now()
	next := time.Now()
	useBuf := ptSerde.UsedBufferPool()

	for {
		// select {
		// case merr := <-msgErrChan:
		// 	return &common.FnOutput{Success: false, Message: merr.Error()}
		// default:
		// }
		if (duration != 0 && time.Since(startTime) >= duration) || (numEvents != 0 && nEmitEvent == numEvents) {
			break
		}
		// procStart := time.Now()
		parNum := nEmitEvent % uint32(sp.NumOutPartition)
		next = next.Add(timeGapUs)
		pt := datatype.PayloadTs{
			Payload: content,
			Ts:      next.UnixMicro(),
		}
		encoded, b, err := ptSerde.Encode(&pt)
		if err != nil {
			commtypes.PushBuffer(&encoded)
			return common.GenErrFnOutput(err)
		}
		now := time.Now()
		if next.After(now) {
			time.Sleep(next.Sub(now))
		}
		_, err = stream.Push(ctx, encoded, uint8(parNum), sharedlog_stream.StreamEntryMeta(false, false),
			commtypes.EmptyProducerId)
		if err != nil {
			commtypes.PushBuffer(&encoded)
			return common.GenErrFnOutput(err)
		}
		if useBuf {
			*b = encoded
			commtypes.PushBuffer(b)
		}
		// streamPusher.MsgChan <- sharedlog_stream.PayloadToPush{Payload: encoded, Partitions: []uint8{uint8(parNum)}, IsControl: false}
		// elapsed := time.Since(procStart)
		// latencies = append(latencies, int(elapsed.Microseconds()))
		nEmitEvent += 1
	}
	return &common.FnOutput{
		Success:  true,
		Duration: time.Since(startTime).Seconds(),
		Counts:   map[string]uint64{"prod": uint64(nEmitEvent)},
		// Latencies: map[string][]int{"e2e": latencies},
	}
}
