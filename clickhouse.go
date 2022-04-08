package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/XSAM/otelsql"
	"github.com/jmoiron/sqlx"
)

var (
	defaultTimeout = 60 * time.Second
	queryDict      = map[int]Query{
		0: {"findEvents", "select environment_id, visitor_id, event_id, MAX(event_count), MIN(toUnixTimestamp64Milli(created_at)) AS created_at, MAX(toUnixTimestamp64Milli(updated_at)) AS updated_at " +
			"from userleap.visitor_events where environment_id=? AND visitor_id=? group by environment_id, visitor_id, event_id"},
		1: {"findSessions", "select toUnixTimestamp64Milli(created_at) AS created_at, toUnixTimestamp64Milli(updated_at) AS updated_at " +
			"from userleap.visitor_sessions where environment_id=? AND visitor_id=? order by updated_at desc limit 1"},
		2: {"findAttributes", "select * from userleap.visitor_attributes where environment_id=? AND visitor_id=?"},
	}
)

// Connect connects to a Clickhouse server.
func runClickhouse(ctx context.Context, recorder *Recorder) {
	fmt.Println("Running clickhouse load tester.")
	config := &ClientConfig{
		Options:        []string{"database=userleap", "timeout=30", fmt.Sprintf("max_threads=%d", Opts.MaxThreads)},
		MaxConnections: Opts.Workers,
	}
	if Opts.Environment == "localhost" {
		config.Servers = []string{"localhost:9000"}
		config.Username = "default"
		config.Password = ""
	} else {
		config.Servers = []string{fmt.Sprintf("%s-clickhouse.clickhouse-operator.svc.cluster.local:9000", Opts.Environment)}
		cmd := exec.Command("zsh", "-c", "aws ssm get-parameter --with-decryption --name /staging/clickhouse/password/admin | jq -r '.Parameter.Value'")
		if resp, err := cmd.Output(); err != nil {
			fmt.Printf("Failed to retrieve CH password: %v\n", err)
			return
		} else {
			config.Username = "datagw"
			config.Password = string(resp[:len(resp)-1])
		}
	}
	db, err := newClient(config)
	if err != nil {
		fmt.Printf("Failed to create ClickHouse client: %v\n", err)
		os.Exit(1)
	}
	start(db, ctx, queryDict, recorder)
	recorder.Done <- true
}

func newClient(config *ClientConfig) (*Client, error) {
	if len(config.Servers) == 0 {
		return nil, fmt.Errorf("no servers provided")
	}
	fmt.Printf("Connecting to ClickHouse at %q, options: %q\n", config.Servers, config.Options)
	options := []string{
		fmt.Sprintf("username=%s", config.Username),
		fmt.Sprintf("password=%s", config.Password),
	}
	if len(config.Servers) > 1 {
		altHosts := strings.Join(config.Servers[1:], ",")
		options = append(options, fmt.Sprintf("alt_hosts=%s", altHosts))
	}
	options = append(options, config.Options...)
	allOptions := strings.Join(options, "&")
	chUri := fmt.Sprintf("tcp://%s?%s", config.Servers[0], allOptions)
	driver, err := otelsql.Register("clickhouse", "clickhouse")
	if err != nil {
		return nil, fmt.Errorf("registering ClickHouse driver for instrumentation: %v", err)
	}
	db, err := sqlx.Connect(driver, chUri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(config.MaxConnections)
	return &Client{db}, nil
}
