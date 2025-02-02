package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/producer_consumer"
)

type GroupByOutputProcessorG[KIn, VIn any] struct {
	byteSliceHasher hashfuncs.ByteSliceHasher
	producer        producer_consumer.MeteredProducerIntr
	ectx            ExecutionContext
	// cHash    *hash.ConsistentHash
	name       string
	msgGSerdeG commtypes.MessageGSerdeG[KIn, VIn]
	BaseProcessorG[KIn, VIn, any, any]
	kUseBuf bool
	vUseBuf bool
	// procTimeStats stats.PrintLogStatsCollector[int64]
}

func NewGroupByOutputProcessorG[KIn, VIn any](procTimeTag string, producer producer_consumer.MeteredProducerIntr,
	ectx ExecutionContext, msgGSerdeG commtypes.MessageGSerdeG[KIn, VIn],
) *GroupByOutputProcessorG[KIn, VIn] {
	// numPartition := producer.Stream().NumPartition()
	g := GroupByOutputProcessorG[KIn, VIn]{
		// cHash:    hash.NewConsistentHash(),
		producer:        producer,
		name:            "to" + producer.TopicName(),
		ectx:            ectx,
		byteSliceHasher: hashfuncs.ByteSliceHasher{},
		msgGSerdeG:      msgGSerdeG,
		kUseBuf:         msgGSerdeG.GetKeySerdeG().UsedBufferPool(),
		vUseBuf:         msgGSerdeG.GetValSerdeG().UsedBufferPool(),
		// procTimeStats:   stats.NewPrintLogStatsCollector[int64](procTimeTag),
	}
	// for i := uint8(0); i < numPartition; i++ {
	// 	g.cHash.Add(i)
	// }
	g.BaseProcessorG.ProcessingFuncG = g.ProcessAndReturn
	return &g
}

func (g *GroupByOutputProcessorG[KIn, VIn]) OutputRemainingStats() {
	// g.procTimeStats.PrintRemainingStats()
}

func (g *GroupByOutputProcessorG[KIn, VIn]) Name() string {
	return g.name
}

func (g *GroupByOutputProcessorG[KIn, VIn]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[KIn, VIn],
) ([]commtypes.MessageG[any, any], error) {
	// parTmp, ok := g.cHash.Get(msg.Key)
	// if !ok {
	// 	return nil, common_errors.ErrFailToGetOutputSubstream
	// }
	// procTime := time.Since(msg.StartProcTime)
	// g.procTimeStats.AddSample(procTime.Nanoseconds())
	msgSerOp, kbuf, vbuf, err := commtypes.MsgGToMsgSer(msg, g.msgGSerdeG)
	if err != nil {
		return nil, err
	}
	msgSer, ok := msgSerOp.Take()
	if ok {
		hash := g.byteSliceHasher.HashSum64(msgSer.KeyEnc)
		par := uint8(hash % uint64(g.producer.Stream().NumPartition()))
		g.ectx.TrackParFunc()(g.producer.TopicName(), par)
		err = g.producer.ProduceData(ctx, msgSer, par)
		if g.kUseBuf && msgSer.KeyEnc != nil && kbuf != nil {
			*kbuf = msgSer.KeyEnc
			commtypes.PushBuffer(kbuf)
		}
		if g.vUseBuf && msgSer.ValueEnc != nil && vbuf != nil {
			*vbuf = msgSer.ValueEnc
			commtypes.PushBuffer(vbuf)
		}
		return nil, err
	} else {
		return nil, nil
	}
}

/*
type GroupByOutputProcessorWithCache[K comparable, V any] struct {
	byteSliceHasher hashfuncs.ByteSliceHasher
	producer        producer_consumer.MeteredProducerIntr
	ectx            ExecutionContext
	cache           *store.Cache[K, commtypes.OptionalValTsG[V]]
	// cHash    *hash.ConsistentHash
	name string
	BaseProcessor
}

var _ = CachedProcessor(&GroupByOutputProcessorWithCache[int, int]{})

func NewGroupByOutputProcessorWithCache[K comparable, V any](
	ctx context.Context,
	producer producer_consumer.MeteredProducerIntr,
	sizeOfKey func(k K) int64, sizeOfVal func(v V) int64, maxCacheBytes int64,
	ectx ExecutionContext,
) *GroupByOutputProcessorWithCache[K, V] {
	// numPartition := producer.Stream().NumPartition()
	g := GroupByOutputProcessorWithCache[K, V]{
		// cHash:    hash.NewConsistentHash(),
		producer:        producer,
		name:            "to" + producer.TopicName(),
		ectx:            ectx,
		byteSliceHasher: hashfuncs.ByteSliceHasher{},
	}
	sizeOfVts := commtypes.OptionalValTsGSize[V]{
		ValSizeFunc: sizeOfVal,
	}
	cache := store.NewCache(func(elements []store.LRUElement[K, commtypes.OptionalValTsG[V]]) error {
		for _, element := range elements {
			kBytes, err := g.producer.KeyEncoder().Encode(element.Key())
			if err != nil {
				return err
			}
			hash := g.byteSliceHasher.HashSum64(kBytes)
			par := uint8(hash % uint64(g.producer.Stream().NumPartition()))
			err = g.ectx.TrackParFunc()(ctx, kBytes, g.producer.TopicName(), par)
			if err != nil {
				return fmt.Errorf("track substream failed: %v", err)
			}
			var v interface{}
			entry := element.Entry()
			vOpTs, ok := entry.Value().Take()
			if !ok {
				return fmt.Errorf("value is not set")
			}
			v, ok = vOpTs.Val.Take()
			if !ok {
				v = nil
			}
			msg := commtypes.Message{Key: element.Key(), Value: v, Timestamp: vOpTs.Timestamp}
			err = g.producer.Produce(ctx, msg, par, false)
			if err != nil {
				return err
			}
		}
		return nil
	}, sizeOfKey, sizeOfVts.SizeOfOptionalValTsG, maxCacheBytes)
	g.cache = cache

	// for i := uint8(0); i < numPartition; i++ {
	// 	g.cHash.Add(i)
	// }
	g.BaseProcessor.ProcessingFunc = g.ProcessAndReturn
	return &g
}

func (g *GroupByOutputProcessorWithCache[K, V]) Name() string {
	return g.name
}

func (g *GroupByOutputProcessorWithCache[K, V]) ProcessAndReturn(ctx context.Context, msg commtypes.Message,
) ([]commtypes.Message, error) {
	vOp := optional.None[V]()
	if !utils.IsNil(msg.Value) {
		vOp = optional.Some(msg.Value.(V))
	}
	err := g.cache.PutMaybeEvict(msg.Key.(K),
		store.DirtyEntry(optional.Some(commtypes.OptionalValTsG[V]{Val: vOp, Timestamp: msg.Timestamp})))
	return nil, err
}

func (g *GroupByOutputProcessorWithCache[K, V]) Flush(ctx context.Context) error {
	return g.cache.Flush()
}

*/
