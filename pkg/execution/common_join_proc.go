package execution

import (
	"fmt"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/stats"
)

type CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR any] struct {
	arg1 *JoinProcArgs[KInL, VInL, KOutL, VOutL]
	arg2 *JoinProcArgs[KInR, VInR, KOutR, VOutR]
	proc_interface.BaseConsumersProducers
	pauseFuncTime  stats.StatsCollector[int64]
	resumeFuncTime stats.StatsCollector[int64]
	chkPtBtwTime   stats.StatsCollector[int64]
}

func NewCommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR any](
	arg1 *JoinProcArgs[KInL, VInL, KOutL, VOutL],
	arg2 *JoinProcArgs[KInR, VInR, KOutR, VOutR],
	ss proc_interface.BaseConsumersProducers,
) *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR] {
	return &CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]{
		arg1:                   arg1,
		arg2:                   arg2,
		BaseConsumersProducers: ss,
		pauseFuncTime: stats.NewStatsCollector[int64](fmt.Sprintf("join_pause_us_%d", arg1.SubstreamNum()),
			stats.DEFAULT_COLLECT_DURATION),
		resumeFuncTime: stats.NewStatsCollector[int64](fmt.Sprintf("join_resume_us_%d", arg1.SubstreamNum()),
			stats.DEFAULT_COLLECT_DURATION),
		chkPtBtwTime: stats.NewStatsCollector[int64](fmt.Sprintf("joinBtwTwoChkpt_ms_%d", arg1.SubstreamNum()),
			stats.DEFAULT_COLLECT_DURATION),
	}
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) OutputRemainingStats() {
	// c.arg1.procLat.PrintRemainingStats()
	// c.arg2.procLat.PrintRemainingStats()
	c.pauseFuncTime.PrintRemainingStats()
	c.resumeFuncTime.PrintRemainingStats()
	c.chkPtBtwTime.PrintRemainingStats()
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) SetRecordFinishFunc(recordFinishFunc exactly_once_intr.RecordPrevInstanceFinishFunc) {
	c.arg1.SetRecordFinishFunc(recordFinishFunc)
	c.arg2.SetRecordFinishFunc(recordFinishFunc)
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc) {
	c.arg1.SetTrackParFunc(trackParFunc)
	c.arg2.SetTrackParFunc(trackParFunc)
}

// arg1 and arg2 shared the same param for the following functions
func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) RecordFinishFunc() exactly_once_intr.RecordPrevInstanceFinishFunc {
	return c.arg1.RecordFinishFunc()
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) TrackParFunc() exactly_once_intr.TrackProdSubStreamFunc {
	return c.arg1.TrackParFunc()
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) FuncName() string {
	return c.arg1.FuncName()
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) CurEpoch() uint16 {
	return c.arg1.CurEpoch()
}

func (c *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) SubstreamNum() uint8 {
	return c.arg1.SubstreamNum()
}
