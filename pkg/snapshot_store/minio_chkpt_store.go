package snapshot_store

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/hashfuncs"
	"strings"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sync/errgroup"
)

type MinioChkptStore struct {
	minioClients []*minio.Client
}

const (
	accessKey       = "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey = "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
)

var _ = SnapshotStore(&MinioChkptStore{})

func NewMinioChkptStore() (*MinioChkptStore, error) {
	raw_addr := os.Getenv("MINIO_ADDR")
	addr_arr := strings.Split(raw_addr, ",")
	fmt.Fprintf(os.Stderr, "minio addr is %v\n", addr_arr)
	mcs := make([]*minio.Client, len(addr_arr))
	for i := 0; i < len(addr_arr); i++ {
		mc, err := minio.New(addr_arr[i], &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretAccessKey, ""),
			Secure: true,
		})
		if err != nil {
			return nil, err
		}
		mcs[i] = mc
	}
	return &MinioChkptStore{
		minioClients: mcs,
	}, nil
}

const CHKPT_BUCKET_NAME = "workload"

func (mc *MinioChkptStore) CreateWorkloadBucket(ctx context.Context) error {
	for i := 0; i < len(mc.minioClients); i++ {
		err := mc.minioClients[i].MakeBucket(ctx, CHKPT_BUCKET_NAME, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (mc *MinioChkptStore) StoreSrcLogoff(ctx context.Context, srcLogOff []commtypes.TpLogOff, instanceId uint8) error {
	bg, ctx := errgroup.WithContext(ctx)
	for _, tpLogOff := range srcLogOff {
		idx := hashfuncs.NameHash(tpLogOff.Tp) % uint64(len(mc.minioClients))
		bg.Go(func() error {
			bs := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs, tpLogOff.LogOff)
			_, err := mc.minioClients[idx].PutObject(ctx, CHKPT_BUCKET_NAME, tpLogOff.Tp, bytes.NewReader(bs), int64(len(bs)), minio.PutObjectOptions{})
			if err != nil {
				return err
			}
			return nil
		})
		debug.Fprintf(os.Stderr, "store src tpoff %s:%#x at minio[%d]\n",
			tpLogOff.Tp, tpLogOff.LogOff, idx)
	}
	return bg.Wait()
}

func (mc *MinioChkptStore) StoreAlignChkpt(ctx context.Context, snapshot []byte,
	srcLogOff []commtypes.TpLogOff, storeName string, instanceId uint8,
) error {
	var keys []string
	bg, ctx := errgroup.WithContext(ctx)
	l := uint64(len(mc.minioClients))
	for _, tpLogOff := range srcLogOff {
		keys = append(keys, fmt.Sprintf("%s_%#x", tpLogOff.Tp, tpLogOff.LogOff))
	}
	keys = append(keys, storeName)
	key := strings.Join(keys, "-")
	idx := uint64(instanceId) % l
	debug.Fprintf(os.Stderr, "store snapshot key: %s at minio[%d]\n", key, idx)
	_, err := mc.minioClients[idx].PutObject(ctx, CHKPT_BUCKET_NAME, key, bytes.NewReader(snapshot),
		int64(len(snapshot)), minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	for _, tpLogOff := range srcLogOff {
		logOff := tpLogOff.LogOff
		tp := tpLogOff.Tp
		bg.Go(func() error {
			bs := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs, logOff)
			_, err := mc.minioClients[idx].PutObject(ctx, CHKPT_BUCKET_NAME, tp,
				bytes.NewReader(bs), int64(len(bs)), minio.PutObjectOptions{})
			return err
		})
	}
	return bg.Wait()
}

func (mc *MinioChkptStore) GetAlignChkpt(ctx context.Context, srcs []string, storeName string, instanceId uint8) ([]byte, error) {
	var keys []string
	l := uint64(len(mc.minioClients))
	idx := uint64(instanceId) % l
	for _, tp := range srcs {
		object, err := mc.minioClients[idx].GetObject(ctx, CHKPT_BUCKET_NAME, tp, minio.GetObjectOptions{})
		if err != nil {
			return nil, err
		}
		logOffsetBytes, err := io.ReadAll(object)
		if err != nil {
			return nil, err
		}
		defer object.Close()
		logOff := binary.LittleEndian.Uint64(logOffsetBytes)
		keys = append(keys, fmt.Sprintf("%s_%#x", tp, logOff))
	}
	keys = append(keys, storeName)
	key := strings.Join(keys, "-")
	debug.Fprintf(os.Stderr, "get snapshot key: %s at minio[%d]\n", key, idx)
	obj, err := mc.minioClients[idx].GetObject(ctx, CHKPT_BUCKET_NAME, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (mc *MinioChkptStore) StoreSnapshot(ctx context.Context,
	snapshot []byte, changelogTpName string, logOff uint64, instanceId uint8,
) error {
	var uint16Serde commtypes.Uint16Serde
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	key := fmt.Sprintf("%s_%#x", changelogTpName, logOff)
	l := uint64(len(mc.minioClients))
	idx := uint64(instanceId) % l
	fmt.Fprintf(os.Stderr, "store snapshot key: %s at minio[%d]\n", key, idx)
	_, err := mc.minioClients[idx].PutObject(ctx, CHKPT_BUCKET_NAME, key, bytes.NewReader(snapshot),
		int64(len(snapshot)), minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	hasData := uint16(1)
	enc, _, err := uint16Serde.Encode(hasData)
	if err != nil {
		return err
	}
	return env.SharedLogSetAuxData(ctx, logOff, enc)
}

func (mc *MinioChkptStore) GetSnapshot(ctx context.Context, changelogTpName string, logOff uint64, instanceId uint8) ([]byte, error) {
	key := fmt.Sprintf("%s_%#x", changelogTpName, logOff)
	idx := uint64(instanceId) % uint64(len(mc.minioClients))
	fmt.Fprintf(os.Stderr, "get snapshot key: %s at minio[%d]\n", key, idx)
	object, err := mc.minioClients[idx].GetObject(ctx, CHKPT_BUCKET_NAME, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	return io.ReadAll(object)
}
