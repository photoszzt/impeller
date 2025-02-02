package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
)

var (
	FLAGS_spec       string
	FLAGS_app_name   string
	FLAGS_outputFile string
)

func outputEvents(events []*ntypes.Event, outputFile string) error {
	earr := events_arr{
		Earr: events,
	}
	earr_bytes, err := json.Marshal(earr)
	if err != nil {
		return err
	}
	absOutFileName, err := filepath.Abs(outputFile)
	if err != nil {
		return err
	}
	err = os.WriteFile(absOutFileName, earr_bytes, 0660)
	return err
}

func main() {
	flag.StringVar(&FLAGS_app_name, "app_name", "q5", "")
	flag.StringVar(&FLAGS_spec, "spec", "", "spec to generate events")
	flag.StringVar(&FLAGS_outputFile, "ofile", "", "output file name")
	flag.Parse()

	if FLAGS_app_name == "" {
		fmt.Fprintf(os.Stderr, "must specify app name")
		return
	}

	if FLAGS_outputFile == "" {
		fmt.Fprintf(os.Stderr, "should specify output file name")
		return
	}
	var specs []byte
	var err error
	if FLAGS_spec != "" {
		specs, err = os.ReadFile(FLAGS_spec)
		if err != nil {
			panic(err)
		}
	}
	if FLAGS_app_name == "q5" && specs != nil {
		err = q5_gen_data(specs, FLAGS_outputFile)
	} else if FLAGS_app_name == "q6" {
		err = q6_gen_data(FLAGS_outputFile)
	} else if FLAGS_app_name == "q7" && specs != nil {
		err = q7_gen_data(specs, FLAGS_outputFile)
	} else if FLAGS_app_name == "q8" && specs != nil {
		err = q8_gen_data(specs, FLAGS_outputFile)
	}
	if err != nil {
		panic(err)
	}
}
