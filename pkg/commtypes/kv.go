package commtypes

import (
	"context"
	"time"
)

type CTXID struct{}

type KeyT interface{}

type ValueT interface{}

type CreateChangelogManagerParam struct {
	NumPartition  uint8
	FlushDuration time.Duration
	TimeOut       time.Duration
}

type ENVID struct{}

func GetCtxId(ctx context.Context) string {
	id, ok := ctx.Value(CTXID{}).(string)
	if ok {
		return id
	} else {
		return ""
	}
}
