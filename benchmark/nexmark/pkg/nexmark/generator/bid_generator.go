package generator

import (
	"context"
	"fmt"
	"math/bits"
	"math/rand"
	"time"

	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
)

const (
	HOT_AUCTION_RATIO  uint32 = 100
	HOT_BIDDER_RATIO   uint32 = 100
	HOT_CHANNELS_RATIO uint32 = 2
	CHANNELS_NUMBER    uint32 = 10_000
)

var (
	HOT_CHANNELS      = [4]string{"Google", "Facebook", "Baidu", "Apple"}
	HOT_URLS          = [4]string{getBaseUrl(), getBaseUrl(), getBaseUrl(), getBaseUrl()}
	CHANNEL_URL_CACHE = map[uint32]*ChannelUrl{}
)

type ChannelUrl struct {
	channel string
	url     string
}

func NextBid(ctx context.Context, eventId uint64, random *rand.Rand, timestamp int64, config *GeneratorConfig, bidUrlCache map[uint32]*ChannelUrl) (*ntypes.Bid, error) {
	auction := uint64(0)
	if random.Intn(int(config.Configuration.HotAuctionRatio)) > 0 {
		auction = LastBase0AuctionId(config, eventId) / uint64(HOT_AUCTION_RATIO) * uint64(HOT_AUCTION_RATIO)
	} else {
		auction = NextBase0AuctionId(eventId, random, config)
	}
	auction += FIRST_AUCTION_ID

	bidder := uint64(0)
	if random.Intn(int(config.Configuration.HotBiddersRatio)) > 0 {
		bidder = LastBase0PersonId(config, eventId) / uint64(HOT_AUCTION_RATIO) * uint64(HOT_AUCTION_RATIO)
	} else {
		bidder = NextBase0PersonId(eventId, random, config)
	}
	bidder += FIRST_PERSON_ID

	price := NextPrice(random)
	channel := ""
	url := ""
	if random.Intn(int(HOT_CHANNELS_RATIO)) > 0 {
		i := random.Intn(len(HOT_CHANNELS))
		channel = HOT_CHANNELS[i]
		url = HOT_URLS[i]
	} else {
		k := uint32(random.Intn(int(CHANNELS_NUMBER)))
		channelAndUrl, ok := CHANNEL_URL_CACHE[k]
		if !ok {
			burl := getBaseUrl()
			if random.Intn(10) > 0 {
				url = burl + "&channel_id=" + fmt.Sprint(bits.Reverse32(k))
			}
			channel = "channel-" + fmt.Sprint(k)
			bidUrlCache[k] = &ChannelUrl{
				channel: channel,
				url:     url,
			}
		} else {
			channel = channelAndUrl.channel
			url = channelAndUrl.url
		}
	}
	bidder += FIRST_PERSON_ID

	currentSize := 8 + 8 + 8 + 8
	extra := NextExtra(random, uint32(currentSize), config.Configuration.AvgBidByteSize)
	return &ntypes.Bid{
		Auction:  auction,
		Bidder:   bidder,
		Price:    price,
		Channel:  channel,
		Url:      url,
		DateTime: timestamp,
		Extra:    extra,
	}, nil
}

func getBaseUrl() string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	return "https://www.nexmark.com" +
		NextString(random, 5) + "/" +
		NextString(random, 5) + "/" +
		NextString(random, 5) + "/" +
		"item.htm?query=1"
}
