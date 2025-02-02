package ntypes

type NexMarkConfig struct {
	/// Shape of event rate curve.
	RateShape RateShape `json:"rateShape"`

	/// Initial overall event rate in `RateUnit`.
	FirstEventRate uint64 `json:"firstEventRate"`

	/// Next overall event rate in `RateUnit`
	NextEventRate uint64 `json:"nextEventRate"`

	/// Length of occasional delay to impose on events (in seconds).
	OccasionalDelaySec uint64 `json:"occasionalDelaySec"`

	/// Number of events in out-of-order groups. 1 implies no out-of-order events. 1000 implies every
	/// 1000 events per generator are emitted in pseudo-random order.
	OutOfOrderGroupSize uint64 `json:"outOfOrderGroupSize"`

	/// Probability that an event will be delayed by delayS.
	ProbDelayedEvent float64 `json:"probDelayedEvent"`

	/// Unit for rates
	RateUnit RateUnit `json:"rateUnit"`

	/// Overall period of rate shape, in seconds.
	RatePeriodSec uint32 `json:"ratePeriodSec"`

	/// Time in seconds to preload the subscription with data, at the initial input rate of the
	/// pipeline
	PreloadSeconds uint32 `json:"preloadSeconds"`

	/// Timeout for stream pipelines to stop in seconds
	StreamTimeout uint32 `json:"streamTimeout"`

	/// Number of events to generate. If zero, generate as many as possible without overflowing
	/// internal counters etc.
	NumEvents uint32 `json:"numEvents"`

	/// Number of event generators to use. Each generates events in its own timeline.
	NumEventGenerators uint32 `json:"numEventGenerators"`

	/// Person Proportion.
	PersonProportion uint32 `json:"personProportion"`

	/// Auction Proportion.
	AuctionProportion uint32 `json:"auctionProportion"`

	/// Bid Proportion
	BidProportion uint32 `json:"bidProportion"`

	/// Average idealized size of a 'new person' event, in bytes.
	AvgPersonByteSize uint32 `json:"avgPersonByteSize"`

	/// Average idealized size of a 'new auction' event, in bytes.
	AvgAuctionByteSize uint32 `json:"avgAuctionByteSize"`

	/// Average idealized size of a 'bid' event, in bytes.
	AvgBidByteSize uint32 `json:"avgBidByteSize"`

	/// Ratio of bids to 'hot' auctions compared to all other auctions.
	HotAuctionRatio uint32 `json:"hotAuctionRatio"`

	/// Ratio of auctions for 'hot' sellers compared to all other people.
	HotSellersRatio uint32 `json:"hotSellersRatio"`

	/// Ratio of bids for 'hot' bidders compared to all other people.
	HotBiddersRatio uint32 `json:"hotBiddersRatio"`

	/// Window size, in seconds, for queries 3, 5, 7 and 8.
	WindowSizeSec uint32 `json:"windowSizeSec"`

	/// Sliding window period, in seconds, for query 5.
	WindowPeriodSec uint32 `json:"windowPeriodSec"`

	/// Number of seconds to hold back events according to their reported timestamp.
	WatermarkHoldbackSec uint32 `json:"watermarkHoldbackSec"`

	/// Average number of auction which should be inflight at any time, per generator.
	NumInFlightAuctions uint32 `json:"numInFlightAuctions"`

	/// Maximum number of people to consider as active for placing auctions or bids.
	NumActivePeople uint32 `json:"numActivePeople"`

	/// If true, and in streaming mode, generate events only when they are due according to their
	/// timestamp.
	IsRateLimited bool `json:"isRateLimited"`

	/// If true, use wallclock time as event time. Otherwise, use a deterministic time in the past so
	/// that multiple runs will see exactly the same event streams and should thus have exactly the
	/// same results.
	UseWallclockEventTime bool `json:"useWallclockEventTime"`
}

func NewNexMarkConfig() *NexMarkConfig {
	return &NexMarkConfig{
		NumEvents:             0,
		NumEventGenerators:    1,
		RateShape:             SQUARE,
		FirstEventRate:        10000,
		NextEventRate:         10000,
		RateUnit:              PER_SECOND,
		RatePeriodSec:         600,
		PreloadSeconds:        0,
		StreamTimeout:         240,
		IsRateLimited:         false,
		UseWallclockEventTime: false,
		PersonProportion:      1,
		AuctionProportion:     3,
		BidProportion:         46,
		AvgPersonByteSize:     200,
		AvgAuctionByteSize:    500,
		AvgBidByteSize:        100,
		HotAuctionRatio:       2,
		HotSellersRatio:       4,
		HotBiddersRatio:       4,
		WindowSizeSec:         10,
		WindowPeriodSec:       5,
		WatermarkHoldbackSec:  0,
		NumInFlightAuctions:   100,
		NumActivePeople:       1000,
		OccasionalDelaySec:    3,
		ProbDelayedEvent:      0.1,
		OutOfOrderGroupSize:   1,
	}
}
