package main

import (
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common/chkpt_manager"
	"sharedlog-stream/benchmark/common/redis_setup"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/handlers"

	configscale "sharedlog-stream/benchmark/common/config_scale"

	// _ "net/http/pprof"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type funcHandlerFactory struct{}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	fmt.Fprintf(os.Stderr, "Enter nexmark: funcName: %v\n", funcName)
	switch funcName {
	case "source":
		return handlers.NewNexmarkSource(env, funcName), nil
	case "testSrc":
		return handlers.NewTestEventSource(env, funcName), nil
	case "query1":
		return handlers.NewQuery1(env, funcName), nil
	case "query2":
		return handlers.NewQuery2(env, funcName), nil
	case "q3JoinTable":
		return handlers.NewQ3JoinTableHandler(env, funcName), nil
	case "q3GroupBy":
		return handlers.NewQ3GroupByHandler(env, funcName), nil
	case "q4JoinStream":
		return handlers.NewQ4JoinStreamHandler(env, funcName), nil
	case "q46GroupBy":
		return handlers.NewQ46GroupByHandler(env, funcName), nil
	case "q4Avg":
		return handlers.NewQ4Avg(env, funcName), nil
	case "q4MaxBid":
		return handlers.NewQ4MaxBid(env, funcName), nil
	case "q5bidkeyedbyauction":
		return handlers.NewBidByAuctionHandler(env, funcName), nil
	case "q5aucbids":
		return handlers.NewQ5AuctionBids(env, funcName), nil
	case "q5maxbid":
		return handlers.NewQ5MaxBid(env, funcName), nil
	case "q6JoinStream":
		return handlers.NewQ6JoinStreamHandler(env, funcName), nil
	case "q6MaxBid":
		return handlers.NewQ6MaxBid(env, funcName), nil
	case "q6Avg":
		return handlers.NewQ6Avg(env, funcName), nil
	case "dump":
		return handlers.NewDump(env), nil
	case "q7BidByPrice":
		return handlers.NewQ7BidByPriceHandler(env, funcName), nil
	case "q7BidByWin":
		return handlers.NewQ7BidByWin(env, funcName), nil
	case "q7MaxBid":
		return handlers.NewQ7MaxBid(env, funcName), nil
	case "q7JoinMaxBid":
		return handlers.NewQ7JoinMaxBid(env, funcName), nil
	case "q8JoinStream":
		return handlers.NewQ8JoinStreamHandler(env, funcName), nil
	case "q8GroupBy":
		return handlers.NewQ8GroupByHandler(env, funcName), nil
	case "scale":
		return configscale.NewConfigScaleHandler(env), nil
	case "emptyInit":
		return handlers.NewInitEmptyQuery(env, funcName), nil
	case "subG2Empty":
		return handlers.NewSubG2Empty(env, funcName), nil
	case "subG3Empty":
		return handlers.NewSubG3Empty(env, funcName), nil
	case "lastEmpty":
		return handlers.NewLastEmptyQuery(env, funcName), nil
	case "chkptmngr":
		return chkpt_manager.NewChkptManagerHandlerGrpc(env), nil
	case "redisSetup":
		return redis_setup.NewRedisSetupHandler(env), nil
	case "fanout":
		return handlers.NewFanout(env, funcName), nil
	default:
		return nil, fmt.Errorf("unknown function name %v", funcName)
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	faas.Serve(&funcHandlerFactory{})
}
