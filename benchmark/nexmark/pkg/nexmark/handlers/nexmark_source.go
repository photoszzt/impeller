package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/generator"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/checkpt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/control_channel"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils"
	"sync"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type nexmarkSourceHandler struct {
	env              types.Environment
	funcName         string
	msgSerde         commtypes.MessageGSerdeG[string, *ntypes.Event]
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker]
	generatorConfig  *generator.GeneratorConfig
	eventsPerGen     uint64
	duration         time.Duration
	streamPusher     *sharedlog_stream.StreamPush
	// rcm              checkpt.RedisChkptManager
	chkptClient checkpt.ChkptMngrClient
	bufPush     bool
}

func NewNexmarkSource(env types.Environment, funcName string) types.FuncHandler {
	return &nexmarkSourceHandler{
		env:      env,
		funcName: funcName,
		bufPush:  utils.CheckBufPush(),
	}
}

func (h *nexmarkSourceHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	inputConfig := &ntypes.NexMarkConfigInput{}
	err := json.Unmarshal(input, inputConfig)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	// h.rcm = checkpt.NewRedisChkptManager()
	output := h.eventGeneration(ctx, inputConfig)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

type nexmarkSrcProcArgs struct {
	channel_url_cache map[uint32]*generator.ChannelUrl
	eventGenerator    *generator.NexmarkGenerator
	msgChan           chan sharedlog_stream.PayloadToPush
	parNumArr         []uint8
	latencies         stats.StatsCollector[int64]
	appId             string
	idx               int
}

func (h *nexmarkSourceHandler) process(ctx context.Context, args *nexmarkSrcProcArgs) *common.FnOutput {
	procStart := stats.TimerBegin()
	var msg commtypes.MessageG[string, *ntypes.Event]
	nextEvent, err := args.eventGenerator.NextEvent(ctx, args.channel_url_cache)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("next event failed: %v\n", err))
	}
	if args.appId == "fanout" {
		msg = commtypes.MessageG[string, *ntypes.Event]{
			Key: optional.None[string](),
			Value: optional.Some(ntypes.NewFanoutEvent(
				&ntypes.Fanout{
					DateTime: nextEvent.EventTimestamp,
					Extra:    generator.NextExactString(args.eventGenerator.Random, 1016),
				})),
		}
	} else {
		// fmt.Fprintf(os.Stderr, "gen event with ts: %v\n", nextEvent.EventTimestamp)
		msg = commtypes.MessageG[string, *ntypes.Event]{
			Key:   optional.None[string](),
			Value: optional.Some(nextEvent.Event),
		}
	}
	msgEncoded, _, err := h.msgSerde.Encode(msg)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("msg serialization failed: %v", err))
	}
	// fmt.Fprintf(os.Stderr, "msg: %v\n", string(msgEncoded))
	args.idx += 1
	idx := args.idx % len(args.parNumArr)
	// parNum := args.idx
	parNum := args.parNumArr[idx]

	nowMs := time.Now().UnixMilli()
	wtsMs := nextEvent.WallclockTimestamp
	if wtsMs > nowMs {
		// fmt.Fprintf(os.Stderr, "sleep %v ms to generate event\n", wtsSec-now)
		time.Sleep(time.Duration(wtsMs-nowMs) * time.Millisecond)
	}
	args.msgChan <- sharedlog_stream.PayloadToPush{
		Payload:    msgEncoded,
		Partitions: []uint8{parNum},
		IsControl:  false,
	}
	elapsed := stats.Elapsed(procStart).Microseconds()
	args.latencies.AddSample(elapsed)
	return nil
}

func getSubstreamIdx(parNum uint8, numPartition uint8, numGenerator uint8) []uint8 {
	if numPartition > numGenerator {
		numSubStreamPerGen := numPartition / numGenerator
		substreamIdx := make([]uint8, 0, numSubStreamPerGen)
		startIdx := parNum * numSubStreamPerGen
		endIdx := startIdx + numSubStreamPerGen
		for i := startIdx; i < endIdx; i++ {
			substreamIdx = append(substreamIdx, i)
		}
		return substreamIdx
	} else {
		return []uint8{parNum % numPartition}
	}
}

