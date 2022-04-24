package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func rateLimitServer(r *Runner) {
	http.HandleFunc("/ratelimit", func(w http.ResponseWriter, req *http.Request) {
		if body, err := ioutil.ReadAll(req.Body); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("Bad request: %v\n Try again! Expected body format: '<command>=<value>', availabe commands are [ set | inc | dec ]", err)))
		} else if kv := strings.Split(string(body), "="); len(kv) != 2 {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintln("Bad request. Try again! Expected body format: '<command>=<value>', availabe commands are [ set | inc | dec ]")))
		} else {
			cmd := kv[0]
			val, err := strconv.Atoi(kv[1])
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(fmt.Sprintln("Bad qps value. Try again! Just provide a valid integer, how hard could it be O.o")))
				return
			}
			switch cmd {
			case "set":
				if val <= 0 {
					w.WriteHeader(400)
					w.Write([]byte(fmt.Sprintf("QPS cannot go below 1. Try again! Current QPS is: %d\n", Opts.MaxQPS)))
					return
				}
				Opts.MaxQPS = val
				r.TickerInterval = tickerInterval()
				w.WriteHeader(200)
				w.Write([]byte(fmt.Sprintf("Set QPS to: %v\n", Opts.MaxQPS)))
				r.broadcast()
			case "inc":
				Opts.MaxQPS += val
				r.TickerInterval = tickerInterval()
				w.WriteHeader(200)
				w.Write([]byte(fmt.Sprintf("Increment QPS by: %v. New QPS is: %d\n", val, Opts.MaxQPS)))
				r.broadcast()
			case "dec":
				if Opts.MaxQPS <= val {
					w.WriteHeader(400)
					w.Write([]byte(fmt.Sprintf("QPS cannot go below 1. Try again! Current QPS is: %d\n", Opts.MaxQPS)))
					return
				}
				Opts.MaxQPS -= val
				r.TickerInterval = tickerInterval()
				w.WriteHeader(200)
				w.Write([]byte(fmt.Sprintf("Decrement QPS by: %v. New QPS is: %d\n", val, Opts.MaxQPS)))
				r.broadcast()
			default:
				w.WriteHeader(400)
				w.Write([]byte(fmt.Sprintln("Unrecognized command. Try again! Availabe commands are [ set | inc | dec ]")))
			}
		}
	})
	fmt.Println("🌏 Starting rate limit server at localhost:8090/ratelimit")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Printf("Something went wrong with rate limit server: %v\n", err)
	}
}
