package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/stats"
	"sort"
	"strings"
)

type benchStats struct {
	Latencies map[string][]int
	Counts    map[string]uint64
	EventTs   []int64
	Duration  float64
}

func ProcessThroughputLat(name string, stat_dir string, latencies map[string][]int,
	counts map[string]uint64, eventTs []int64, duration float64,
	num map[string]uint64, endToEnd *float64,
) {
	path := path.Join(stat_dir, fmt.Sprintf("%s.json", name))
	if err := os.MkdirAll(stat_dir, 0750); err != nil {
		panic(fmt.Sprintf("Fail to create stat dir: %v", err))
	}
	st := benchStats{
		Latencies: latencies,
		Counts:    counts,
		Duration:  duration,
		EventTs:   eventTs,
	}
	stBytes, err := json.Marshal(st)
	if err != nil {
		panic(fmt.Sprintf("fail to marshal stats: %v", err))
	}
	err = os.WriteFile(path, stBytes, 0666)
	if err != nil {
		panic(fmt.Sprintf("fail to write stats to file: %v", err))
	}
	for n, lat_arr := range latencies {
		if len(lat_arr) != 0 {
			ts := stats.IntSlice(lat_arr)
			sort.Sort(ts)
			if !strings.Contains(n, "eventTimeLatency") {
				sumTime := float64(0)
				for _, lat := range lat_arr {
					sumTime += float64(lat) / 1000000.0
				}
				processed := uint64(len(lat_arr))
				var ok bool
				if counts != nil {
					processed, ok = counts[n]
					if !ok {
						debug.Fprintf(os.Stderr, "does not find %s in consumed\n", n)
						processed = uint64(len(lat_arr))
					}
				} else {
					debug.Fprint(os.Stderr, "consumed is empty")
				}
				if n == "src" || strings.Contains(name, "source") || strings.Contains(n, "Src") || n == "sink" || strings.Contains(n, "Sink") {
					num[n] += processed
				}
				if n == "e2e" {
					if *endToEnd < sumTime {
						*endToEnd = sumTime
					}
				}
				tput := float64(processed) / sumTime
				fmt.Fprintf(os.Stderr, "sum of %s time: %v ", n, sumTime)
				fmt.Fprintf(os.Stderr, "processed: %v, throughput: (event/s) %v, p50: %d us, p90: %d us, p99: %d us\n",
					processed, tput, stats.P(ts, 0.5), stats.P(ts, 0.9), stats.P(ts, 0.99))
			} else {
				fmt.Fprintf(os.Stderr, "%s, p50: %d ms, p90: %d ms, p99: %d ms\n",
					n, stats.P(ts, 0.5), stats.P(ts, 0.9), stats.P(ts, 0.99))
			}
		}
	}
	for n, count := range counts {
		fmt.Fprintf(os.Stderr, "%s processed: %v, throughput: (event/s) %v\n", n, count, float64(count)/duration)
		if strings.Contains(name, "source") || strings.Contains(n, "Src") || strings.Contains(n, "src") || strings.Contains(n, "Sink") || strings.Contains(n, "sink") {
			num[n] += count
		}
	}
	fmt.Fprintf(os.Stderr, "%s duration: %v\n\n", name, duration)
}
