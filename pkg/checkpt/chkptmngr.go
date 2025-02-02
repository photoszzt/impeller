package checkpt

import (
	"context"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func GetChkptMngrAddr() []string {
	raw_addr := os.Getenv("CHKPT_MNGR_ADDR")
	return strings.Split(raw_addr, ",")
}

func PrepareChkptClientGrpc() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	retryPolicy := `{
		"methodConfig": [{
		  "name": [{"service": "checkpt.ChkptMngr"}],
		  "waitForReady": true,
		  "retryPolicy": {
			  "MaxAttempts": 4,
			  "InitialBackoff": ".002s",
			  "MaxBackoff": ".01s",
			  "BackoffMultiplier": 2.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(retryPolicy))
	mngr_addr := GetChkptMngrAddr()
	log.Info().Strs("chkpt mngr addr", mngr_addr).Str("connected to", mngr_addr[0])
	return grpc.Dial(mngr_addr[0], opts...)
	// log.Info().Str("connected to", engine1)
	// return grpc.Dial(engine1, opts...)
}

type CheckpointManagerServer struct {
	UnimplementedChkptMngrServer

	tpCounts sync.Map
	ended    atomic.Bool
}

func NewCheckpointManagerServer() *CheckpointManagerServer {
	s := CheckpointManagerServer{}
	s.ended.Store(false)
	return &s
}

func (s *CheckpointManagerServer) Init(ctx context.Context, in *FinMsg) (*emptypb.Empty, error) {
	for _, tp := range in.TopicNames {
		v := new(uint32)
		s.tpCounts.Store(tp, v)
	}
	return &emptypb.Empty{}, nil
}

func (s *CheckpointManagerServer) ResetCheckpointCount(ctx context.Context, in *FinMsg) (*emptypb.Empty, error) {
	for _, tp := range in.TopicNames {
		v, ok := s.tpCounts.Load(tp)
		if !ok {
			return nil, common_errors.ErrTopicNotFound
		}
		c := v.(*uint32)
		atomic.StoreUint32(c, 0)
	}
	return &emptypb.Empty{}, nil
}

func (s *CheckpointManagerServer) ChkptMngrEnded(context.Context, *emptypb.Empty) (*Ended, error) {
	e := s.ended.Load()
	return &Ended{Ended: e}, nil
}

func (s *CheckpointManagerServer) FinishChkpt(ctx context.Context, in *FinMsg) (*emptypb.Empty, error) {
	for _, tp := range in.TopicNames {
		v, ok := s.tpCounts.Load(tp)
		if !ok {
			return nil, common_errors.ErrTopicNotFound
		}
		c := v.(*uint32)
		atomic.AddUint32(c, 1)
	}
	return &emptypb.Empty{}, nil
}

func (s *CheckpointManagerServer) ReqChkmngrEndedIfNot(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	ended := s.ended.Load()
	if !ended {
		s.ended.CompareAndSwap(false, true)
	}
	return &emptypb.Empty{}, nil
}

func (s *CheckpointManagerServer) CheckChkptFinish(ctx context.Context, in *CheckFinMsg) (*ChkptFinished, error) {
	allFinished := true
	for idx, tp := range in.TopicNames {
		v, ok := s.tpCounts.Load(tp)
		if !ok {
			return &ChkptFinished{Fin: false}, common_errors.ErrTopicNotFound
		}
		c := v.(*uint32)
		count := atomic.LoadUint32(c)
		parNum := in.Pars[idx]
		if count != uint32(parNum) {
			allFinished = false
			break
		}
	}
	return &ChkptFinished{Fin: allFinished}, nil
}

/*
func (s *CheckpointManagerServer) WaitForChkptFinish(
	finalOutputTopicNames []string,
	finalNumOutPartition []uint8,
) (bool, error) {
	for {
		allFinished := true
		for idx, tp := range finalOutputTopicNames {
			v, ok := s.tpCounts.Load(tp)
			if !ok {
				return false, common_errors.ErrTopicNotFound
			}
			c := v.(*uint32)
			count := atomic.LoadUint32(c)
			parNum := finalNumOutPartition[idx]
			if count != uint32(parNum) {
				allFinished = false
				break
			}
		}
		if allFinished {
			return false, nil
		}
		ended := s.ended.Load()
		if ended {
			return true, nil
		}
		time.Sleep(5 * time.Millisecond)
	}
}
*/
