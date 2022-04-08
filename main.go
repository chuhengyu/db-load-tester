package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"

	"github.com/jmoiron/sqlx"
)

const version = "v1.0.1"

type Registry struct {
	NewClient   func() (*sqlx.DB, error)
	InputParser func(string) []interface{}
	QueryDict   []Query
}

var (
	Opts struct {
		database           string
		InputFile          string
		NumQuery           int64
		Workers            int
		Verbose            int
		MaxThreads         int
		NumCPU             int
		ConnectionPoolSize int
		QpsKnob            int
		Servers            string
		Username           string
		Password           string
		UseCache           bool
		MaxQPS             int
		ReportInterval     int
	}
	registry = map[string]*Registry{
		// DB integration goes here.
		"clickhouse": {NewClickhouseClient, ParseTabDelimited, QueryDict},
	}
)

func main() {
	initFlags()
	fmt.Printf("🌏 Database Load Tester version: %s 🌏\n", version)
	runner := NewRunner()
	// Specify number of cores to use for load testing.
	if Opts.NumCPU != 0 {
		currentCPU := runtime.NumCPU()
		runtime.GOMAXPROCS(Opts.NumCPU)
		defer runtime.GOMAXPROCS(currentCPU)
	}
	if Opts.Verbose >= 1 {
		fmt.Printf("Load tester is using %d cores.\n", runtime.NumCPU())
	}
	if r := registry[Opts.database]; r == nil {
		fmt.Printf("Unrecognized database target [%s]. Available ones are: %v\n", Opts.database, reflect.ValueOf(registry).MapKeys())
		os.Exit(1)
	}
	fmt.Printf("Running %s load tester.\n", Opts.database)
	runner.Run(registry[Opts.database])
	<-runner.Recorder.Done
	runner.Recorder.PrintResult()
}

func initFlags() {
	flag.StringVar(&Opts.InputFile, "input", "", "The input file that provides necessary query parameters (required). The file should have the expected format for the db-under-test. For example: for clickhosue, each entry should be in the form of 'environment_id\\tvisitor_id'.")
	flag.StringVar(&Opts.database, "db", "clickhouse", "Target database for load testing.")
	flag.StringVar(&Opts.Username, "username", "default", "Username used for connecting to db.")
	flag.StringVar(&Opts.Password, "password", "", "Password used for connecting to db.")
	flag.StringVar(&Opts.Servers, "servers", "localhost:9000", "The bootstrap database servers.")
	flag.Int64Var(&Opts.NumQuery, "n", 10000, "The number of total queries. WARNING: make sure the input file has more entries than this number or there is a chance that the load tester may fail.")
	flag.IntVar(&Opts.MaxThreads, "max-server-threads", 1, "Max server threads for clickhouse for each client connection.")
	flag.IntVar(&Opts.Verbose, "verbose", 0, "Option do display more info (e.g. progress in %) during the test.")
	flag.IntVar(&Opts.Workers, "workers", 5, "Number of concurrent workers for each connection.")
	flag.IntVar(&Opts.NumCPU, "cpu", 0, "Number of CPUs to use. By default (value 0), load tester uses all the cores in the host machine.")
	flag.IntVar(&Opts.ConnectionPoolSize, "connections", 1, "Number of client connections. This option comes in handy when db-under-test limits the number of queries per connection.")
	flag.IntVar(&Opts.ReportInterval, "report-interval", 1000, "The frequency (in Milliseconds) for reporting the progress.")
	flag.IntVar(&Opts.QpsKnob, "qps-knob", 1, "The knob to control the ingestion load. Lower value means heavier load.")
	flag.IntVar(&Opts.MaxQPS, "max-qps", 1000000, "Initial target ingestion QPS. For every second, the load tester performs best effort rate limiting if the current QPS goes __above__ the threshold.")
	flag.BoolVar(&Opts.UseCache, "use-cache", true, "Disable this option means clickhouse would always read data from storage disk.")
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
