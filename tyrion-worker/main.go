package main

import (
	"bytes"
	"flag"
	"io"
	"net/http"
	"net/url"
)

var argDaemon = flag.Bool("d", false, "set this parameter to run it as a server")
var argTaskFile = flag.String("json", "./task.json", "the file container a task in json format")
var argNrWorkers = flag.Int("n", 10, "number of concurrent workers")

type HttpResponseReader struct {
}

func (self *HttpResponseReader) ReadResponse(tag, url, method, content string, params url.Values, headers http.Header) (status int, body io.ReadCloser, err error) {
	req, err := http.NewRequest(method, url, bytes.NewBufferString(content))
	if err != nil {
		return
	}
	req.Header = headers
	req.Form = params
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	status = resp.StatusCode
	body = resp.Body
	return
}

func main() {
	flag.Parse()
	StartWorkers(*argNrWorkers)
}
