package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

var argDaemon = flag.Bool("d", false, "set this parameter to run it as a server")
var argBind = flag.String("bind", "0.0.0.0:9891", "bind address for the HTTP server. Only work if -d is specified")
var argJsonFile = flag.String("json", "./task.json", "the file container a task in json format")
var argNrWorkers = flag.Int("n", 10, "number of concurrent workers")

func main() {
	flag.Parse()
	N := *argNrWorkers
	if N <= 0 {
		N = 1
	}
	StartWorkers(N)
	server := NewTaskServer()
	var err error
	if *argDaemon {
		err = http.ListenAndServe(*argBind, server)
	} else if len(*argJsonFile) > 0 {
		var f io.ReadCloser
		f, err = os.Open(*argJsonFile)
		if err == nil {
			defer f.Close()
			server.ServeJson(os.Stdout, f)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
