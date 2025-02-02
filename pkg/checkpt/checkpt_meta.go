package checkpt

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/redis_client"
	"strconv"
	"time"

	"github.com/go-redis/redis/v9"
)

const (
	CHKPT_META_NODE   = 0
	REQ_CHKMNGR_ENDED = "req_chkmngr_ended"
	CHKPT_MNGR_ENDED  = "chkmngr_ended"
)

type RedisChkptManager struct {
	rds []*redis.Client
}

func NewRedisChkptManager() RedisChkptManager {
	rcm := RedisChkptManager{
		rds: redis_client.GetRedisClients(),
	}
	return rcm
}

func NewRedisChkptManagerFromClients(rds []*redis.Client) RedisChkptManager {
	rcm := RedisChkptManager{
		rds: rds,
	}
	return rcm
}

func (c *RedisChkptManager) StoreInitSrcLogoff(ctx context.Context, srcLogOff []commtypes.TpLogOff, numSrcPar uint8) error {
	l := len(c.rds)
	for i := uint8(0); i < numSrcPar; i++ {
		idx := int(i) % l
		err := c.rds[idx].RPush(ctx, srcLogOff[i].Tp, srcLogOff[i].LogOff).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *RedisChkptManager) InitReqRes(ctx context.Context) error {
	var err error
	for {
		err = c.rds[CHKPT_META_NODE].Ping(ctx).Err()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	err = c.rds[CHKPT_META_NODE].Set(ctx, REQ_CHKMNGR_ENDED, 0, 0).Err()
	if err != nil {
		return fmt.Errorf("redis set REQ_CHKMNGR_ENDED: %v", err)
	}
	fmt.Fprintf(os.Stderr, "set %s to 0\n", REQ_CHKMNGR_ENDED)
	err = c.rds[CHKPT_META_NODE].Set(ctx, CHKPT_MNGR_ENDED, 0, 0).Err()
	if err != nil {
		return fmt.Errorf("redis set CHKPT_MNGR_ENDED: %v", err)
	}
	fmt.Fprintf(os.Stderr, "set %s to 0\n", CHKPT_MNGR_ENDED)
	return nil
}

func (c *RedisChkptManager) ResetCheckPointCount(ctx context.Context, finalOutputTopicNames []string) error {
	_, err := c.rds[CHKPT_META_NODE].Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, tp := range finalOutputTopicNames {
			pipe.Set(ctx, tp, 0, 0)
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("ResetCheckPointCount: %v", err)
	}
	return err
}

func (c *RedisChkptManager) WaitForChkptFinish(
	ctx context.Context,
	finalOutputTopicNames []string,
	finalNumOutPartition []uint8,
) (bool, error) {
	for {
		allFinished := true
		cmds, err := c.rds[CHKPT_META_NODE].Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for _, tp := range finalOutputTopicNames {
				pipe.Get(ctx, tp)
			}
			return nil
		})
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return false, fmt.Errorf("redis pipelined get: %v", err)
		}
		for idx, cmd := range cmds {
			parNum := finalNumOutPartition[idx]
			gotParStr := cmd.(*redis.StringCmd).Val()
			gotPar, err := strconv.Atoi(gotParStr)
			if err != nil {
				return false, err
			}
			if gotPar != int(parNum) {
				allFinished = false
				break
			}
		}
		if allFinished {
			return false, nil
		}
		req, err := c.GetReqChkMngrEnd(ctx)
		if err != nil {
			return false, err
		}
		if req == 1 {
			err = c.SetChkMngrEnded(ctx)
			if err != nil {
				return false, err
			}
			return true, nil
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (c *RedisChkptManager) FinishChkpt(ctx context.Context, finalOutParNames []string) error {
	_, err := c.rds[CHKPT_META_NODE].Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, tp := range finalOutParNames {
			pipe.Incr(ctx, tp)
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("FinishChkpt: %v", err)
	}
	return err
}

func (c *RedisChkptManager) ReqChkMngrEnd(ctx context.Context) error {
	debug.Fprintf(os.Stderr, "set %s to 1\n", REQ_CHKMNGR_ENDED)
	err := c.rds[CHKPT_META_NODE].Set(ctx, REQ_CHKMNGR_ENDED, 1, 0).Err()
	if err != nil {
		err = fmt.Errorf("ReqChkMngrEnd: %v", err)
	}
	return err
}

func (c *RedisChkptManager) GetReqChkMngrEnd(ctx context.Context) (int, error) {
	ret, err := c.rds[CHKPT_META_NODE].Get(ctx, REQ_CHKMNGR_ENDED).Int()
	if err != nil {
		err = fmt.Errorf("GetReqChkMngrEnd: %v", err)
	}
	return ret, err
}

func (c *RedisChkptManager) SetChkMngrEnded(ctx context.Context) error {
	debug.Fprintf(os.Stderr, "set %s to 1\n", CHKPT_MNGR_ENDED)
	err := c.rds[CHKPT_META_NODE].Set(ctx, CHKPT_MNGR_ENDED, 1, 0).Err()
	if err != nil {
		err = fmt.Errorf("SetChkMngrEnded: %v", err)
	}
	return err
}

func (c *RedisChkptManager) GetChkMngrEnded(ctx context.Context) (int, error) {
	ret, err := c.rds[CHKPT_META_NODE].Get(ctx, CHKPT_MNGR_ENDED).Int()
	if err != nil {
		err = fmt.Errorf("GetChkMngrEnded: %v", err)
	}
	return ret, err
}
