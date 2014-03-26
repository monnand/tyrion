package main

import "flag"

var argDaemon = flag.Bool("d", false, "set this parameter to run it as a server")
var argTaskFile = flag.String("json", "./task.json", "the file container a task in json format")
var argNrWorkers = flag.Int("n", 10, "number of concurrent workers")

func main() {
	flag.Parse()
	StartWorkers(*argNrWorkers)
}
