package store_restore

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/store"
	"time"

	"golang.org/x/xerrors"
)

func RestoreChangelogKVStateStore(
	ctx context.Context,
	kvchangelog store.KeyValueStoreOpWithChangelog,
	parNum uint8,
) error {
	count := 0
	numLogEntry := 0
	restoreKVStart := time.Now()
	for {
		subC, err := kvchangelog.ConsumeOneLogEntry(ctx, parNum)
		// nothing to restore
		if common_errors.IsStreamEmptyError(err) {
			elapsed := time.Since(restoreKVStart)
			fmt.Fprintf(os.Stderr, "%s(%d) restore, count: %d, numLogEntry: %d, elapsed: %v\n",
				kvchangelog.ChangelogTopicName(), parNum, count, numLogEntry, elapsed)
			return nil
		} else if xerrors.Is(err, common_errors.ErrStreamSourceTimeout) {
			elapsed := time.Since(restoreKVStart)
			fmt.Fprintf(os.Stderr, "%s(%d) restore, count: %d, numLogEntry: %d, elapsed: %v\n",
				kvchangelog.ChangelogTopicName(), parNum, count, numLogEntry, elapsed)
			return nil
		} else if xerrors.Is(err, common_errors.ErrReachCutoffPos) {
			elapsed := time.Since(restoreKVStart)
			fmt.Fprintf(os.Stderr, "%s(%d) restore, count: %d, numLogEntry: %d, elapsed: %v\n",
				kvchangelog.ChangelogTopicName(), parNum, count, numLogEntry, elapsed)
			return nil
		} else if err != nil {
			return fmt.Errorf("ReadNext failed: %v", err)
		}
		count += subC
		numLogEntry += 1
	}
}
