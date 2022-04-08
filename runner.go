package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/schollz/progressbar/v3"
)

const defaultTimeout = 5 * 60 * time.Second

type Runner struct {
	InputChannels  []chan string
	InputParser    func(string) []interface{}
	clients        []*Client
	Recorder       *Recorder
	TickerInterval time.Duration
}

type Client struct {
	*sqlx.DB
	QueryDict    []Query
	qpsBroadcast []chan struct{}
}

type Query struct {
	Name  string
	Query string
}

func NewRunner() *Runner {
	r := &Runner{
		clients:        []*Client{},
		Recorder:       NewReorder(),
		TickerInterval: tickerInterval(),
	}
	err := r.processFile()
	if err != nil {
		fmt.Printf("failed to load file: %v\n", err)
		os.Exit(1)
	}
	return r
}

// Starts each connection with a new routine.
func (r *Runner) Run(registry *Registry) {
	if err := r.initFrom(registry); err != nil {
		fmt.Printf("failed at init: %v", err)
		os.Exit(1)
	}
	defer r.close()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	rand.Seed(time.Now().UnixNano())
	// report progress during the test.
	go r.showProgress(timeoutCtx)
	go rateLimitServer(r)
	r.Recorder.Start = time.Now()
	var connectionGroup sync.WaitGroup
	for i, _ := range r.clients {
		// sleep a short random period of time to unsync tickers between clients.
		// time.Sleep(time.Duration(rand.Intn(int(r.TickerInterval))))
		connectionGroup.Add(1)
		go r.start(timeoutCtx, &connectionGroup, i)
	}
	// listens to the interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		if Opts.Verbose >= 1 {
			fmt.Printf("Received signal [%v]. Shutting down all workers...\n", sig)
		}
		cancel()
	}()
	connectionGroup.Wait()
}

func (r *Runner) initFrom(registry *Registry) error {
	r.InputParser = registry.InputParser
	for i := 0; i < Opts.ConnectionPoolSize; i++ {
		db, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create new client for clickhouse: %v", err)
		}
		cl := &Client{db, registry.QueryDict, []chan struct{}{}}
		// all workers are subscribed to the broadcast mechanism by default.
		for j := 0; j < Opts.Workers; j++ {
			cl.qpsBroadcast = append(cl.qpsBroadcast, make(chan struct{}))
		}
		r.clients = append(r.clients, cl)
	}
	return nil
}

// Each connection spins up ${Opts.Workers} number of workers that all consume from the same channel.
func (r *Runner) start(ctx context.Context, connectionGroup *sync.WaitGroup, index int) {
	defer connectionGroup.Done()
	var workerGroup sync.WaitGroup
	rand.Seed(time.Now().UnixNano())
	client := r.clients[index]
	for i := 0; i < Opts.Workers; i++ {
		workerGroup.Add(1)
		go func(worker int) {
			defer workerGroup.Done()
			ticker := time.NewTicker(r.TickerInterval)
			defer ticker.Stop()
			// Tracks the processing time __besides__ the actual query time. Ignore the first number.
			debugStart := time.Now()
			for s := range r.InputChannels[index] {
				if Opts.Verbose >= 2 && index == 0 && worker == 0 {
					fmt.Printf("\n[MISC] for-loop took %v\n", time.Since(debugStart))
				}
				q := client.QueryDict[rand.Intn(len(client.QueryDict))]
				start := time.Now()
				resp, e := client.QueryxContext(ctx, q.Query, r.InputParser(s)...)
				if resp != nil {
					resp.Close()
				}
				duration := time.Since(start)
				debugStart = time.Now()
				r.Recorder.Errors <- e
				r.Recorder.Meter.AddTime(duration)
				select {
				case <-ctx.Done():
					if Opts.Verbose >= 1 {
						fmt.Printf("Context canceled. Shutting down worker %d-%d\n", index, worker)
					}
					return
				case <-client.qpsBroadcast[worker]:
					// it's a known issue that ticker.Reset(...) doesn't drain the buffer, so let's get a new one.
					// this only happens if/when we need to tune the QPS, theoretically not much of a burden for GC.
					ticker.Stop()
					ticker = time.NewTicker(r.TickerInterval)
				case <-ticker.C:
					// Rate limit
				}
			}
		}(i)
	}
	workerGroup.Wait()
}

