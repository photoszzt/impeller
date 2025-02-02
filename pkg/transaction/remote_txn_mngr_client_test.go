package transaction

import (
	"sharedlog-stream/pkg/stats"
	"testing"

	"github.com/zhangyunhao116/skipmap"
	"github.com/zhangyunhao116/skipset"
)

func checkContains(t *testing.T, r *RemoteTxnManagerClient, tp string, substream uint8) {
	v, ok := r.currentTopicSubstream.Load(tp)
	if !ok {
		t.Fatal("should contain test")
	}
	if !v.Contains(uint32(substream)) {
		t.Fatal("should contain")
	}
}

func TestRemoteTxnManagerClient(t *testing.T) {
	r := &RemoteTxnManagerClient{
		currentTopicSubstream: skipmap.NewString[*skipset.Uint32Set](),
		TransactionalId:       "test",
		waitAndappendTxnMeta:  stats.NewPrintLogStatsCollector[int64]("waitAndappendTxnMeta"),
	}
	r.AddTopicSubstream("test", 1)
	r.AddTopicSubstream("test", 3)
	r.AddTopicSubstream("2", 2)
	if !r.addedNewTpPar.Load() {
		t.Fatal("new tp par should be set")
	}
	checkContains(t, r, "test", 1)
	checkContains(t, r, "test", 3)
	checkContains(t, r, "2", 2)
}
