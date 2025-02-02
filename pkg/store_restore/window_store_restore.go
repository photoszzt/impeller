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

func RestoreChangelogWindowStateStore(
	ctx context.Context,
	wschangelog store.WindowStoreOpWithChangelog,
	parNum uint8,
) error {
	count := 0
	numLogEntry := 0
	restoreStart := time.Now()
	for {
		subC, err := wschangelog.ConsumeOneLogEntry(ctx, parNum)
		// nothing to restore
		if common_errors.IsStreamEmptyError(err) {
			elapsed := time.Since(restoreStart)
			fmt.Fprintf(os.Stderr, "%s(%d) restore, count: %d, numLogEntry: %d, elapsed: %v\n",
				wschangelog.ChangelogTopicName(), parNum, count, numLogEntry, elapsed)
			return nil
		} else if xerrors.Is(err, common_errors.ErrStreamSourceTimeout) {
			elapsed := time.Since(restoreStart)
			fmt.Fprintf(os.Stderr, "%s(%d) restore, count: %d, numLogEntry: %d, elapsed: %v\n",
				wschangelog.ChangelogTopicName(), parNum, count, numLogEntry, elapsed)
			return nil
		} else if err != nil {
			return fmt.Errorf("ReadNext failed: %v", err)
		}
		count += subC
		numLogEntry += 1
	}
}