// Load the input file into memory for faster read.
// The file is evenly splitted into separate channels, one for each connection.
func (r *Runner) processFile() error {
	fmt.Print("Loading input file into memory... ")
	bar := progressbar.NewOptions64(
		-1,
		progressbar.OptionSetDescription("Loading input file into memory..."),
		progressbar.OptionSpinnerType(70),
	)
	start := time.Now()
	var chs []chan string
	file, err := os.Open(Opts.InputFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %s: %v", Opts.InputFile, err)
	}
	defer file.Close()
	ch := newChan(len(chs))
	lineRead := 0
	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			file.Close()
			break
		}
		s := string(line)
		select {
		case ch <- s:
			// line recorded.
		default:
			bar.Add(1)
			time.Sleep(100 * time.Millisecond)
			// channel full. Get a new one.
			close(ch)
			chs = append(chs, ch)
			ch = newChan(len(chs))
			ch <- s
		}
		lineRead++
		if lineRead == int(Opts.NumQuery) {
			break
		}
	}
	close(ch)
	chs = append(chs, ch)
	fmt.Print("Done\n")
	if Opts.Verbose >= 1 {
		fmt.Printf("[INFO] Processing file took %v; Generated %d channels. Each of them has ", time.Since(start), len(chs))
		for _, c := range chs {
			fmt.Printf("%d ", len(c))
		}
		fmt.Println("messages respectively.")
	}
	r.InputChannels = chs
	return nil
}

func (r *Runner) showProgress(ctx context.Context) {
	bar := progressbar.Default(100)
	ticker := time.NewTicker(time.Duration(Opts.ReportInterval) * time.Millisecond)
	prevCount := 0
	prevTimestamp := r.Recorder.Start
	for {
		select {
		case <-ctx.Done():
			bar.Finish()
			return
		case <-ticker.C:
			count := len(r.Recorder.Errors)
			now := time.Now()
			percentage := float64(count) / float64(Opts.NumQuery) * 100
			bar.Describe(fmt.Sprintf("Overall QPS: %.1f, Current QPS: %.1f ", float64(count)/time.Since(r.Recorder.Start).Seconds(), float64(count-prevCount)/now.Sub(prevTimestamp).Seconds()))
			bar.Set(int(percentage))
			prevCount = count
			prevTimestamp = now
		}
	}
}

func (r *Runner) broadcast() {
	for _, cl := range r.clients {
		cl.broadcast()
	}
}

func (r *Runner) close() {
	r.Recorder.Close()
	for _, client := range r.clients {
		client.close()
	}
}

// Best effort to evenly distribute data into channels based on the number of connections specified.
// Nonethelss, the last channel has more data due to modulo residue.
// 'numSoFar' is the number of channels we have created so far.
func newChan(numSoFar int) chan string {
	unit := Opts.NumQuery / int64(Opts.ConnectionPoolSize)
	if (Opts.NumQuery-int64(numSoFar)*unit)/unit > 1 {
		return make(chan string, unit)
	}
	return make(chan string, Opts.NumQuery-int64(numSoFar)*unit)
}

func tickerInterval() time.Duration {
	return time.Second / time.Duration(Opts.MaxQPS) * time.Duration(Opts.ConnectionPoolSize) * time.Duration(Opts.Workers)
}

func (client *Client) broadcast() {
	for _, w := range client.qpsBroadcast {
		w <- struct{}{}
	}
}

func (client *Client) close() {
	for _, ch := range client.qpsBroadcast {
		close(ch)
	}
}
