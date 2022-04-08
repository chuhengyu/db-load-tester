package main

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	_ "github.com/ClickHouse/clickhouse-go"
)

var (
	QueryDict = []Query{
	}
)

func NewClickhouseClient() (*sqlx.DB, error) {
	config := &clickhouse.ClientConfig{
		Options: []string{"timeout=30", "connection_open_strategy=time_random",
			fmt.Sprintf("max_threads=%d", Opts.MaxThreads)},
		MaxConnections: Opts.Workers,
	}
	if !Opts.UseCache {
		config.Options = append(config.Options, "min_bytes_to_use_direct_io=1")
	}
	config.Servers = strings.Split(Opts.Servers, ",")
	config.Username = Opts.Username
	config.Password = Opts.Password
	db, err := clickhouse.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClickHouse client: %v", err)
	}
	return db.DB, nil
}

// Extracts query parameters from each line of input.
func ParseTabDelimited(data string) []interface{} {
	// For clickhouse the data format must be "envID\tvID"
	array := strings.Split(data, "\t")
	return []interface{}{strings.TrimSpace(array[0]), strings.TrimSpace(array[1])}
}
