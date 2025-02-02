package common

import (
	"sharedlog-stream/pkg/commtypes"
	"time"
)

const (
	SrcConsumeTimeout = 2 * time.Millisecond
	ClientRetryTimes  = 100
	CommitDuration    = time.Duration(100) * time.Millisecond
	FlushDuration     = time.Duration(100) * time.Millisecond
)

type QueryInput struct {
	TestParams            map[string]commtypes.FailParam `json:"testParams,omitempty"`
	AppId                 string                         `json:"aid"`
	FaasGateway           string                         `json:"gateway"`
	Engine1               string                         `json:"engine1,omitempty"`
	NumOutPartitions      []uint8                        `json:"numOutPartition,omitempty"`
	NumSubstreamProducer  []uint8                        `json:"numSubstreamProducer,omitempty"`
	OutputTopicNames      []string                       `json:"outputTopicName,omitempty"`
	InputTopicNames       []string                       `json:"inputTopicName,omitempty"`
	CommitEveryMs         uint64                         `json:"commEveryMs,omitempty"`
	FlushMs               uint32                         `json:"flushMs,omitempty"`
	WarmupS               uint32                         `json:"warmup,omitempty"`
	SnapEveryS            uint32                         `json:"snapEveryS,omitempty"`
	Duration              uint32                         `json:"duration,omitempty"`
	BufMaxSize            uint32                         `json:"bufMaxSize,omitempty"`
	ScaleEpoch            uint16                         `json:"epoch"`
	WaitForEndMark        bool                           `json:"waitEnd,omitempty"`
	NumChangelogPartition uint8                          `json:"numChangelogPartition,omitempty"`
	NumInPartition        uint8                          `json:"numInPartition,omitempty"`
	ParNum                uint8                          `json:"ParNum,omitempty"`
	GuaranteeMth          uint8                          `json:"gua,omitempty"`
	SerdeFormat           uint8                          `json:"serdeFormat,omitempty"`
	TableType             uint8                          `json:"tabT,omitempty"`
}

func (q *QueryInput) Clone() QueryInput {
	return QueryInput{
		TestParams:           q.TestParams,
		AppId:                q.AppId,
		FaasGateway:          q.FaasGateway,
		Engine1:              q.Engine1,
		NumOutPartitions:     q.NumOutPartitions,
		NumSubstreamProducer: q.NumSubstreamProducer,
		OutputTopicNames:     q.OutputTopicNames,
		InputTopicNames:      q.InputTopicNames,
		CommitEveryMs:        q.CommitEveryMs,
		FlushMs:              q.FlushMs,
		WarmupS:              q.WarmupS,
		SnapEveryS:           q.SnapEveryS,
		Duration:             q.Duration,
		BufMaxSize:           q.BufMaxSize,
		ScaleEpoch:           q.ScaleEpoch,
		WaitForEndMark:       q.WaitForEndMark,
		NumInPartition:       q.NumInPartition,
		ParNum:               q.ParNum,
		GuaranteeMth:         q.GuaranteeMth,
		SerdeFormat:          q.SerdeFormat,
		TableType:            q.TableType,
	}
}

type ConfigScaleInput struct {
	Config      map[string]uint8 `json:"sg,omitempty"`
	AppId       string           `json:"aid,omitempty"`
	FuncNames   []string         `json:"fns,omitempty"`
	BufMaxSize  uint32           `json:"bufMaxSize"`
	ScaleEpoch  uint16           `json:"epoch,omitempty"`
	SerdeFormat uint8            `json:"serdeFormat,omitempty"`
	Bootstrap   bool             `json:"bs,omitempty"`
}

func (c *ConfigScaleInput) Clone() ConfigScaleInput {
	return ConfigScaleInput{
		Config:      c.Config,
		AppId:       c.AppId,
		FuncNames:   c.FuncNames,
		ScaleEpoch:  c.ScaleEpoch,
		SerdeFormat: c.SerdeFormat,
		Bootstrap:   c.Bootstrap,
		BufMaxSize:  c.BufMaxSize,
	}
}

type RedisSetupInput struct {
	FinalOutputTopicNames []string `json:"outTpNames"`
}

type RTxnMngrInput struct {
	SerdeFormat uint8 `json:"serde_format"`
}

