package store

import (
	"fmt"
	"os"
	"sharedlog-stream/pkg/data_structure/genericlist"
	"sharedlog-stream/pkg/data_structure/linkedhashset"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/stats"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	ptrSize = int64(unsafe.Sizeof((*int)(nil)))
)

type LRUEntry[V any] struct {
	value   optional.Option[V]
	tm      TimeMeta
	isDirty bool
}

func DirtyEntry[V any](v optional.Option[V]) LRUEntry[V] {
	return LRUEntry[V]{isDirty: true, value: v}
}

func CleanEntry[V any](v optional.Option[V]) LRUEntry[V] {
	return LRUEntry[V]{isDirty: false, value: v}
}

func (lruE *LRUEntry[V]) MarkClean() {
	lruE.isDirty = false
}

func (lruE *LRUEntry[V]) IsDirty() bool {
	return lruE.isDirty
}

func (lruE *LRUEntry[V]) Value() optional.Option[V] {
	return lruE.value
}

type LRUElement[K, V any] struct {
	key   K
	entry LRUEntry[V]
}

func (e *LRUElement[K, V]) Key() K {
	return e.key
}

func (e *LRUElement[K, V]) Entry() LRUEntry[V] {
	return e.entry
}

type Cache[K comparable, V any] struct {
	mux              sync.Mutex
	sizeOfKey        func(K) int64
	cache            map[K]*genericlist.Element[LRUElement[K, V]] // protected by mux
	dirtyKeys        *linkedhashset.LinkedHashSet[K]              // protected by mux
	orderList        *genericlist.List[LRUElement[K, V]]          // protected by mux
	flushCallback    FlushCallbackFunc[K, V]
	sizeOfVal        func(V) int64
	hitRatio         stats.StatsCollector[float64]
	currentSizeBytes int64
	maxCacheBytes    int64
	numReadHits      uint64
	numReadMisses    uint64
	numOverwrites    uint64
	numFlushes       uint64
	numEvicts        uint64
}

type FlushCallbackFunc[K comparable, V any] func(entries []LRUElement[K, V]) error

func NewCache[K comparable, V any](flushCallback FlushCallbackFunc[K, V],
	sizeOfKey func(k K) int64, sizeOfVal func(v V) int64, maxCacheBytes int64,
) *Cache[K, V] {
	return &Cache[K, V]{
		cache:            make(map[K]*genericlist.Element[LRUElement[K, V]]),
		dirtyKeys:        linkedhashset.New[K](),
		orderList:        genericlist.New[LRUElement[K, V]](),
		flushCallback:    flushCallback,
		hitRatio:         stats.NewStatsCollector[float64]("cache_hit_ratio", stats.DEFAULT_COLLECT_DURATION),
		sizeOfKey:        sizeOfKey,
		sizeOfVal:        sizeOfVal,
		maxCacheBytes:    maxCacheBytes,
		currentSizeBytes: 0,
		numReadHits:      0,
		numReadMisses:    0,
		numOverwrites:    0,
		numFlushes:       0,
	}
}

func (c *Cache[K, V]) hits() uint64 {
	return atomic.LoadUint64(&c.numReadHits)
}

func (c *Cache[K, V]) misses() uint64 {
	return atomic.LoadUint64(&c.numReadMisses)
}

func (c *Cache[K, V]) overwrites() uint64 {
	return atomic.LoadUint64(&c.numOverwrites)
}

func (c *Cache[K, V]) flushes() uint64 {
	return atomic.LoadUint64(&c.numFlushes)
}

func (c *Cache[K, V]) evicts() uint64 {
	return atomic.LoadUint64(&c.numEvicts)
}

func (c *Cache[K, V]) first() LRUEntry[V] {
	c.mux.Lock()
	ret := c.orderList.Front().Value.entry
	c.mux.Unlock()
	return ret
}

func (c *Cache[K, V]) last() LRUEntry[V] {
	c.mux.Lock()
	ret := c.orderList.Back().Value.entry
	c.mux.Unlock()
	return ret
}

/*
func (c *Cache[K, V]) head() *genericlist.Element[LRUElement[K, V]] {
	c.mux.Lock()
	ret := c.orderList.Front()
	c.mux.Unlock()
	return ret
}

func (c *Cache[K, V]) tail() *genericlist.Element[LRUElement[K, V]] {
	c.mux.Lock()
	ret := c.orderList.Back()
	c.mux.Unlock()
	return ret
}
*/

func (c *Cache[K, V]) len() int {
	return len(c.cache)
}

func elementValueSize[K, V any](element *genericlist.Element[LRUElement[K, V]], sizeOfVal func(V) int64) int64 {
	v, exists := element.Value.entry.Value().Take()
	vSize := int64(0)
	if exists {
		vSize = sizeOfVal(v)
	}
	return vSize + 3*ptrSize
}

func (c *Cache[K, V]) getInternalLockHeld(key K) *genericlist.Element[LRUElement[K, V]] {
	element, ok := c.cache[key]
	if ok {
		atomic.AddUint64(&c.numReadHits, 1)
		hitRadio := float64(atomic.LoadUint64(&c.numReadHits)) / float64(atomic.LoadUint64(&c.numReadHits)+atomic.LoadUint64(&c.numReadMisses))
		c.hitRatio.AddSample(hitRadio)
		return element
	} else {
		atomic.AddUint64(&c.numReadMisses, 1)
		return nil
	}
}

func (c *Cache[K, V]) updateLRULockHeld(element *genericlist.Element[LRUElement[K, V]]) {
	c.orderList.MoveToFront(element)
}

