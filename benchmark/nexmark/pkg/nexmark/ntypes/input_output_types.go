package ntypes

import (
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DEFAULT_AVG_PERSON_SIZE = 200
	DEFAULT_AVG_AUC_SIZE    = 500
	DEFAULT_AVG_BID_SIZE    = 100
)

// entry with slashes is assigned to NexMarkConfig
type NexMarkConfigInput struct {
	TopicName              string        `json:"topic_name"`
	RateShape              string        `json:"rate_shape"` //
	AppId                  string        `json:"aid"`
	Engine1                string        `json:"engine1,omitempty"`
	EventsNum              uint64        `json:"events_num"`  //
	RatePeriod             time.Duration `json:"rate_period"` //
	FinalOutTpNames        []string      `json:"finOutTpNs,omitempty"`
	FirstEventRate         uint32        `json:"first_event_rate"` //
	NextEventRate          uint32        `json:"next_event_rate"`  //
	BidAvgSize             uint32        `json:"bid_avg_size"`     //
	PersonAvgSize          uint32        `json:"person_avg_size"`  //
	AuctionAvgSize         uint32        `json:"auction_avg_size"` //
	Duration               uint32        `json:"duration"`
	PersonProportion       uint32        `json:"person_proportion"`         //
	AuctionProportion      uint32        `json:"auction_proportion"`        //
	BidProportion          uint32        `json:"bid_proportion"`            //
	BidHotRatioAuctions    uint32        `json:"bid_hot_ratio_auctions"`    //
	BidHotRatioBidders     uint32        `json:"bid_hot_ratio_bidders"`     //
	AuctionHotRatioSellers uint32        `json:"auction_hot_ratio_sellers"` //
	FlushMs                uint32        `json:"flushms"`
	BufMaxSize             uint32        `json:"bufMaxSize"`
	CommitEveryMs          uint32        `json:"commEveryMs,omitempty"`
	RateLimited            bool          `json:"rate_limited"` //
	WaitForEndMark         bool          `json:"waitEnd"`
	SerdeFormat            uint8         `json:"serde_format"`
	NumOutPartition        uint8         `json:"numOutPar,omitempty"`
	ParNum                 uint8         `json:"parNum,omitempty"`
	NumSrcInstance         uint8         `json:"nSrcIns,omitempty"`
	GuaranteeMth           uint8         `json:"gua,omitempty"`
}

func NewNexMarkConfigInput(topicName string, serdeFormat commtypes.SerdeFormat) *NexMarkConfigInput {
	return &NexMarkConfigInput{
		TopicName:              topicName,
		Duration:               0,
		RateShape:              "square",
		RatePeriod:             time.Duration(10) * time.Second,
		RateLimited:            false,
		FirstEventRate:         100,
		NextEventRate:          100,
		PersonAvgSize:          DEFAULT_AVG_PERSON_SIZE,
		AuctionAvgSize:         DEFAULT_AVG_AUC_SIZE,
		BidAvgSize:             DEFAULT_AVG_BID_SIZE,
		PersonProportion:       1,
		AuctionProportion:      3,
		BidProportion:          46,
		BidHotRatioAuctions:    2,
		BidHotRatioBidders:     4,
		AuctionHotRatioSellers: 4,
		EventsNum:              0,
		SerdeFormat:            uint8(serdeFormat),
	}
}

