package main

import (
	"time"

	"github.com/jamiealquiza/tachymeter"
)

type Recorder struct {
	Meter   *tachymeter.Tachymeter
	Start   time.Time
	Finish  time.Time
	Done    chan (bool)
	Results chan *QueryResult
}

type QueryResult struct {
	Err error
}

func NewReorder() *Recorder {
	return &Recorder{
		Meter:   tachymeter.New(&tachymeter.Config{Size: int(Opts.NumQuery)}),
		Done:    make(chan bool, 1),
		Results: make(chan *QueryResult, Opts.NumQuery),
	}
}