func (c *Cache[K, V]) get(key K) (LRUEntry[V], bool /* found */) {
	c.mux.Lock()
	element := c.getInternalLockHeld(key)
	if element != nil {
		c.updateLRULockHeld(element)
		c.mux.Unlock()
		return element.Value.entry, true
	} else {
		c.mux.Unlock()
		return LRUEntry[V]{}, false
	}
}

func (c *Cache[K, V]) put(key K, value LRUEntry[V]) error {
	c.mux.Lock()
	if !value.IsDirty() && c.dirtyKeys.Contains(key) {
		c.mux.Unlock()
		return fmt.Errorf("Attempting to put a clean entrty for key[%v] into cache when it already contains a dirty entry for the same key", key)
	}
	kSize := c.sizeOfKey(key)
	element := c.cache[key]
	if element != nil {
		atomic.AddUint64(&c.numOverwrites, 1)
		totSize := kSize + elementValueSize(element, c.sizeOfVal)
		atomic.AddInt64(&c.currentSizeBytes, int64(-1)*totSize)
		element.Value.entry = value
		c.updateLRULockHeld(element)
	} else {
		element = c.orderList.PushFront(LRUElement[K, V]{key: key, entry: value})
		c.cache[key] = element
	}
	if value.isDirty {
		c.dirtyKeys.Remove(key)
		c.dirtyKeys.Add(key)
	}
	c.mux.Unlock()
	totSize := kSize + elementValueSize(element, c.sizeOfVal)
	atomic.AddInt64(&c.currentSizeBytes, totSize)
	return nil
}

func (c *Cache[K, V]) maybeEvict() error {
	numEvicted := uint32(0)
	for atomic.LoadInt64(&c.currentSizeBytes) > c.maxCacheBytes {
		err := c.evict()
		if err != nil {
			return err
		}
		atomic.AddUint32(&numEvicted, 1)
		atomic.AddUint64(&c.numEvicts, 1)
	}
	// if atomic.LoadUint32(&numEvicted) > 0 {
	// 	// fmt.Fprintf(os.Stderr, "Evicted %v elements from cache\n", numEvicted)
	// }
	return nil
}

func (c *Cache[K, V]) PutMaybeEvict(key K, value LRUEntry[V]) error {
	err := c.put(key, value)
	if err != nil {
		return err
	}
	return c.maybeEvict()
}

func (c *Cache[K, V]) PutIfAbsentMaybeEvict(key K, value LRUEntry[V]) (LRUEntry[V], bool, error) {
	ret, ok, err := c.putIfAbsent(key, value)
	if err != nil {
		return LRUEntry[V]{}, false, err
	}
	err = c.maybeEvict()
	if err != nil {
		return LRUEntry[V]{}, false, err
	}
	return ret, ok, nil
}

func (c *Cache[K, V]) evict() error {
	c.mux.Lock()
	element := c.orderList.Back()
	if element != nil {
		kSize := c.sizeOfKey(element.Value.key)
		totSize := kSize + elementValueSize(element, c.sizeOfVal)
		atomic.AddInt64(&c.currentSizeBytes, int64(-totSize))
		c.orderList.Remove(element)
		delete(c.cache, element.Value.key)
		if element.Value.entry.isDirty {
			err := c.flushLockHeld(element)
			if err != nil {
				c.mux.Unlock()
				return err
			}
		}
	}
	c.mux.Unlock()
	return nil
}

func (c *Cache[K, V]) flush(*genericlist.Element[LRUElement[K, V]]) error {
	c.mux.Lock()
	err := c.flushLockHeld(nil)
	c.mux.Unlock()
	return err
}

func (c *Cache[K, V]) Flush() error {
	return c.flush(nil)
}

func (c *Cache[K, V]) flushLockHeld(evicted *genericlist.Element[LRUElement[K, V]]) error {
	atomic.AddUint64(&c.numFlushes, 1)
	if c.dirtyKeys.Len() == 0 {
		return nil
	}
	entries := make([]LRUElement[K, V], 0, c.dirtyKeys.Len())
	deleted := make([]K, 0, c.dirtyKeys.Len())
	if evicted != nil {
		entries = append(entries, evicted.Value)
		c.dirtyKeys.Remove(evicted.Value.key)
	}
	c.dirtyKeys.IterateCb(func(key K) bool {
		element := c.getInternalLockHeld(key)
		if element == nil {
			fmt.Fprintf(os.Stderr, "key=%v found in dirty key set, but entry is nil\n", key)
			return false
		}
		entries = append(entries, c.cache[key].Value)
		element.Value.entry.MarkClean()
		if element.Value.entry.value.IsNone() {
			deleted = append(deleted, key)
		}
		return true
	})
	c.dirtyKeys.Clear()
	err := c.flushCallback(entries)
	if err != nil {
		return err
	}
	for _, k := range deleted {
		c.deleteLockHeld(k)
	}
	return nil
}

func (c *Cache[K, V]) deleteLockHeld(key K) (LRUEntry[V], bool) {
	element, ok := c.cache[key]
	if ok {
		delete(c.cache, key)
	} else {
		return LRUEntry[V]{}, false
	}
	c.orderList.Remove(element)
	c.dirtyKeys.Remove(key)
	return element.Value.entry, true
}

func (c *Cache[K, V]) delete(key K) (LRUEntry[V], bool) {
	c.mux.Lock()
	entry, ok := c.deleteLockHeld(key)
	c.mux.Unlock()
	return entry, ok
}

func (c *Cache[K, V]) putIfAbsent(key K, value LRUEntry[V]) (LRUEntry[V], bool, error) {
	originalValue, found := c.get(key)
	if !found {
		err := c.put(key, value)
		if err != nil {
			return LRUEntry[V]{}, false, err
		}
	}
	return originalValue, found, nil
}