func (h *nexmarkSourceHandler) setupGeneratorConfig(inputConfig *ntypes.NexMarkConfigInput) *common.FnOutput {
	nexmarkConfig, err := ntypes.ConvertToNexmarkConfiguration(inputConfig)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("fail to convert to nexmark configuration: %v", err))
	}
	generatorConfig := generator.NewGeneratorConfig(nexmarkConfig, time.Now().UnixMilli(), 1, uint64(nexmarkConfig.NumEvents), 1)
	fmt.Fprint(os.Stderr, "Generator config: \n")
	fmt.Fprintf(os.Stderr, "\tInterEventDelayUs: %v\n", generatorConfig.InterEventDelayUs)
	fmt.Fprintf(os.Stderr, "\tEventPerEpoch    : %v\n", generatorConfig.EventPerEpoch)
	fmt.Fprintf(os.Stderr, "\tMaxEvents        : %v\n", generatorConfig.MaxEvents)
	fmt.Fprintf(os.Stderr, "\tFirstEventNumber : %v\n", generatorConfig.FirstEventNumber)
	fmt.Fprintf(os.Stderr, "\tEpochPeriodMs    : %v\n", generatorConfig.EpochPeriodMs)
	fmt.Fprintf(os.Stderr, "\tStepLengthSec    : %v\n", generatorConfig.StepLengthSec)
	fmt.Fprintf(os.Stderr, "\tBaseTime         : %v\n", generatorConfig.BaseTime)
	fmt.Fprintf(os.Stderr, "\tFirstEventId     : %v\n", generatorConfig.FirstEventId)
	fmt.Fprintf(os.Stderr, "\tTotalProportion  : %v\n", generatorConfig.TotalProportion)
	fmt.Fprintf(os.Stderr, "\tBidProportion    : %v\n", generatorConfig.BidProportion)
	fmt.Fprintf(os.Stderr, "\tAuctionProportion: %d\n", generatorConfig.AuctionProportion)
	fmt.Fprintf(os.Stderr, "\tPersonProportion : %d\n", generatorConfig.PersonProportion)

	fmt.Fprint(os.Stderr, "Nexmark config: \n")
	fmt.Fprintf(os.Stderr, "\tNumEvents            : %v\n", generatorConfig.Configuration.NumEvents)
	fmt.Fprintf(os.Stderr, "\tNumEventGenerators   : %v\n", generatorConfig.Configuration.NumEventGenerators)
	fmt.Fprintf(os.Stderr, "\tRateShape            : %v\n", generatorConfig.Configuration.RateShape)
	fmt.Fprintf(os.Stderr, "\tFirstEventRate       : %v\n", generatorConfig.Configuration.FirstEventRate)
	fmt.Fprintf(os.Stderr, "\tNextEventRate        : %v\n", generatorConfig.Configuration.NextEventRate)
	fmt.Fprintf(os.Stderr, "\tRateUnit             : %v\n", generatorConfig.Configuration.RateUnit)
	fmt.Fprintf(os.Stderr, "\tRatePeriodSec        : %v\n", generatorConfig.Configuration.RatePeriodSec)
	fmt.Fprintf(os.Stderr, "\tPreloadSeconds       : %v\n", generatorConfig.Configuration.PreloadSeconds)
	fmt.Fprintf(os.Stderr, "\tStreamTimeout        : %v\n", generatorConfig.Configuration.StreamTimeout)
	fmt.Fprintf(os.Stderr, "\tIsRateLimited        : %v\n", generatorConfig.Configuration.IsRateLimited)
	fmt.Fprintf(os.Stderr, "\tUseWallclockEventTime: %v\n", generatorConfig.Configuration.UseWallclockEventTime)
	fmt.Fprintf(os.Stderr, "\tAvgPersonByteSize    : %v\n", generatorConfig.Configuration.AvgPersonByteSize)
	fmt.Fprintf(os.Stderr, "\tAvgAuctionByteSize   : %v\n", generatorConfig.Configuration.AvgAuctionByteSize)
	fmt.Fprintf(os.Stderr, "\tAvgBidByteSize       : %v\n", generatorConfig.Configuration.AvgBidByteSize)
	fmt.Fprintf(os.Stderr, "\tHotAuctionRatio      : %v\n", generatorConfig.Configuration.HotAuctionRatio)
	fmt.Fprintf(os.Stderr, "\tHotSellersRatio      : %v\n", generatorConfig.Configuration.HotSellersRatio)
	fmt.Fprintf(os.Stderr, "\tHotBiddersRatio      : %v\n", generatorConfig.Configuration.HotBiddersRatio)
	fmt.Fprintf(os.Stderr, "\tWindowSizeSec        : %v\n", generatorConfig.Configuration.WindowSizeSec)
	fmt.Fprintf(os.Stderr, "\tWindowPeriodSec      : %v\n", generatorConfig.Configuration.WindowPeriodSec)
	fmt.Fprintf(os.Stderr, "\tWatermarkHoldbackSec : %v\n", generatorConfig.Configuration.WatermarkHoldbackSec)
	fmt.Fprintf(os.Stderr, "\tNumInFlightAuctions  : %v\n", generatorConfig.Configuration.NumInFlightAuctions)
	fmt.Fprintf(os.Stderr, "\tNumActivePeople      : %v\n", generatorConfig.Configuration.NumActivePeople)
	fmt.Fprintf(os.Stderr, "\tOccasionalDelaySec   : %v\n", generatorConfig.Configuration.OccasionalDelaySec)
	fmt.Fprintf(os.Stderr, "\tProbDelayedEvent     : %v\n", generatorConfig.Configuration.ProbDelayedEvent)
	fmt.Fprintf(os.Stderr, "\tOutOfOrderGroupSize  : %v\n", generatorConfig.Configuration.OutOfOrderGroupSize)

	fmt.Fprintf(os.Stderr, "Input config: \n")
	fmt.Fprintf(os.Stderr, "\tSrcTopicName   : %v\n", inputConfig.TopicName)
	fmt.Fprintf(os.Stderr, "\tAppId          : %v\n", inputConfig.AppId)
	fmt.Fprintf(os.Stderr, "\tFinalOutTpNames: %v\n", inputConfig.FinalOutTpNames)
	fmt.Fprintf(os.Stderr, "\tDuration(s)    : %v\n", inputConfig.Duration)
	fmt.Fprintf(os.Stderr, "\tFlushMs        : %v\n", inputConfig.FlushMs)
	fmt.Fprintf(os.Stderr, "\tBufMaxSize     : %v\n", inputConfig.BufMaxSize)
	fmt.Fprintf(os.Stderr, "\tCommitEveryMs  : %v\n", inputConfig.CommitEveryMs)
	fmt.Fprintf(os.Stderr, "\tWaitForEndMark : %v\n", inputConfig.WaitForEndMark)
	fmt.Fprintf(os.Stderr, "\tSerdeFormat    : %v\n", commtypes.SerdeFormat(inputConfig.SerdeFormat).String())
	fmt.Fprintf(os.Stderr, "\tNumOutPartition: %v\n", inputConfig.NumOutPartition)
	fmt.Fprintf(os.Stderr, "\tParNum         : %v\n", inputConfig.ParNum)
	fmt.Fprintf(os.Stderr, "\tNumSrcInstance : %v\n", inputConfig.NumSrcInstance)
	fmt.Fprintf(os.Stderr, "\tGuarantee      : %v\n", exactly_once_intr.GuaranteeMth(inputConfig.GuaranteeMth).String())
	fmt.Fprintf(os.Stderr, "\tEngine1        : %v\n", inputConfig.Engine1)
	h.generatorConfig = generatorConfig
	h.eventsPerGen = inputConfig.EventsNum / uint64(h.generatorConfig.Configuration.NumEventGenerators)
	h.duration = time.Duration(inputConfig.Duration) * time.Second
	debug.Fprintf(os.Stderr, "Events per gen: %v, duration: %v\n", h.eventsPerGen, h.duration)
	return nil
}

