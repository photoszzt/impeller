package common

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

func SetLogLevelFromEnv() {
	logLevel := os.Getenv("LOG_LEVEL")
	if level, err := zerolog.ParseLevel(logLevel); err == nil {
		zerolog.SetGlobalLevel(level)
	}
}

func CheckCacheConfig() bool {
	useCacheStr := os.Getenv("USE_CACHE")
	useCache := useCacheStr == "true" || useCacheStr == "1"
	fmt.Fprintf(os.Stderr, "use cacheStr: %s, use cache: %v\n", useCacheStr, useCache)
	return useCache
}
