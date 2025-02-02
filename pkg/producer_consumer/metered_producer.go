package producer_consumer

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/utils/syncutils"
	"sync/atomic"
	"time"
)

func checkMeasureSink() bool {
	measure_str := os.Getenv("MEASURE_SINK")
	measure := measure_str == "true" || measure_str == "1"
	return measure
}

type ConcurrentMeteredSink struct {
	mu syncutils.Mutex

	producer           *ShardedSharedLogStreamProducer
	produceTp          *stats.ConcurrentThroughputCounter
	lat                *stats.ConcurrentStatsCollector[int64]
	eventTimeLatencies []int   // protected by mu
	eventTs            []int64 // protected by mu
	warmup             stats.WarmupGoroutineSafe
	ctrlCount          uint32
	measure            bool
}

var _ = MeteredProducerIntr(&ConcurrentMeteredSink{})

func NewConcurrentMeteredSyncProducer(sink *ShardedSharedLogStreamProducer, warmup time.Duration) (*ConcurrentMeteredSink, error) {
	sink_name := fmt.Sprintf("%s_sink", sink.TopicName())
	// stats_dir := getStatsDir()
	// err := os.MkdirAll(stats_dir, 0755)
	// if err != nil {
	// 	return nil, err
	// }
	return &ConcurrentMeteredSink{
		producer: sink,
		lat: stats.NewConcurrentStatsCollector[int64](sink_name,
			stats.DEFAULT_COLLECT_DURATION),
		produceTp: stats.NewConcurrentThroughputCounter(sink_name,
			stats.DEFAULT_COLLECT_DURATION),
		measure:            checkMeasureSink(),
		warmup:             stats.NewWarmupGoroutineSafeChecker(warmup),
		eventTimeLatencies: make([]int, 0, 4096),
		eventTs:            make([]int64, 0, 4096),
	}, nil
}

func (s *ConcurrentMeteredSink) MarkFinalOutput() {
	s.producer.MarkFinalOutput()
}

func (s *ConcurrentMeteredSink) IsFinalOutput() bool {
	return s.producer.IsFinalOutput()
}

func (s *ConcurrentMeteredSink) StartWarmup() {
	if s.measure {
		s.warmup.StartWarmup()
	}
}

func (s *ConcurrentMeteredSink) SetLastMarkerSeq(lastMarkerSeq uint64) {
	s.producer.SetLastMarkerSeq(lastMarkerSeq)
}

// func (s *ConcurrentMeteredSink) GetEventTimeLatency() []int {
// 	return s.eventTimeLatencies
// }

func (s *ConcurrentMeteredSink) GetCount() uint64 {
	return s.produceTp.GetCount()
}

func (s *ConcurrentMeteredSink) InitFlushTimer() {}

func (s *ConcurrentMeteredSink) ProduceData(ctx context.Context, msgSer commtypes.MessageSerialized, parNum uint8) error {
	if s.IsFinalOutput() {
		procStart := time.Now()
		ts := msgSer.TimestampMs
		if ts != 0 {
			curTs := procStart.UnixMilli()
			els := int(curTs - ts)
			s.mu.Lock()
			s.eventTimeLatencies = append(s.eventTimeLatencies, els)
			s.eventTs = append(s.eventTs, ts)
			s.mu.Unlock()
		}
	}
	assignInjTime(&msgSer)
	s.produceTp.Tick(1)
	// if s.measure {
	// 	s.warmup.Check()
	// 	if s.warmup.AfterWarmup() {
	// 		if s.isFinalOutput {
	// 			procStart := time.Now()
	// 			ts, err := extractEventTs(&msg)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			if ts != 0 {
	// 				els := int(procStart.UnixMilli() - ts)
	// 				s.mu.Lock()
	// 				s.eventTimeLatencies = append(s.eventTimeLatencies, els)
	// 				s.mu.Unlock()
	// 			}
	// 		}
	// 	}
	// }
	procStart := stats.TimerBegin()
	err := s.producer.ProduceData(ctx, msgSer, parNum)
	elapsed := stats.Elapsed(procStart).Microseconds()
	s.lat.AddSample(elapsed)
	return err
}

