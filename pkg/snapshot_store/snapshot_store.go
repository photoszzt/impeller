package snapshot_store

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/redis_client"
	"sharedlog-stream/pkg/stats"
	"strconv"
	"strings"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/go-redis/redis/v9"
)

type SnapshotStore interface {
	StoreSrcLogoff(ctx context.Context, srcLogOff []commtypes.TpLogOff, instanceId uint8) error
	StoreAlignChkpt(ctx context.Context, snapshot []byte,
		srcLogOff []commtypes.TpLogOff, storeName string, instanceId uint8,
	) error
	GetAlignChkpt(ctx context.Context, srcs []string, storeName string, instanceId uint8) ([]byte, error)
	StoreSnapshot(ctx context.Context,
		snapshot []byte, changelogTpName string, logOff uint64, instanceId uint8,
	) error
	GetSnapshot(ctx context.Context, changelogTpName string, logOff uint64, instanceId uint8) ([]byte, error)
}

var _ = SnapshotStore(&RedisSnapshotStore{})

type RedisSnapshotStore struct {
	rdb_arr  []*redis.Client
	snapSize *stats.ConcurrentStatsCollector[int]
}

func NewRedisSnapshotStore(createSnapshot bool) RedisSnapshotStore {
	if createSnapshot {
		c := redis_client.GetRedisClients()
		fmt.Fprintf(os.Stderr, "redis client arr size %d\n", len(c))
		return RedisSnapshotStore{
			rdb_arr:  c,
			snapSize: stats.NewConcurrentStatsCollector[int]("redisStore", stats.DEFAULT_COLLECT_DURATION),
		}
	} else {
		return RedisSnapshotStore{}
	}
}

func (rs *RedisSnapshotStore) PrintRemainingStats() {
	if rs.snapSize != nil {
		rs.snapSize.PrintRemainingStats()
	}
}

func (rs *RedisSnapshotStore) GetRedisClients() []*redis.Client {
	debug.Assert(len(rs.rdb_arr) != 0, "rdb arr should not be empty")
	return rs.rdb_arr
}

func (rs *RedisSnapshotStore) StoreSrcLogoff(ctx context.Context,
	srcLogOff []commtypes.TpLogOff, instanceId uint8,
) error {
	l := uint64(len(rs.rdb_arr))
	idx := uint64(instanceId) % l
	cmds, err := rs.rdb_arr[idx].Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, tpOff := range srcLogOff {
			p.RPush(ctx, tpOff.Tp, tpOff.LogOff)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		err := cmd.(*redis.IntCmd).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *RedisSnapshotStore) StoreAlignChkpt(ctx context.Context, snapshot []byte,
	srcLogOff []commtypes.TpLogOff, storeName string, instanceId uint8,
) error {
	var keys []string
	l := uint64(len(rs.rdb_arr))
	for _, tpLogOff := range srcLogOff {
		keys = append(keys, fmt.Sprintf("%s_%#x", tpLogOff.Tp, tpLogOff.LogOff))
	}
	keys = append(keys, storeName)
	key := strings.Join(keys, "-")
	idx := uint64(instanceId) % l
	debug.Fprintf(os.Stderr, "store chkpt key: %s at redis[%d], instance id %d, redis arr len %d\n", key, idx, instanceId, l)
	rs.snapSize.AddSample(len(snapshot))
	err := rs.rdb_arr[idx].Set(ctx, key, snapshot, time.Duration(0)*time.Second).Err()
	if err != nil {
		return err
	}
	return rs.StoreSrcLogoff(ctx, srcLogOff, instanceId)
}

func (rs *RedisSnapshotStore) GetAlignChkpt(ctx context.Context, srcs []string, storeName string, instanceId uint8) ([]byte, error) {
	var keys []string
	for _, tp := range srcs {
		idx := hashfuncs.NameHash(tp) % uint64(len(rs.rdb_arr))
		logOffStrs, err := rs.rdb_arr[idx].LRange(ctx, tp, -1, -1).Result()
		if err != nil {
			return nil, err
		}
		logOff, err := strconv.Atoi(logOffStrs[0])
		if err != nil {
			return nil, err
		}
		keys = append(keys, fmt.Sprintf("%s_%#x", tp, logOff))
	}
	keys = append(keys, storeName)
	key := strings.Join(keys, "-")
	idx := uint64(instanceId) % uint64(len(rs.rdb_arr))
	debug.Fprintf(os.Stderr, "get snapshot key: %s at redis[%d]\n", key, idx)
	return rs.rdb_arr[idx].Get(ctx, key).Bytes()
}

func (rs *RedisSnapshotStore) StoreSnapshot(ctx context.Context,
	snapshot []byte, changelogTpName string, logOff uint64, instanceId uint8,
) error {
	var uint16Serde commtypes.Uint16Serde
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	key := fmt.Sprintf("%s_%#x", changelogTpName, logOff)
	idx := uint64(instanceId) % uint64(len(rs.rdb_arr))
	fmt.Fprintf(os.Stderr, "store snapshot key: %s at redis[%d]\n", key, idx)
	err := rs.rdb_arr[idx].Set(ctx, key, snapshot, time.Duration(60)*time.Second).Err()
	if err != nil {
		return err
	}
	rs.snapSize.AddSample(len(snapshot))
	hasData := uint16(1)
	enc, _, err := uint16Serde.Encode(hasData)
	if err != nil {
		return err
	}
	return env.SharedLogSetAuxData(ctx, logOff, enc)
}

func (rs *RedisSnapshotStore) GetSnapshot(ctx context.Context, changelogTpName string, logOff uint64, instanceId uint8) ([]byte, error) {
	key := fmt.Sprintf("%s_%#x", changelogTpName, logOff)
	idx := uint64(instanceId) % uint64(len(rs.rdb_arr))
	fmt.Fprintf(os.Stderr, "get snapshot key: %s at redis[%d]\n", key, idx)
	return rs.rdb_arr[idx].Get(ctx, key).Bytes()
}
