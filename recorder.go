package main

import (
	"fmt"
	"time"

	"github.com/jamiealquiza/tachymeter"
)

type Recorder struct {
	Meter  *tachymeter.Tachymeter
	Done   chan bool
	Errors chan error
	Start  time.Time
	End    time.Time
}

func NewReorder() *Recorder {
	return &Recorder{
		Meter:  tachymeter.New(&tachymeter.Config{Size: int(Opts.NumQuery)}),
		Done:   make(chan bool, 1),
		Errors: make(chan error, Opts.NumQuery),
	}
}

func (r *Recorder) PrintResult() {
	r.Meter.SetWallTime(r.End.Sub(r.Start))
	result := r.Meter.Calc()
	errMap := make(map[string]int)
	for err := range r.Errors {
		if err != nil {
			errMap[err.Error()]++
		} else {
			errMap["success"]++
		}
	}
	fmt.Printf("\n===========================Stats================================\n")
	fmt.Printf("Elapsed time: %v\n", r.Meter.WallTime)
	// Print pre-formatted console output.
	fmt.Printf("%s\n\n", result)
	fmt.Printf("Status code distribution:\n")
	for k, v := range errMap {
		fmt.Printf("  [%s]:\t[%d]\n", k, v)
	}
	fmt.Println()
	// Print text histogram.
	fmt.Println(result.Histogram.String(50))
	fmt.Printf("================================================================\n")
}

func (r *Recorder) Close() {
	r.End = time.Now()
	close(r.Done)
	close(r.Errors)
}
