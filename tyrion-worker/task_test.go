package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"sync"
	"testing"
)

// A simply key value store for test purpose
type kvStoreResponseReader struct {
	store map[string]string
	lock  sync.RWMutex
}

func (self *kvStoreResponseReader) Size() int {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return len(self.store)
}

func newKvStore() ResponseReader {
	ret := new(kvStoreResponseReader)
	ret.store = make(map[string]string, 10)
	return ret
}

func (self *kvStoreResponseReader) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	method := req.Method
	params := req.Params

	status := 404
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
			body := ioutil.NopCloser(bytes.NewBufferString(v))
			status = 200
			resp = new(Response)
			resp.Body = body
			resp.Status = status
		} else {
			err = fmt.Errorf("get a non-exist key %v", key)
			status = 404
			return
		}
	}
	return
}

func genConcurrentSetOps(kv map[string]string) *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = true
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
	ca.Actions = ret
	return ca
}

func genConcurrentDelOps(kv map[string]string) *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = true
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
	ca.Actions = ret
	return ca
}

func genConcurrentGetOps(kv map[string]string) *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = true
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
		spec.MustMatch = true
		ret = append(ret, spec)
	}
	ca.Actions = ret
	return ca
}

func genConcurrentGetOpsWithWrongRespTemp(kv map[string]string) *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = true
	ret := make([]*ActionSpec, 0, len(kv))
	for k, v := range kv {
		spec := new(ActionSpec)
		spec.Tag = "get"
		spec.URLTemplate = "http://localhost/get"
		spec.Method = "GET"
		spec.Params = make(map[string][]string, 2)
		spec.Params["key"] = []string{k}
		spec.ExpStatus = 200
		spec.RespTemp = v + "somevalue"
		spec.MustMatch = true
		ret = append(ret, spec)
	}
	ca.Actions = ret
	return ca
}

func TestKeyValueStoreWorkers(t *testing.T) {
	N := 100
	kv := make(map[string]string, N)
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("key%v", i)
		value := fmt.Sprintf("value%v", i)
		kv[key] = value
	}

	StartWorkers(20)
	defer StopAllWorkers()

	taskSpec := new(TaskSpec)
	actions := make([]*ConcurrentActions, 0, 3)

	a := genConcurrentSetOps(kv)
	actions = append(actions, a)

	a = genConcurrentGetOps(kv)
	actions = append(actions, a)

	a = genConcurrentDelOps(kv)
	actions = append(actions, a)

	taskSpec.ConcurrentActions = actions

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
	if kvrr, ok := rr.(*kvStoreResponseReader); ok {
		if kvrr.Size() != 0 {
			t.Errorf("Still have %v elements in the storage", kvrr.Size())
		}
	} else {
		t.Errorf("not a kvStoreResponseReader")
	}
}

func TestWorkerOnMatchFailed(t *testing.T) {
	N := 100
	kv := make(map[string]string, N)
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("key%v", i)
		value := fmt.Sprintf("value%v", i)
		kv[key] = value
	}

	StartWorkers(20)
	defer StopAllWorkers()

	taskSpec := new(TaskSpec)
	actions := make([]*ConcurrentActions, 0, 3)

	a := genConcurrentSetOps(kv)
	actions = append(actions, a)

	a = genConcurrentGetOpsWithWrongRespTemp(kv)
	actions = append(actions, a)

	a = genConcurrentDelOps(kv)
	actions = append(actions, a)

	taskSpec.ConcurrentActions = actions

	rr := newKvStore()
	worker := taskSpec.GetWorker(rr)
	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	nrErrors := 0
	go func() {
		defer wg.Done()
		for _ = range errChan {
			nrErrors++
		}
	}()
	worker.Execute(errChan)
	close(errChan)
	wg.Wait()
	if nrErrors != len(kv) {
		t.Errorf("Only received %v errors. should be %v", nrErrors, len(kv))
	}
}

type userProfile struct {
	Name string
	Info map[string]string
}

type userInfoDb struct {
	profiles map[string]*userProfile
	lock     sync.RWMutex
}

func newUserInfoDb() ResponseReader {
	ret := new(userInfoDb)
	ret.profiles = make(map[string]*userProfile, 100)
	return ret
}

func (self *userInfoDb) getUserName(params url.Values) (user string, err error) {
	if u, ok := params["user"]; ok && len(u) > 0 {
		user = u[0]
	} else {
		err = fmt.Errorf("no user")
		return
	}
	return
}

