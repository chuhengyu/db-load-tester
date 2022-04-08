package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
)

type Client struct {
	*sqlx.DB
}

type Query struct {
	name  string
	query string
}

type ClientConfig struct {
	Servers        []string
	Username       string
	Password       string
	Options        []string
	MaxConnections int
}

func start(cl *Client, ctx context.Context, q map[int]Query, recorder *Recorder) {
	rand.Seed(time.Now().UnixNano())
	ch, err := processFile()
	if err != nil {
		fmt.Printf("Error processing input file: %v\n", err)
		return
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	var wg sync.WaitGroup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	fmt.Println("Starting workers...")
	recorder.Start = time.Now()
	defer close(recorder.Results)
	total := 0
	for i := 0; i < Opts.Workers; i++ {
		wg.Add(1)
		go func(worker int) {
			// 		ticker := time.NewTicker(AgentInterruptPeriod)
			// defer ticker.Stop()
			loop := 0
			for s := range ch {
				if Opts.ShowProgress && total != 0 && total%1000 == 0 {
					fmt.Printf("Executed __roughly__ %d queries.\n", total)
				}
				total++
				// select {
				// case <-ctx.Done():
				// 	return false, nil
				// case <-ticker.C:
				// 	recorder.Add(responseTimes)
				// 	responseTimes = responseTimes[:0]
				// default:
				// 	// nothing to do
				// }

				array := strings.Split(s, "\t") // envID, vID
				envID := strings.TrimSpace(array[0])
				vID := strings.TrimSpace(array[1])
				q := queryDict[rand.Intn(len(queryDict))]
				// fmt.Printf("worker %d in loop %d executing %s\n", worker, loop, q.name)
				start := time.Now()
				resp, e := cl.QueryContext(timeoutCtx, q.query, envID, vID)
				if resp != nil {
					resp.Close()
				}
				duration := time.Since(start)
				recorder.Results <- &QueryResult{Err: e}
				recorder.Meter.AddTime(duration)
				select {
				case <-timeoutCtx.Done():
					fmt.Printf("Context canceled. Shutting down worker %d\n", worker)
					wg.Done()
					return
				case sig := <-sigCh:
					fmt.Printf("Received signal %v, worker %d shutting down.", sig, worker)
					wg.Done()
					cancel()
					return
				default:
					// non-blocking
				}
				loop++
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	recorder.Finish = time.Now()
}

func processFile() (chan string, error) {
	// p := regexp.MustCompile(`^[a-zA-Z0-9]{5,}[ \t]{1,}[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	fmt.Print("Loading input file into memory...")
	ch := make(chan string, Opts.NumQuery)
	defer close(ch)
	file, err := os.Open(Opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s: %v", Opts.InputFile, err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			file.Close()
			break
		}
		s := string(line)
		// if !p.MatchString(s) {
		// 	return nil, fmt.Errorf("invalid input format, expected: environment_id\\tvisitor_id, got: %v\n", s)
		// }
		select {
		case ch <- s:
			// line recorded.
		default:
			// channel full. No longer need records.
			fmt.Println("Done.")
			return ch, nil
		}
	}
	fmt.Println("Done.")
	return ch, nil
}
