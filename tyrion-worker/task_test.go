package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"
)

// A simply key value store for test purpose
type kvStoreResponseReader struct {
	store map[string]string
	lock  sync.RWMutex
}

func newKvStore() ResponseReader {
	ret := new(kvStoreResponseReader)
	ret.store = make(map[string]string, 10)
	return ret
}

func (self *kvStoreResponseReader) ReadResponse(tag, url, method, content string, params url.Values, headers http.Header) (status int, body io.ReadCloser, err error) {
	var key string
	if keys, ok := params["key"]; ok && len(keys) > 0 {
		key = keys[0]
	} else {
		err = fmt.Errorf("No key specified")
		status = 404
		return
	}
	switch method {
	case "POST":
		fallthrough
	case "PUT":
		self.lock.Lock()
		defer self.lock.Unlock()
		if k, ok := self.store[key]; ok {
			err = fmt.Errorf("Already has key %v", k)
			status = 404
			return
		}
		if value, ok := params["value"]; ok && len(value) > 0 {
			self.store[key] = value[0]
			status = 200
		} else {
			err = fmt.Errorf("No value for key %v", key)
			status = 404
			return
		}
	case "DELETE":
		self.lock.Lock()
		defer self.lock.Unlock()
		if _, ok := self.store[key]; !ok {
			err = fmt.Errorf("delete a non-exist key %v", key)
			status = 404
			return
		}
		delete(self.store, key)
		status = 200
	case "GET":
		self.lock.RLock()
		defer self.lock.RUnlock()
		if v, ok := self.store[key]; ok {
			body = ioutil.NopCloser(bytes.NewBufferString(v))
			status = 200
		} else {
			err = fmt.Errorf("get a non-exist key %v", key)
			status = 404
			return
		}
	}
	return
}

func genConcurrentSetOps(kv map[string]string) []*ActionSpec {
	ret := make([]*ActionSpec, 0, len(kv))
	for k, v := range kv {
		spec := new(ActionSpec)
		spec.Tag = "set"
		spec.URLTemplate = "http://localhost/set"
		spec.Method = "POST"
		spec.Params = make(map[string][]string, 2)
		spec.Params["key"] = []string{k}
		spec.Params["value"] = []string{v}
		spec.ExpStatus = 200
		ret = append(ret, spec)
	}
	return ret
}

func genConcurrentDelOps(kv map[string]string) []*ActionSpec {
	ret := make([]*ActionSpec, 0, len(kv))
	for k, _ := range kv {
		spec := new(ActionSpec)
		spec.Tag = "del"
		spec.URLTemplate = "http://localhost/del"
		spec.Method = "DELETE"
		spec.Params = make(map[string][]string, 2)
		spec.Params["key"] = []string{k}
		spec.ExpStatus = 200
		ret = append(ret, spec)
	}
	return ret
}

func genConcurrentGetOps(kv map[string]string) []*ActionSpec {
	ret := make([]*ActionSpec, 0, len(kv))
	for k, v := range kv {
		spec := new(ActionSpec)
		spec.Tag = "get"
		spec.URLTemplate = "http://localhost/get"
		spec.Method = "GET"
		spec.Params = make(map[string][]string, 2)
		spec.Params["key"] = []string{k}
		spec.ExpStatus = 200
		spec.RespTemp = v
		// spec.RespTemp = "somevalue"
		ret = append(ret, spec)
	}
	return ret
}

func TestWorker(t *testing.T) {
	N := 2
	kv := make(map[string]string, N)
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("key%v", i)
		value := fmt.Sprintf("value%v", i)
		kv[key] = value
	}

	StartWorkers(1)

	taskSpec := new(TaskSpec)
	actions := make([][]*ActionSpec, 0, 3)

	a := genConcurrentSetOps(kv)
	actions = append(actions, a)

	a = genConcurrentGetOps(kv)
	actions = append(actions, a)

	a = genConcurrentDelOps(kv)
	actions = append(actions, a)

	taskSpec.Actions = actions

	rr := newKvStore()
	worker := taskSpec.GetWorker(rr)
	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range errChan {
			t.Errorf("Error: %v", err)
		}
	}()
	worker.Execute(errChan)
	close(errChan)
	wg.Wait()
}