func (h *nexmarkSourceHandler) getStreamPublisher(inputConfig *ntypes.NexMarkConfigInput) *common.FnOutput {
	stream, err := sharedlog_stream.NewSizableShardedSharedLogStream(inputConfig.TopicName,
		inputConfig.NumOutPartition,
		commtypes.SerdeFormat(inputConfig.SerdeFormat), inputConfig.BufMaxSize)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("fail to create output stream: %v", err))
	}
	h.streamPusher = sharedlog_stream.NewStreamPush(stream)
	return nil
}

func (h *nexmarkSourceHandler) getProcArgs(inputConfig *ntypes.NexMarkConfigInput) *nexmarkSrcProcArgs {
	substreamIdxOut := getSubstreamIdx(inputConfig.ParNum, inputConfig.NumOutPartition,
		uint8(h.generatorConfig.Configuration.NumEventGenerators))
	fmt.Fprintf(os.Stderr, "generate events to %v\n", substreamIdxOut)
	eventGenerator := generator.NewSimpleNexmarkGenerator(h.generatorConfig, int(inputConfig.ParNum))
	return &nexmarkSrcProcArgs{
		channel_url_cache: make(map[uint32]*generator.ChannelUrl),
		eventGenerator:    eventGenerator,
		idx:               0,
		latencies: stats.NewStatsCollector[int64](fmt.Sprintf("srcGen_%d", inputConfig.ParNum),
			stats.DEFAULT_COLLECT_DURATION),
		msgChan:   h.streamPusher.MsgChan,
		parNumArr: substreamIdxOut,
		appId:     inputConfig.AppId,
	}
}

