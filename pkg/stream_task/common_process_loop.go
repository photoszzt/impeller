package stream_task

import (
	"context"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"

	// "sharedlog-stream/pkg/producer_consumer"
	"time"

	"golang.org/x/xerrors"
)

func CommonAppProcessFunc[K, V any](proc processor.IProcessFuncG[K, V], inMsgSerde commtypes.MessageGSerdeG[K, V]) ProcessFunc {
	return func(ctx context.Context, task *StreamTask, args processor.ExecutionContext) (*common.FnOutput, []*commtypes.RawMsgAndSeq) {
		return CommonProcess(ctx, task, args.(*processor.BaseExecutionContext),
			func(ctx context.Context, msg commtypes.MessageG[K, V], argsTmp interface{}) error {
				return proc(ctx, msg)
			}, inMsgSerde)
	}
}

func CommonProcess[K, V any](ctx context.Context, t *StreamTask, ectx *processor.BaseExecutionContext,
	procMsg proc_interface.ProcessMsgFunc[K, V], inMsgSerde commtypes.MessageGSerdeG[K, V],
) (*common.FnOutput, []*commtypes.RawMsgAndSeq) {
	if t.HandleErrFunc != nil {
		if err := t.HandleErrFunc(); err != nil {
			return common.GenErrFnOutput(fmt.Errorf("handleErrFunc: %v", err)), nil
		}
	}
	consumer := ectx.Consumers()[0]
	isInitialSrc := consumer.IsInitialSource()
	rawMsgSeq, err := consumer.Consume(ctx, ectx.SubstreamNum())
	if err != nil {
		// debug.Fprintf(os.Stderr, "consume %s %d error %v\n",
		// 	consumer.TopicName(), ectx.SubstreamNum(), err)
		if xerrors.Is(err, common_errors.ErrStreamSourceTimeout) {
			// count := consumer.GetCount()
			// debug.Fprintf(os.Stderr, "consume %s %d timeout, count %d\n",
			// 	consumer.TopicName(), ectx.SubstreamNum(), count)
			return &common.FnOutput{Success: true, Message: err.Error()}, nil
		}
		return common.GenErrFnOutput(fmt.Errorf("Consume: %v", err)), nil
	}
	if rawMsgSeq.IsControl {
		if rawMsgSeq.Mark == commtypes.SCALE_FENCE {
			if ectx.CurEpoch() < rawMsgSeq.ScaleEpoch {
				ectx.Consumers()[0].RecordCurrentConsumedSeqNum(rawMsgSeq.LogSeqNum)
				return nil, []*commtypes.RawMsgAndSeq{rawMsgSeq}
			}
			return nil, nil
		} else if rawMsgSeq.Mark == commtypes.STREAM_END {
			consumer.SrcProducerEnd(rawMsgSeq.ProdIdx)
			if consumer.AllProducerEnded() {
				return nil, []*commtypes.RawMsgAndSeq{rawMsgSeq}
			} else {
				return nil, nil
			}
		} else if rawMsgSeq.Mark == commtypes.EPOCH_END {
			return nil, nil
		} else if rawMsgSeq.Mark == commtypes.CHKPT_MARK {
			return nil, []*commtypes.RawMsgAndSeq{rawMsgSeq}
		}
	}
	msgs, err := commtypes.DecodeRawMsgSeqG(rawMsgSeq, inMsgSerde)
	if err != nil {
		return common.GenErrFnOutput(err), nil
	}
	if msgs.MsgArr == nil && msgs.Msg.Key.IsNone() && msgs.Msg.Value.IsNone() {
		return nil, nil
	}
	ectx.Consumers()[0].RecordCurrentConsumedSeqNum(msgs.LogSeqNum)
	err = processMsgAndSeq(ctx, msgs, ectx, procMsg, isInitialSrc, rawMsgSeq.InjTsMs)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("processMsgAndSeq: %v", err)), nil
	}
	return nil, nil
}

func processMsgAndSeq[K, V any](ctx context.Context,
	msg *commtypes.MsgAndSeqG[K, V],
	args processor.ExecutionContext,
	procMsg proc_interface.ProcessMsgFunc[K, V], isInitialSrc bool,
	beforeInjToStreamTs int64,
) error {
	// meteredConsumer := args.Consumers()[0]
	_ = beforeInjToStreamTs
	if msg.MsgArr != nil {
		for _, subMsg := range msg.MsgArr {
			if subMsg.Key.IsNone() && subMsg.Value.IsNone() {
				continue
			}
			// if !isInitialSrc {
			// 	batchTime := beforeInjToStreamTs - subMsg.InjTMs
			// 	meteredConsumer.CollectBatchTime(batchTime)
			// }
			// producer_consumer.ExtractProduceToConsumeTimeMsgG(meteredConsumer, &subMsg)
			if isInitialSrc {
				err := subMsg.ExtractEventTimeFromVal()
				if err != nil {
					return fmt.Errorf("ExtractEventTimeFromVal: %v", err)
				}
			}
			subMsg.StartProcTime = time.Now()
			err := procMsg(ctx, subMsg, args)
			if err != nil {
				return fmt.Errorf("procMsg: %v", err)
			}
		}
		return nil
	} else {
		if isInitialSrc {
			err := msg.Msg.ExtractEventTimeFromVal()
			if err != nil {
				return fmt.Errorf("ExtractEventTimeFromVal: %v", err)
			}
		}
		// producer_consumer.ExtractProduceToConsumeTimeMsgG(meteredConsumer, &msg.Msg)
		msg.Msg.StartProcTime = time.Now()
		err := procMsg(ctx, msg.Msg, args)
		if err != nil {
			return fmt.Errorf("procMsg: %v", err)
		} else {
			return nil
		}
	}
}
