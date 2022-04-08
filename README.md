# Database Load Tester

A simple load tester for databases, currently supporting ClickHouse.

## Getting Started

### Installing

The load tester can be built with bazel [cross-compile](https://github.com/bazelbuild/rules_go#how-do-i-cross-compile). The full list of platforms can be found [here](https://github.com/bazelbuild/rules_go/blob/d606ba2c123826f153bbd037cf46e80b5eb8dc3e/go/private/platforms.bzl). Example:

Linux: `bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //utils/db-load-tester`

MacOS: `bazel build --platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64 //utils/db-load-tester`

The output binary will be generated under the bazel bin folder like `/tmp/monorepo-bazel-bin/utils/db-load-tester/db-load-tester_/db-load-tester`.

### Usage

#### Sample exec command:

```
./db-load-tester-linux \
  --input=data.dat \
  --n=1000 \
  --connections=1 \
  --workers=5 \
  --max-qps=300
```

#### Flags

```
  -connections int
    	Number of client connections. This option comes in handy when db-under-test limits the number of queries per connection. (default 1)
  -cpu int
    	Number of CPUs to use. By default (value 0), load tester uses all the cores in the host machine.
  -db string
    	Target database for load testing. (default "clickhouse")
  -input string
    	The input file that provides necessary query parameters (required). The file should have the expected format for the db-under-test. For example: for clickhosue, each entry should be in the form of 'environment_id\tvisitor_id'.
  -max-qps int
    	Initial target ingestion QPS. For every second, the load tester performs best effort rate limiting if the current QPS goes __above__ the threshold. (default 1000000)
  -max-server-threads int
    	Max server threads for clickhouse for each client connection. (default 1)
  -n int
    	The number of total queries. WARNING: make sure the input file has more entries than this number or there is a chance that the load tester may fail. (default 10000)
  -password string
    	Password used for connecting to db.
  -qps-knob int
    	The knob to control the ingestion load. Lower value means heavier load. (default 1)
  -report-interval int
    	The frequency (in Milliseconds) for reporting the progress. (default 500)
  -servers string
    	The bootstrap database servers. (default "localhost:9000")
  -use-cache
    	Disable this option means clickhouse would always read data from storage disk. (default true)
  -username string
    	Username used for connecting to db. (default "default")
  -verbose int
    	Option do display more info (e.g. progress in %) during the test.
  -workers int
    	Number of concurrent workers for each connection. (default 5)
```

### Generate the input file

Improvise :)

### Sample output

```
===========================Stats================================
Elapsed time: 3.054524008s
20000 samples of 20000 events
Cumulative:	5m2.322367465s
HMean:		13.383172ms
Avg.:		15.116118ms
p50: 		14.525208ms
p75:		18.047638ms
p95:		24.022192ms
p99:		29.290625ms
p999:		47.142027ms
Long 5%:	27.981394ms
Short 5%:	6.517714ms
Max:		50.77058ms
Min:		2.79898ms
Range:		47.9716ms
StdDev:		5.183258ms
Rate/sec.:	6547.67

Status code distribution:
  [success]:	[20000]

     2.798ms - 7.596ms ------
    7.596ms - 12.393ms ------------------------------------
    12.393ms - 17.19ms --------------------------------------------------
    17.19ms - 21.987ms ---------------------------
   21.987ms - 26.784ms ----------
   26.784ms - 31.581ms --
   31.581ms - 36.379ms -
   36.379ms - 41.176ms -
   41.176ms - 45.973ms -
    45.973ms - 50.77ms -

================================================================
```
