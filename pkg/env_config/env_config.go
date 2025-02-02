package env_config

import (
	"fmt"
	"os"
)

var (
	PARALLEL_RESTORE   = checkParallelRestore()
	ASYNC_SECOND_PHASE = checkAsyncSecondPhase()
	CREATE_SNAPSHOT    = checkCreateSnapshot()
)

func checkParallelRestore() bool {
	parallelRestore_str := os.Getenv("PARALLEL_RESTORE")
	parallelRestore := false
	if parallelRestore_str == "true" || parallelRestore_str == "1" {
		parallelRestore = true
	}
	fmt.Fprintf(os.Stderr, "parallel restore str: %s, %v\n", parallelRestore_str, parallelRestore)
	return parallelRestore
}

func checkAsyncSecondPhase() bool {
	async2ndPhaseStr := os.Getenv("ASYNC_SECOND_PHASE")
	async2ndPhase := async2ndPhaseStr == "true" || async2ndPhaseStr == "1"
	fmt.Fprintf(os.Stderr, "env str: %s, async2ndPhase: %v\n", async2ndPhaseStr, async2ndPhase)
	return async2ndPhase
}

func checkCreateSnapshot() bool {
	createSnapshotStr := os.Getenv("CREATE_SNAPSHOT")
	createSnapshot := createSnapshotStr == "true" || createSnapshotStr == "1"
	fmt.Fprintf(os.Stderr, "env str: %s, create snapshot: %v\n", createSnapshotStr, createSnapshot)
	return createSnapshot
}
