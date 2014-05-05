package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func init() {
	RegisterPlugin(&RetryPluginFactory{})
}

type RetryPluginFactory struct {
}

func (self *RetryPluginFactory) String() string {
	return "retry"
}

func (self *RetryPluginFactory) stringToIntList(str string) (l []int, err error) {
	lstr := strings.Split(str, ",")
	ret := make([]int, 0, len(lstr))

	for _, v := range lstr {
		n := 0
		n, err = strconv.Atoi(v)
		if err != nil {
			return
		}
		ret = append(ret, n)
	}
	l = ret
	return
}

func (self *RetryPluginFactory) NewPlugin(params map[string]string, rest ResponseReader) (rr ResponseReader, err error) {
	if rest == nil {
		err = fmt.Errorf("retry replut cannot be the last plugin")
		return
	}
	retryStatusesStr := "500"
	ok := false
	if retryStatusesStr, ok = params["retry-when"]; !ok {
		retryStatusesStr = "500"
	}
	retryStatuses, err := self.stringToIntList(retryStatusesStr)
	if err != nil {
		return
	}
	maxTimeOut := "10s"
	if maxTimeOut, ok = params["max-wait"]; !ok {
		maxTimeOut = "10s"
	}
	maxWait, err := time.ParseDuration(maxTimeOut)
	if err != nil {
		return
	}
	ret := &RetryPlugin{
		rest:            rest,
		maxTimeOut:      maxWait,
		retryOnStatuses: retryStatuses,
	}
	rr = ret
	return
}

type RetryPlugin struct {
	rest            ResponseReader
	maxTimeOut      time.Duration
	retryOnStatuses []int
}

func (self *RetryPlugin) Close() error {
	if self.rest == nil {
		return fmt.Errorf("retry plugin needs rest")
	}
	return self.rest.Close()
}

func (self *RetryPlugin) shouldRetry(resp *Response) bool {
	if resp == nil {
		return true
	}
	for _, s := range self.retryOnStatuses {
		if resp.Status == s {
			return true
		}
	}
	return false
}

func (self *RetryPlugin) ReadResponse(
	req *Request,
	env *Env,
) (resp *Response, updates *Env, err error) {
	if self.rest == nil {
		err = fmt.Errorf("retry plugin needs rest")
		return
	}

	resp, updates, err = self.rest.ReadResponse(req, env)
	if err != nil {
		return
	}
	for self.shouldRetry(resp) {
		sleep := time.Duration(rand.Int63n(int64(self.maxTimeOut)))
		if sleep < 1*time.Second {
			sleep = time.Second
		}
		time.Sleep(sleep)
		resp, updates, err = self.rest.ReadResponse(req, env)
		if err != nil {
			return
		}
	}
	return
}