func ConvertToNexmarkConfiguration(config *NexMarkConfigInput) (*NexMarkConfig, error) {
	rateUnit, err := StrToRateShape(config.RateShape)
	if err != nil {
		return nil, err
	}
	nexmarkConfig := NewNexMarkConfig()
	nexmarkConfig.RateShape = rateUnit
	nexmarkConfig.RatePeriodSec = uint32(config.RatePeriod.Seconds())
	nexmarkConfig.IsRateLimited = config.RateLimited
	nexmarkConfig.FirstEventRate = uint64(config.FirstEventRate)
	nexmarkConfig.NextEventRate = uint64(config.NextEventRate)
	nexmarkConfig.AvgPersonByteSize = config.PersonAvgSize
	nexmarkConfig.AvgAuctionByteSize = config.AuctionAvgSize
	nexmarkConfig.AvgBidByteSize = config.BidAvgSize
	nexmarkConfig.PersonProportion = config.PersonProportion
	nexmarkConfig.AuctionProportion = config.AuctionProportion
	nexmarkConfig.BidProportion = config.BidProportion
	nexmarkConfig.HotAuctionRatio = config.BidHotRatioAuctions
	nexmarkConfig.HotBiddersRatio = config.BidHotRatioBidders
	nexmarkConfig.HotSellersRatio = config.AuctionHotRatioSellers
	nexmarkConfig.NumEvents = uint32(config.EventsNum)
	nexmarkConfig.NumEventGenerators = uint32(config.NumSrcInstance)
	return nexmarkConfig, nil
}

type GeneratorParams struct {
	FaasGateway    string
	Engine1        string
	EventsNum      uint64
	Duration       uint32
	Tps            uint32
	FlushMs        uint32
	CommitMs       uint32
	BufMaxSize     uint32
	SerdeFormat    commtypes.SerdeFormat
	WaitForEndMark bool
	GuaranteeMth   uint8
}

func (gp *GeneratorParams) InvokeSourceFunc(client *http.Client,
	srcInvokeConfig common.SrcInvokeConfig,
	response *common.FnOutput, wg *sync.WaitGroup, warmup bool,
) {
	defer wg.Done()
	avgAucBytes := DEFAULT_AVG_AUC_SIZE + srcInvokeConfig.AdditionalBytes
	avgBidBytes := DEFAULT_AVG_BID_SIZE + srcInvokeConfig.AdditionalBytes
	avgPersonBytes := DEFAULT_AVG_PERSON_SIZE + srcInvokeConfig.AdditionalBytes
	nexmarkConfig := NewNexMarkConfigInput(srcInvokeConfig.TopicName, gp.SerdeFormat)
	nexmarkConfig.Engine1 = gp.Engine1
	nexmarkConfig.FinalOutTpNames = srcInvokeConfig.FinalTpNames
	nexmarkConfig.Duration = gp.Duration
	nexmarkConfig.AppId = srcInvokeConfig.AppId
	nexmarkConfig.EventsNum = gp.EventsNum
	nexmarkConfig.FirstEventRate = gp.Tps
	nexmarkConfig.NextEventRate = gp.Tps
	nexmarkConfig.NumOutPartition = srcInvokeConfig.NumOutPartition
	nexmarkConfig.ParNum = srcInvokeConfig.InstanceID
	nexmarkConfig.NumSrcInstance = srcInvokeConfig.NumSrcInstance
	nexmarkConfig.FlushMs = gp.FlushMs
	nexmarkConfig.WaitForEndMark = gp.WaitForEndMark
	nexmarkConfig.BufMaxSize = gp.BufMaxSize
	nexmarkConfig.GuaranteeMth = gp.GuaranteeMth
	nexmarkConfig.CommitEveryMs = gp.CommitMs
	nexmarkConfig.AuctionAvgSize = uint32(avgAucBytes)
	nexmarkConfig.BidAvgSize = uint32(avgBidBytes)
	nexmarkConfig.PersonAvgSize = uint32(avgPersonBytes)
	url := common.BuildFunctionUrl(gp.FaasGateway, "source")
	fmt.Printf("func source url is %v\n", url)
	if err := common.JsonPostRequest(client, url, srcInvokeConfig.NodeConstraint, nexmarkConfig, response); err != nil {
		log.Error().Err(err).Msg("source request failed")
	} else if !response.Success {
		log.Error().Msgf("source request failed: %s", response.Message)
	}
	fmt.Fprintf(os.Stderr, "source-%d invoke done\n", srcInvokeConfig.InstanceID)
}
