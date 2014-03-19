package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"testing"
)

type responseReaderMock struct {
	resp   string
	status int
	err    error
}

func newMockResponseReader(resp string, status int, err error) ResponseReader {
	return &responseReaderMock{resp, status, err}
}

func (self *responseReaderMock) ReadResponse(url, method, content string, params url.Values) (status int, body io.ReadCloser, err error) {
	// TODO check the url, method, content and params
	body = ioutil.NopCloser(bytes.NewBufferString(self.resp))
	status = self.status
	err = self.err
	return
}

func envHasValues(env *Env, vals map[string]string) error {
	for k, v := range vals {
		if value, ok := env.NameValuePairs[k]; ok {
			if value != v {
				return fmt.Errorf("%v should be %v, not %v", k, v, value)
			}
		} else {
			return fmt.Errorf("Cannot find %v", k)
		}
	}
	return nil
}

func TestPerformAction(t *testing.T) {
	var as ActionSpec
	as.URLTemplate = "http://localhost:8080/{{.user}}"
	as.Method = "GET"
	as.RespTemp = "(?P<firstName>([a-zA-Z]+)) (?P<lastName>([a-zA-Z]+)): (?P<tel>[0-9]+)"
	response := "Nan Deng: 666999333 Alan Turing: 9996664444"
	rr := newMockResponseReader(response, 200, nil)
	var env Env
	env.NameValuePairs = make(map[string]string, 1)
	env.NameValuePairs["user"] = "monnand"

	action, err := as.GetAction(rr)
	if err != nil {
		t.Error(err)
		return
	}
	updates, err := action.Perform(&env)
	if err != nil {
		t.Error(err)
		return
	}
	if len(updates) != 2 {
		t.Errorf("Only got %v updates, instead of 2", len(updates))
		return
	}
}