func (s *ConcurrentMeteredSink) ProduceCtrlMsg(ctx context.Context, msg *commtypes.RawMsgAndSeq, parNums []uint8) (int, error) {
	c, err := s.producer.ProduceCtrlMsg(ctx, msg, parNums)
	atomic.AddUint32(&s.ctrlCount, uint32(c))
	return c, err
}

func (s *ConcurrentMeteredSink) TopicName() string   { return s.producer.TopicName() }
func (s *ConcurrentMeteredSink) Name() string        { return s.producer.Name() }
func (s *ConcurrentMeteredSink) SetName(name string) { s.producer.SetName(name) }
func (s *ConcurrentMeteredSink) Flush(ctx context.Context) (uint32, error) {
	return s.producer.Flush(ctx)
}

func (s *ConcurrentMeteredSink) ConfigExactlyOnce(rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	s.producer.ConfigExactlyOnce(rem, guarantee)
}

func (s *ConcurrentMeteredSink) Stream() sharedlog_stream.Stream {
	return s.producer.Stream()
}

func (s *ConcurrentMeteredSink) GetInitialProdSeqNum(substreamNum uint8) uint64 {
	return s.producer.GetInitialProdSeqNum(substreamNum)
}
func (s *ConcurrentMeteredSink) ResetInitialProd() { s.producer.ResetInitialProd() }
func (s *ConcurrentMeteredSink) OutputRemainingStats() {
	s.lat.PrintRemainingStats()
	s.producer.OutputRemainingStats()
}

func (s *ConcurrentMeteredSink) GetEventTimeLatency() []int {
	s.mu.Lock()
	ret := s.eventTimeLatencies
	s.mu.Unlock()
	return ret
}

func (s *ConcurrentMeteredSink) GetEventTs() []int64 {
	s.mu.Lock()
	ret := s.eventTs
	s.mu.Unlock()
	return ret
}

func (s *ConcurrentMeteredSink) NumCtrlMsg() uint32 {
	return atomic.LoadUint32(&s.ctrlCount)
}

func (s *ConcurrentMeteredSink) SetFlushCallback(cb exactly_once_intr.FlushCallbackFunc) {
	s.producer.SetFlushCallback(cb)
}

type MeteredProducer struct {
	producer           *ShardedSharedLogStreamProducer
	eventTimeLatencies []int
	eventTs            []int64
	latencies          stats.StatsCollector[int64]
	produceTp          stats.ThroughputCounter
	warmup             stats.Warmup
	ctrlCount          uint32
	measure            bool
}

func NewMeteredProducer(sink *ShardedSharedLogStreamProducer, warmup time.Duration) (*MeteredProducer, error) {
	sink_name := fmt.Sprintf("%s_sink", sink.TopicName())
	return &MeteredProducer{
		producer:           sink,
		eventTimeLatencies: make([]int, 0, 4096),
		latencies:          stats.NewStatsCollector[int64](sink_name, stats.DEFAULT_COLLECT_DURATION),
		produceTp:          stats.NewThroughputCounter(sink_name, stats.DEFAULT_COLLECT_DURATION),
		// eventTimeSample: stats.NewStatsCollector[int64](sink_name+"_ets", stats.DEFAULT_COLLECT_DURATION),
		measure: checkMeasureSink(),
		warmup:  stats.NewWarmupChecker(warmup),
		eventTs: make([]int64, 0, 4096),
	}, nil
}

func (s *MeteredProducer) SetLastMarkerSeq(seq uint64) {
	s.producer.SetLastMarkerSeq(seq)
}

func (s *MeteredProducer) InitFlushTimer() {}

func (s *MeteredProducer) MarkFinalOutput() {
	s.producer.MarkFinalOutput()
}

