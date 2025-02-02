package benchutil

import (
	"context"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stream_task"
	"time"
)

func UpdateStreamTaskArgs(sp *common.QueryInput, argsBuilder stream_task.SetGuarantee) stream_task.BuildStreamTaskArgs {
	debug.Assert(sp.AppId != "", "app id should not be empty")
	ret := argsBuilder.Guarantee(exactly_once_intr.GuaranteeMth(sp.GuaranteeMth)).
		AppID(sp.AppId).
		Warmup(time.Duration(sp.WarmupS) * time.Second).
		CommitEveryMs(sp.CommitEveryMs).
		FlushEveryMs(sp.FlushMs).
		Duration(sp.Duration).
		SerdeFormat(commtypes.SerdeFormat(sp.SerdeFormat)).
		BufMaxSize(sp.BufMaxSize).
		FaasGateway(sp.FaasGateway).
		Engine1(sp.Engine1).
		WaitEndMark(sp.WaitForEndMark)
	if sp.TestParams != nil {
		ret.TestParams(sp.TestParams)
	}
	if sp.SnapEveryS != 0 {
		ret.SnapshotEveryS(sp.SnapEveryS)
	}
	return ret
}

func GetShardedInputOutputStreams(ctx context.Context,
	input *common.QueryInput,
) (*sharedlog_stream.ShardedSharedLogStream, []*sharedlog_stream.ShardedSharedLogStream, error) {
	serdeFormat := commtypes.SerdeFormat(input.SerdeFormat)
	inputStream, err := sharedlog_stream.NewShardedSharedLogStream(input.InputTopicNames[0], input.NumInPartition,
		serdeFormat, input.BufMaxSize)
	if err != nil {
		return nil, nil, fmt.Errorf("NewSharedlogStream for input stream failed: %v", err)
	}
	var output_streams []*sharedlog_stream.ShardedSharedLogStream
	for idx, name := range input.OutputTopicNames {
		outputStream, err := sharedlog_stream.NewShardedSharedLogStream(name, input.NumOutPartitions[idx],
			serdeFormat, input.BufMaxSize)
		if err != nil {
			return nil, nil, fmt.Errorf("NewSharedlogStream for output stream failed: %v", err)
		}
		output_streams = append(output_streams, outputStream)
	}
	return inputStream, output_streams, nil
}

func GetShardedInputOutputStreamsTest(ctx context.Context,
	input *common.TestParam,
) ([]*sharedlog_stream.ShardedSharedLogStream, []*sharedlog_stream.ShardedSharedLogStream, error) {
	serdeFormat := commtypes.SerdeFormat(input.SerdeFormat)
	var input_streams []*sharedlog_stream.ShardedSharedLogStream
	for _, param := range input.InStreamParam {
		inputStream, err := sharedlog_stream.NewShardedSharedLogStream(param.TopicName,
			param.NumPartition, serdeFormat, input.BufMaxSize)
		if err != nil {
			return nil, nil, fmt.Errorf("NewSharedlogStream for input stream failed: %v", err)
		}
		input_streams = append(input_streams, inputStream)
	}
	var output_streams []*sharedlog_stream.ShardedSharedLogStream
	for _, param := range input.OutStreamParam {
		outputStream, err := sharedlog_stream.NewShardedSharedLogStream(param.TopicName,
			param.NumPartition, serdeFormat, input.BufMaxSize)
		if err != nil {
			return nil, nil, fmt.Errorf("NewSharedlogStream for output stream failed: %v", err)
		}
		output_streams = append(output_streams, outputStream)
	}
	return input_streams, output_streams, nil
}

func UseCache(useCache bool, gua exactly_once_intr.GuaranteeMth) bool {
	c := useCache
	if gua == exactly_once_intr.ALIGN_CHKPT {
		c = false
	}
	return c
}

func GetGuarantee(guarantee_str string) exactly_once_intr.GuaranteeMth {
	guarantee := exactly_once_intr.AT_LEAST_ONCE
	if guarantee_str == "2pc" {
		guarantee = exactly_once_intr.TWO_PHASE_COMMIT
	} else if guarantee_str == "epoch" {
		guarantee = exactly_once_intr.EPOCH_MARK
	} else if guarantee_str == "none" {
		guarantee = exactly_once_intr.NO_GUARANTEE
	} else if guarantee_str == "align_chkpt" {
		guarantee = exactly_once_intr.ALIGN_CHKPT
	} else if guarantee_str == "remote_2pc" {
		guarantee = exactly_once_intr.REMOTE_2PC
	}
	return guarantee
}
