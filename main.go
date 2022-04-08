package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

var (
	Opts struct {
		database     string
		Environment  string
		InputFile    string
		NumQuery     int64
		Workers      int
		ShowProgress bool
		MaxThreads   int
	}
)

/*
TODOs:
- add workder ticker
- add gradual warm-up
- add logger (and tracing?)
*/

// Cross-platform build:
// Linux
//  GOOS=linux GOARCH=amd64 go build -o db-load-testing-linux
// Mac OS
//  GOOS=darwin GOARCH=amd64 go build -o db-load-testing-macos
// Windows
//  GOOS=windows GOARCH=amd64 go build -o db-load-testing-windows

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	initFlags()
	recorder := NewReorder()
	switch Opts.database {
	case "clickhouse":
		fallthrough
	default:
		runClickhouse(ctx, recorder)
	}
	<-recorder.Done
	recorder.printResult()
}

func initFlags() {
	flag.StringVar(&Opts.InputFile, "f", "", "inpt file path (required)")
	flag.StringVar(&Opts.Environment, "e", "localhost", "load testing environment (qa50/staging/production)")
	flag.StringVar(&Opts.database, "db", "clickhouse", "which database to load test")
	flag.IntVar(&Opts.MaxThreads, "th", 1, "number of max threads for clickhouse.")
	flag.Int64Var(&Opts.NumQuery, "n", 10000, "number of total queries")
	flag.BoolVar(&Opts.ShowProgress, "p", false, "show basic progress during the test")
	flag.IntVar(&Opts.Workers, "w", 5, "number of concurrent workers")
	flag.Parse()
	if flag.NFlag() == 0 {
		printUsageAndExit()
	}
}

func printUsageAndExit() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func (r *Recorder) printResult() {
	result := r.Meter.Calc()
	errMap := make(map[string]int)
	for r := range r.Results {
		if r.Err != nil {
			errMap[r.Err.Error()]++
		} else {
			errMap["success"]++
		}
	}
	fmt.Printf("\n===========================Stats================================\n")
	fmt.Printf("Load test ran for: %v\n", r.Finish.Sub(r.Start))
	// Print pre-formatted console output.
	fmt.Printf("%s\n\n", result)
	fmt.Printf("Status code distribution:\n")
	for k, v := range errMap {
		fmt.Printf("  [%s]:\t[%d]\n", k, v)
	}
	// Print text histogram.
	fmt.Println(result.Histogram.String(50))
	fmt.Printf("\n================================================================\n")
}