func (h *nexmarkSourceHandler) setupSerde(sf uint8) *common.FnOutput {
	var err error
	serdeFormat := commtypes.SerdeFormat(sf)
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.msgSerde, err = commtypes.GetMsgGSerdeG[string](serdeFormat, commtypes.StringSerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.epochMarkerSerde, err = commtypes.GetEpochMarkerSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *nexmarkSourceHandler) stop(ctx context.Context, wg *sync.WaitGroup, cmm *control_channel.ControlChannelManager) {
	cmm.SendQuit()
	close(h.streamPusher.MsgChan)
	wg.Wait()
	h.flush(ctx, h.streamPusher.Stream)
}

func (h *nexmarkSourceHandler) succRet(startTime time.Time) *common.FnOutput {
	return &common.FnOutput{
		Success:  true,
		Duration: time.Since(startTime).Seconds(),
		Counts: map[string]uint64{
			"sink":      h.streamPusher.GetCount(),
			"sink_ctrl": h.streamPusher.NumCtrlMsgs(),
		},
	}
}

func (h *nexmarkSourceHandler) propagateScaleFence(m *txn_data.ControlMetadata, procArgs *nexmarkSrcProcArgs, parNum uint8) *common.FnOutput {
	numSubstreams := m.Config[h.streamPusher.Stream.TopicName()]
	for len(h.streamPusher.MsgChan) > 0 {
		time.Sleep(time.Duration(100) * time.Microsecond)
	}
	err := h.streamPusher.Stream.ScaleSubstreams(numSubstreams)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	// piggy back scale fence in txn marker
	txnMarker := commtypes.EpochMarker{
		Mark:       commtypes.SCALE_FENCE,
		ScaleEpoch: m.Epoch,
	}
	encoded, _, err := h.epochMarkerSerde.Encode(txnMarker)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	procArgs.parNumArr = getSubstreamIdx(parNum, numSubstreams,
		uint8(h.generatorConfig.Configuration.NumEventGenerators))
	debug.Fprintf(os.Stderr, "updated numPartition is %d, generate events to %v\n",
		numSubstreams, procArgs.parNumArr)
	h.streamPusher.MsgChan <- sharedlog_stream.PayloadToPush{
		Payload:    encoded,
		IsControl:  true,
		Partitions: []uint8{parNum},
		Mark:       commtypes.SCALE_FENCE,
	}
	return nil
}

func (h *nexmarkSourceHandler) eventGeneration(
	ctx context.Context, inputConfig *ntypes.NexMarkConfigInput,
) *common.FnOutput {
	dctx, dcancel := context.WithCancel(ctx)
	defer dcancel()
	var wg sync.WaitGroup
	fn_out := h.getStreamPublisher(inputConfig)
	if fn_out != nil {
		return fn_out
	}
	fn_out = h.setupGeneratorConfig(inputConfig)
	if fn_out != nil {
		return fn_out
	}
	fn_out = h.setupSerde(inputConfig.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	cmm, err := control_channel.NewControlChannelManager(inputConfig.AppId,
		commtypes.SerdeFormat(inputConfig.SerdeFormat), inputConfig.BufMaxSize, 0, inputConfig.ParNum)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	cmm.StartMonitorControlChannel(dctx)

	procArgs := h.getProcArgs(inputConfig)
	wg.Add(1)
	go h.streamPusher.AsyncStreamPush(dctx, &wg, commtypes.EmptyProducerId, func(ctx context.Context) error { return nil })
	h.streamPusher.InitFlushTimer(time.Duration(inputConfig.FlushMs) * time.Millisecond)
	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "StartTs: %d\n", startTime.UnixMilli())
	commitEveryMs := time.Duration(inputConfig.CommitEveryMs) * time.Millisecond
	gua := exactly_once_intr.GuaranteeMth(inputConfig.GuaranteeMth)
	if gua == exactly_once_intr.ALIGN_CHKPT {
		// conn, err := stream_task.PrepareChkptClientGrpc(inputConfig.Engine1)
		conn, err := checkpt.PrepareChkptClientGrpc()
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		client := checkpt.NewChkptMngrClient(conn)
		h.chkptClient = client
		for {
			if time.Since(startTime) >= commitEveryMs {
				// err := h.genFirstChkpt(ctx, inputConfig.FinalOutTpNames, inputConfig.ParNum)
				err := h.genFirstChkpt(inputConfig.ParNum)
				if err != nil {
					fmt.Fprintf(os.Stderr, "gen first chkpt failed: %v\n", err)
				}
				fmt.Fprintf(os.Stderr, "gen first chkpt\n")
				break
			}
			fn_out = h.process(dctx, procArgs)
			if fn_out != nil && !fn_out.Success {
				h.stop(dctx, &wg, cmm)
				return fn_out
			}
		}
	}

	for {
		select {
		case merr := <-h.streamPusher.MsgErrChan:
			cmm.SendQuit()
			return common.GenErrFnOutput(fmt.Errorf("failed to push to src stream: %v", merr))
		case out := <-cmm.OutputChan():
			if out.Valid() {
				m := out.Value()
				debug.Fprintf(os.Stderr, "got data from control channel: %+v\n", m)
				// don't need to send scale fence on the initial deployment
				if len(m.Config) != 0 && m.Epoch != 1 {
					numInstance := m.Config[h.funcName]
					if inputConfig.ParNum >= numInstance {
						fmt.Fprintf(os.Stderr, "funcName: %s, parNum %v >= numInstance %v, quit\n", h.funcName, inputConfig.ParNum, numInstance)
						h.stop(dctx, &wg, cmm)
						return h.succRet(startTime)
					}
					fn_out = h.propagateScaleFence(m, procArgs, inputConfig.ParNum)
					if fn_out != nil {
						return fn_out
					}
				}
			} else {
				cerr := out.Err()
				h.stop(dctx, &wg, cmm)
				return common.GenErrFnOutput(fmt.Errorf("control channel manager failed: %v", cerr))
			}
		default:
		}
		if !procArgs.eventGenerator.HasNext() || procArgs.idx == int(h.eventsPerGen) || h.duration != 0 && time.Since(startTime) >= h.duration {
			if inputConfig.WaitForEndMark {
				if gua == exactly_once_intr.ALIGN_CHKPT {
					debug.Fprintf(os.Stderr, "requesting chkmngr to end\n")
					_, err = h.chkptClient.ReqChkmngrEndedIfNot(ctx, &emptypb.Empty{})
					// err = h.rcm.ReqChkMngrEnd(ctx)
					// if err != nil {
					// 	return common.GenErrFnOutput(err)
					// }
					// for {
					// 	got, err := h.rcm.GetChkMngrEnded(ctx)
					// 	if err != nil {
					// 		return common.GenErrFnOutput(err)
					// 	}
					// 	if got == 1 {
					// 		break
					// 	}
					// }
					fmt.Fprintf(os.Stderr, "[%v] chkmngr has terminated\n", inputConfig.ParNum)
				}
				for _, par := range procArgs.parNumArr {
					ret_err := h.genEndMark(startTime, par)
					if ret_err != nil {
						return ret_err
					}
				}
			}
			break
		}
		fn_out = h.process(dctx, procArgs)
		if fn_out != nil && !fn_out.Success {
			h.stop(dctx, &wg, cmm)
			return fn_out
		}
	}
	h.stop(dctx, &wg, cmm)
	fmt.Fprintf(os.Stderr, "[%v] all stopped\n", inputConfig.ParNum)
	return h.succRet(startTime)
}

func (h *nexmarkSourceHandler) genEndMark(startTime time.Time, parNum uint8,
) *common.FnOutput {
	// debug.Fprintf(os.Stderr, "generator exhausted, stop\n")
	marker := commtypes.EpochMarker{
		Mark:      commtypes.STREAM_END,
		StartTime: startTime.UnixMilli(),
		ProdIndex: parNum,
	}
	encoded, _, err := h.epochMarkerSerde.Encode(marker)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.streamPusher.MsgChan <- sharedlog_stream.PayloadToPush{
		Payload:    encoded,
		IsControl:  true,
		Partitions: []uint8{parNum},
		Mark:       commtypes.STREAM_END,
	}
	return nil
}

// func (h *nexmarkSourceHandler) genFirstChkpt(ctx context.Context,
//
//	finalOutputTopicNames []string, parNum uint8,
//
// ) error {
func (h *nexmarkSourceHandler) genFirstChkpt(parNum uint8) error {
	marker := commtypes.EpochMarker{
		Mark:      commtypes.CHKPT_MARK,
		StartTime: 1,
		ProdIndex: parNum,
	}
	encoded, _, err := h.epochMarkerSerde.Encode(marker)
	if err != nil {
		return err
	}
	h.streamPusher.MsgChan <- sharedlog_stream.PayloadToPush{
		Payload:    encoded,
		IsControl:  true,
		Partitions: []uint8{parNum},
		Mark:       commtypes.CHKPT_MARK,
	}
	// return h.rcm.ResetCheckPointCount(ctx, finalOutputTopicNames)
	return nil
}

func (h *nexmarkSourceHandler) flush(ctx context.Context, stream *sharedlog_stream.SizableShardedSharedLogStream) {
	if h.bufPush {
		_, err := stream.Flush(ctx, commtypes.EmptyProducerId)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[Error] Flush failed: %v\n", err)
		}
	}
}