func (self *userInfoDb) getUserProfile(params url.Values) (user *userProfile, err error) {
	ret := new(userProfile)
	ret.Info = make(map[string]string, len(params))
	for k, v := range params {
		if len(v) == 0 {
			continue
		}
		if k == "user" {
			ret.Name = v[0]
			continue
		}
		ret.Info[k] = v[0]
	}
	if len(ret.Name) == 0 {
		err = fmt.Errorf("No username")
		return
	}
	user = ret
	return
}

func (self *userInfoDb) addUser(params url.Values) (status int, body io.ReadCloser, err error) {
	status = 404
	userName, err := self.getUserName(params)
	if err != nil {
		return
	}
	profile, err := self.getUserProfile(params)
	if err != nil {
		return
	}
	self.lock.Lock()
	defer self.lock.Unlock()

	if _, ok := self.profiles[userName]; ok {
		err = fmt.Errorf("already exist: %v", userName)
		return
	}
	self.profiles[userName] = profile
	status = 200
	return
}

func (self *userInfoDb) listUsers(params url.Values) (status int, body io.ReadCloser, err error) {
	status = 404
	self.lock.RLock()
	defer self.lock.RUnlock()

	var retBody bytes.Buffer
	for k, _ := range self.profiles {
		fmt.Fprintf(&retBody, "User: %v\n", k)
	}
	status = 200
	body = ioutil.NopCloser(&retBody)
	return
}

func (self *userInfoDb) getUserInfo(params url.Values) (status int, body io.ReadCloser, err error) {
	status = 404
	userName, err := self.getUserName(params)
	if err != nil {
		return
	}
	infoKey := ""
	if k, ok := params["info"]; ok && len(k) > 0 {
		infoKey = k[0]
	} else {
		err = fmt.Errorf("NoInfo")
		return
	}
	self.lock.RLock()
	defer self.lock.RUnlock()
	var buf bytes.Buffer
	if user, ok := self.profiles[userName]; ok {
		if infoKey == "name" {
			fmt.Fprintf(&buf, "%s", user.Name)
		} else {
			if info, ok := user.Info[infoKey]; ok {
				fmt.Fprintf(&buf, "%s", info)
			} else {
				return
			}
		}
	} else {
		return
	}
	body = ioutil.NopCloser(&buf)
	status = 200
	return
}

func (self *userInfoDb) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	tag := req.Tag
	params := req.Params
	var ret Response
	switch tag {
	case "set":
		ret.Status, ret.Body, err = self.addUser(params)
	case "get":
		ret.Status, ret.Body, err = self.getUserInfo(params)
	case "list":
		ret.Status, ret.Body, err = self.listUsers(params)
	default:
		err = fmt.Errorf("Unknown tag: %v", tag)
	}
	if err == nil {
		resp = &ret
	}
	return
}

func genConcurrentAddUserOps(users []string) *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = true

	ret := make([]*ActionSpec, 0, len(users))
	for _, u := range users {
		a := new(ActionSpec)
		a.Tag = "set"
		a.Method = "POST"
		a.Params = make(map[string][]string, 2)
		a.Params["user"] = []string{u}
		a.Params["somekey"] = []string{"something"}
		ret = append(ret, a)
	}
	ca.Actions = ret
	return ca
}

func genListUserOp() *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = false
	ret := make([]*ActionSpec, 1)
	ret[0] = new(ActionSpec)
	ret[0].Tag = "list"
	ret[0].Method = "GET"
	ret[0].RespTemp = "User:\\s*(?P<username>([a-zA-Z0-9]+))"
	ca.Actions = ret
	return ca
}

func genReadUserinfoOps() *ConcurrentActions {
	ca := new(ConcurrentActions)
	ca.ProceedWhenNoUpdate = true
	ret := make([]*ActionSpec, 1)
	ret[0] = new(ActionSpec)
	ret[0].Tag = "get"
	ret[0].Method = "GET"
	ret[0].Params = make(map[string][]string, 2)
	ret[0].Params["user"] = []string{"{{.username}}"}
	ret[0].Params["info"] = []string{"name"}
	ret[0].RespTemp = "{{.username}}"
	ca.Actions = ret
	return ca
}

func TestForkWorkers(t *testing.T) {
	N := 200
	users := make([]string, N)
	for i := 0; i < N; i++ {
		users[i] = fmt.Sprintf("user%v", i)
	}

	StartWorkers(30)
	defer StopAllWorkers()

	taskSpec := new(TaskSpec)
	actions := make([]*ConcurrentActions, 0, 3)

	a := genConcurrentAddUserOps(users)
	actions = append(actions, a)

	a = genListUserOp()
	actions = append(actions, a)

	a = genReadUserinfoOps()
	actions = append(actions, a)

	taskSpec.ConcurrentActions = actions

	rr := newUserInfoDb()
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
