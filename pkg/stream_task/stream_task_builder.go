package stream_task

import (
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/stats"
)

type StreamTaskBuilder struct {
	task *StreamTask
}

type (
	ResumeFuncType func(task *StreamTask, gua exactly_once_intr.GuaranteeMth)
	PauseFuncType  func(gua exactly_once_intr.GuaranteeMth) *common.FnOutput
)

type BuildStreamTask interface {
	Build() *StreamTask
	AppProcessFunc(process ProcessFunc) BuildStreamTask
	InitFunc(i func(task *StreamTask)) BuildStreamTask
	PauseFunc(p PauseFuncType) BuildStreamTask
	ResumeFunc(r ResumeFuncType) BuildStreamTask
	HandleErrFunc(he func() error) BuildStreamTask
	MarkFinalStage() BuildStreamTask
}

func NewStreamTaskBuilder() BuildStreamTask {
	t := &StreamTaskBuilder{
		task: &StreamTask{
			pauseFunc:        nil,
			resumeFunc:       nil,
			initFunc:         nil,
			HandleErrFunc:    nil,
			flushStageTime:   stats.NewPrintLogStatsCollector[int64]("flushStage(us)"),
			flushAtLeastOne:  stats.NewPrintLogStatsCollector[int64]("flushAtLeastOne(us)"),
			commitTxnAPITime: stats.NewPrintLogStatsCollector[int64]("commitTxnAPITime(us)"),
			sendOffsetTime:   stats.NewPrintLogStatsCollector[int64]("sendOffsetTime(us)"),
			txnCommitTime:    stats.NewPrintLogStatsCollector[int64]("txnCommitTime(us)"),
			markPartUs:       stats.NewPrintLogStatsCollector[int64]("markPart(us)"),
			epochMarkTime:    stats.NewPrintLogStatsCollector[int64]("epochMarkTime(us)"),

			markEpochAppend:   stats.NewPrintLogStatsCollector[int64]("appendEpochMark(us)"),
			markEpochPrepare:  stats.NewPrintLogStatsCollector[int64]("markEpochPrepare(us)"),
			waitPrevTxnInCmt:  stats.NewPrintLogStatsCollector[int64]("waitPrevTxnInCmt"),
			waitPrevTxnInPush: stats.NewPrintLogStatsCollector[int64]("waitPrevTxnInPush"),

			markerSize:    stats.NewPrintLogStatsCollector[int]("epochMarkerSize(B)"),
			producerFlush: stats.NewPrintLogStatsCollector[int64]("producerFlush"),
			kvcFlush:      stats.NewPrintLogStatsCollector[int64]("kvcFlush"),
			wscFlush:      stats.NewPrintLogStatsCollector[int64]("wscFlush"),
			finishChkpt:   stats.NewStatsCollector[int64]("finishChkpt", stats.DEFAULT_COLLECT_DURATION),

			txnCounter:                 stats.NewCounter("txnCount"),
			waitedInCmtCounter:         stats.NewCounter("waitedPrevTxnInCmtCount"),
			waitedInPushCounter:        stats.NewAtomicCounter("waitedPrevTxnInPushCount"),
			appendedMetaInCmtCounter:   stats.NewCounter("appendedMetaInCmtCount"),
			appendedMetaInPushCounter:  stats.NewAtomicCounter("appendedMetaInPushCount"),
			waitAndAppendInCmtCounter:  stats.NewCounter("waitAndAppendMetaInCmtCount"),
			waitAndAppendInPushCounter: stats.NewAtomicCounter("waitAndAppendMetaInPushCount"),
			epochMarkTimes:             0,
			isFinalStage:               false,
		},
	}
	t.task.waitedInPushCounter.InitCounter()
	t.task.appendedMetaInPushCounter.InitCounter()
	t.task.waitAndAppendInPushCounter.InitCounter()
	return t
}

func (b *StreamTaskBuilder) AppProcessFunc(process ProcessFunc) BuildStreamTask {
	b.task.appProcessFunc = process
	return b
}

func (b *StreamTaskBuilder) Build() *StreamTask {
	return b.task
}

func (b *StreamTaskBuilder) PauseFunc(p PauseFuncType) BuildStreamTask {
	b.task.pauseFunc = p
	return b
}

func (b *StreamTaskBuilder) ResumeFunc(r ResumeFuncType) BuildStreamTask {
	b.task.resumeFunc = r
	return b
}

func (b *StreamTaskBuilder) InitFunc(i func(task *StreamTask)) BuildStreamTask {
	b.task.initFunc = i
	return b
}

func (b *StreamTaskBuilder) HandleErrFunc(he func() error) BuildStreamTask {
	b.task.HandleErrFunc = he
	return b
}

func (b *StreamTaskBuilder) MarkFinalStage() BuildStreamTask {
	b.task.isFinalStage = true
	return b
}