func (s *MeteredProducer) SetFlushCallback(cb exactly_once_intr.FlushCallbackFunc) {
	s.producer.SetFlushCallback(cb)
}

func (s *MeteredProducer) IsFinalOutput() bool {
	return s.producer.IsFinalOutput()
}

func (s *MeteredProducer) StartWarmup() {
	if s.measure {
		s.warmup.StartWarmup()
	}
}

func (s *MeteredProducer) ProduceCtrlMsg(ctx context.Context, msg *commtypes.RawMsgAndSeq, parNums []uint8) (int, error) {
	ctrlCnt, err := s.producer.ProduceCtrlMsg(ctx, msg, parNums)
	s.ctrlCount += uint32(ctrlCnt)
	return ctrlCnt, err
}

func (s *MeteredProducer) ProduceData(ctx context.Context, msg commtypes.MessageSerialized, parNum uint8) error {
	if s.IsFinalOutput() {
		procStart := time.Now()
		ts := msg.TimestampMs
		if msg.TimestampMs != 0 {
			curTs := procStart.UnixMilli()
			els := curTs - ts
			s.eventTimeLatencies = append(s.eventTimeLatencies, int(els))
			s.eventTs = append(s.eventTs, curTs)
			// if len(s.eventTimeLatencies) < cap(s.eventTimeLatencies) {
			// 	s.eventTimeLatencies = append(s.eventTimeLatencies, int(els))
			// 	// s.eventTimeSample.AddSample(els)
			// } else {
			// 	data := fmt.Sprintf("%v", s.eventTimeLatencies)
			// }
		}
	}
	assignInjTime(&msg)
	s.produceTp.Tick(1)

	// if s.measure {
	// 	s.warmup.Check()
	// 	if s.warmup.AfterWarmup() {
	// 		if s.isFinalOutput {
	// 			procStart := time.Now()
	// 			ts, err := extractEventTs(&msg)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			if ts != 0 {
	// 				els := int(procStart.UnixMilli() - ts)
	// 				s.eventTimeLatencies = append(s.eventTimeLatencies, els)
	// 			}
	// 		}
	// 	}
	// }
	procStart := stats.TimerBegin()
	err := s.producer.ProduceDataNoLock(ctx, msg, parNum)
	elapsed := stats.Elapsed(procStart).Microseconds()
	s.latencies.AddSample(elapsed)
	return err
}

func (s *MeteredProducer) TopicName() string   { return s.producer.TopicName() }
func (s *MeteredProducer) Name() string        { return s.producer.Name() }
func (s *MeteredProducer) SetName(name string) { s.producer.SetName(name) }
func (s *MeteredProducer) Flush(ctx context.Context) (uint32, error) {
	return s.producer.FlushNoLock(ctx)
}

func (s *MeteredProducer) ConfigExactlyOnce(rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	s.producer.ConfigExactlyOnce(rem, guarantee)
}

func (s *MeteredProducer) Stream() sharedlog_stream.Stream {
	return s.producer.Stream()
}

func (s *MeteredProducer) GetInitialProdSeqNum(substreamNum uint8) uint64 {
	return s.producer.GetInitialProdSeqNum(substreamNum)
}

func (s *MeteredProducer) ResetInitialProd() {
	s.producer.ResetInitialProd()
}

func (s *MeteredProducer) OutputRemainingStats() {
	// s.latencies.PrintRemainingStats()
	// s.eventTimeSample.PrintRemainingStats()
	s.producer.OutputRemainingStats()
}

func (s *MeteredProducer) GetEventTimeLatency() []int {
	return s.eventTimeLatencies
}

func (s *MeteredProducer) GetEventTs() []int64 {
	return s.eventTs
}

func (s *MeteredProducer) GetCount() uint64 {
	return s.produceTp.GetCount()
}

func (s *MeteredProducer) NumCtrlMsg() uint32 {
	return s.ctrlCount
}