type ChkptMngrInput struct {
	SrcTopicName string `json:"srcTpName"`
	// final stage's output partition and partition number
	FinalOutputTopicNames []string `json:"outTpNames"`
	FinalNumOutPartitions []uint8  `json:"numOutParts"`
	ChkptEveryMs          uint64   `json:"chkptEveryMs"`
	BufMaxSize            uint32   `json:"bufMaxSize"`
	SrcNumPart            uint8    `json:"srcNumPart"`
	GuaranteeMth          uint8    `json:"gua"`
	SerdeFormat           uint8    `json:"serde_format"`
}

type DumpInput struct {
	DumpDir       string `json:"dumpDir"`
	TopicName     string `json:"tp"`
	KeySerde      string `json:"keySerde"`
	ValueSerde    string `json:"valSerde"`
	NumPartitions uint8  `json:"numPartitions"`
	SerdeFormat   uint8  `json:"serdeFormat"`
}

type StreamParam struct {
	TopicName     string `json:"tp"`
	KeySerde      string `json:"keySerde"`
	ValueSerde    string `json:"valSerde"`
	NumPartitions uint8  `json:"numPartitions"`
}

type StreamParams struct {
	StreamParams []*StreamParam `json:"streamParams"`
}

type DumpStreams struct {
	DumpDir      string         `json:"dumpDir"`
	StreamParams []*StreamParam `json:"streamParams"`
	SerdeFormat  uint8          `json:"serdeFormat"`
}

type TestSourceInput struct {
	FileName    string `json:"fname"`
	SerdeFormat uint8  `json:"serdeFormat"`
}

type FnOutput struct {
	Latencies map[string][]int  `json:"latencies,omitempty"`
	Counts    map[string]uint64 `json:"counts,omitempty"`
	Message   string            `json:"message,omitempty"`
	EventTs   []int64           `json:"eventTs,omitempty"`
	Duration  float64           `json:"duration,omitempty"`
	Success   bool              `json:"success,omitempty"`
}

func GenErrFnOutput(err error) *FnOutput {
	return &FnOutput{Success: false, Message: err.Error()}
}

type SourceParam struct {
	TopicName       string `json:"topicName"`
	FileName        string `json:"fname,omitempty"`
	Duration        uint32 `json:"duration,omitempty"` // in sec
	SerdeFormat     uint8  `json:"serdeFormat"`
	NumEvents       uint32 `json:"numEvents,omitempty"`
	NumOutPartition uint8  `json:"numOutPar,omitempty"`
}

type BenchSourceParam struct {
	TopicName       string `json:"topicName"`
	FileName        string `json:"fname,omitempty"`
	Duration        uint32 `json:"duration,omitempty"`
	NumEvents       uint32 `json:"numEvents,omitempty"`
	Tps             uint32 `json:"tps"`
	WarmUpTime      uint32 `json:"wt"`
	WarmUpEvents    uint32 `json:"we"`
	FlushMs         uint32 `json:"flushMs"`
	BufMaxSize      uint32 `json:"bufMaxSize"`
	SerdeFormat     uint8  `json:"serdeFormat"`
	NumOutPartition uint8  `json:"numOutPar,omitempty"`
}

type TranProcessBenchParam struct {
	InTopicName   string `json:"inTopicName"`
	OutTopicName  string `json:"outTopicName"`
	AppId         string `json:"appId"`
	FaasGateway   string `json:"gateway"`
	CommitEveryMs uint64 `json:"commEveryMs,omitempty"`
	Duration      uint32 `json:"duration,omitempty"`
	BufMaxSize    uint32 `json:"bufMaxSize,omitempty"`
	FlushMs       uint32 `json:"flushMs,omitempty"`
	SerdeFormat   uint8  `json:"serdeFormat"`
	NumPartition  uint8  `json:"numPartition"`
}

type TestStreamParam struct {
	TopicName    string
	NumPartition uint8
}

type TestParam struct {
	InStreamParam  []TestStreamParam `json:"inTopicNames"`
	OutStreamParam []TestStreamParam `json:"outTopicNames"`
	AppId          string            `json:"appId"`
	CommitEveryMs  uint64            `json:"commEveryMs,omitempty"`
	FlushMs        uint32            `json:"flushMs,omitempty"`
	BufMaxSize     uint32            `json:"bufMaxSize,omitempty"`
	SerdeFormat    uint8             `json:"serdeFormat"`
}
