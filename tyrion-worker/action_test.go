package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/mock"
)

type responseReaderMock struct {
	mock.Mock
}

func (self *responseReaderMock) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	args := self.Called(req, env)
	return args.Get(0).(*Response), args.Get(1).(*Env), args.Error(2)
}

func (self *responseReaderMock) Close() error {
	return nil
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

func stringMapEq(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for ak, av := range a {
		if bv, ok := b[ak]; ok {
			if bv != av {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func TestUpdateEnv(t *testing.T) {
}

func TestPerformActionNoMatch(t *testing.T) {
	var as ActionSpec
	as.URLTemplate = "http://localhost:8080/{{.user}}"
	as.Method = "GET"
	as.RespTemps = []string{"(?P<firstName>([a-zA-Z]+)) (?P<lastName>([a-zA-Z]+)): (?P<tel>[a-z]+)"}
	as.MustMatch = true
	as.Headers = make(map[string][]string, 10)
	as.Headers["Content-Type"] = []string{"text/text"}
	as.URLQuery = make(map[string][]string, 10)
	as.URLQuery["name"] = []string{"{{.user}}"}
	as.Content = &HttpRequestContent{}
	as.Content.RawContent = "Username: {{.user}}"
	as.Tag = "sometag"
	response := "Nan Deng: 666999333 \nAlan Turing: 9996664444"
	expUpdates := make([]map[string]string, 2)
	expUpdates[0] = make(map[string]string)
	expUpdates[0]["firstName"] = "Nan"
	expUpdates[0]["lastName"] = "Deng"
	expUpdates[0]["tel"] = "666999333"
	expUpdates[1] = make(map[string]string)
	expUpdates[1]["firstName"] = "Alan"
	expUpdates[1]["lastName"] = "Turing"
	expUpdates[1]["tel"] = "9996664444"
	var env Env
	env.NameValuePairs = make(map[string]string, 1)
	env.NameValuePairs["user"] = "monnand"

	expurl := "http://localhost:8080/monnand"
	method := "GET"
	content := &HttpRequestContent{}
	content.RawContent = "Username: monnand"

	v := url.Values{}
	v.Set("name", "monnand")

	h := http.Header{}
	h.Set("Content-Type", "text/text")

	rr := new(responseReaderMock)
	req := &Request{
		Tag:      as.Tag,
		URL:      expurl,
		Method:   method,
		Content:  content,
		URLQuery: v,
		Headers:  h,
	}
	resp := &Response{
		Status: 200,
		Body:   ioutil.NopCloser(bytes.NewBufferString(response)),
	}
	rr.On("ReadResponse", req, &env).Return(resp, &Env{}, nil)
	action, err := as.GetAction(rr)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = action.Perform(&env)
	if err == nil {
		t.Error("Should be an error")
		return
	}
	rr.AssertExpectations(t)
}

func TestPerformAction(t *testing.T) {
	var as ActionSpec
	as.URLTemplate = "http://localhost:8080/{{.user}}"
	as.Method = "GET"
	as.RespTemps = []string{"(?P<firstName>([a-zA-Z]+)) (?P<lastName>([a-zA-Z]+)): (?P<tel>[0-9]+)"}
	as.Headers = make(map[string][]string, 10)
	as.Headers["Content-Type"] = []string{"text/text"}
	as.URLQuery = make(map[string][]string, 10)
	as.URLQuery["name"] = []string{"{{.user}}"}
	as.Content = &HttpRequestContent{}
	as.Content.RawContent = "Username: {{.user}}"
	as.Tag = "sometag"
	response := "Nan Deng: 666999333 \nAlan Turing: 9996664444"
	expUpdates := make([]map[string]string, 2)
	expUpdates[0] = make(map[string]string)
	expUpdates[0]["firstName"] = "Nan"
	expUpdates[0]["lastName"] = "Deng"
	expUpdates[0]["tel"] = "666999333"
	expUpdates[1] = make(map[string]string)
	expUpdates[1]["firstName"] = "Alan"
	expUpdates[1]["lastName"] = "Turing"
	expUpdates[1]["tel"] = "9996664444"
	var env Env
	env.NameValuePairs = make(map[string]string, 1)
	env.NameValuePairs["user"] = "monnand"

	expurl := "http://localhost:8080/monnand"
	method := "GET"
	content := &HttpRequestContent{}
	content.RawContent = "Username: monnand"

	v := url.Values{}
	v.Set("name", "monnand")

	h := http.Header{}
	h.Set("Content-Type", "text/text")

	rr := new(responseReaderMock)
	req := &Request{
		Tag:      as.Tag,
		URL:      expurl,
		Method:   method,
		Content:  content,
		URLQuery: v,
		Headers:  h,
	}
	resp := &Response{
		Status: 200,
		Body:   ioutil.NopCloser(bytes.NewBufferString(response)),
	}
	rr.On("ReadResponse", req, &env).Return(resp, &Env{}, nil)
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
	rr.AssertExpectations(t)
	if len(updates) != 2 {
		t.Errorf("Only got %v updates, instead of 2", len(updates))
		return
	}
	if !stringMapEq(expUpdates[0], updates[0].NameValuePairs) {
		t.Errorf("updates: %+v, %+v\n", updates[0].NameValuePairs, updates[1].NameValuePairs)
	}
	if !stringMapEq(expUpdates[1], updates[1].NameValuePairs) {
		t.Errorf("updates: %+v, %+v\n", updates[0].NameValuePairs, updates[1].NameValuePairs)
	}
}

func TestPerformActionWithForks(t *testing.T) {
	var as ActionSpec
	as.URLTemplate = "http://localhost:8080/{{.user}}"
	as.Method = "GET"
	as.RespTemps = []string{"(?P<firstName>([a-zA-Z]+)) (?P<lastName>([a-zA-Z]+)): (?P<tel>[0-9]+)"}
	as.Headers = make(map[string][]string, 10)
	as.Headers["Content-Type"] = []string{"text/text"}
	as.URLQuery = make(map[string][]string, 10)
	as.URLQuery["name"] = []string{"{{.user}}"}
	as.MaxNrForks = 1
	as.Tag = "sometag"
	as.Content = &HttpRequestContent{}
	as.Content.RawContent = "Username: {{.user}}"
	response := "Nan Deng: 666999333 \nAlan Turing: 9996664444"
	expUpdates := make([]map[string]string, 2)
	expUpdates[0] = make(map[string]string)
	expUpdates[0]["firstName"] = "Nan"
	expUpdates[0]["lastName"] = "Deng"
	expUpdates[0]["tel"] = "666999333"
	expUpdates[1] = make(map[string]string)
	expUpdates[1]["firstName"] = "Alan"
	expUpdates[1]["lastName"] = "Turing"
	expUpdates[1]["tel"] = "9996664444"
	var env Env
	env.NameValuePairs = make(map[string]string, 1)
	env.NameValuePairs["user"] = "monnand"

	expurl := "http://localhost:8080/monnand"
	method := "GET"
	content := &HttpRequestContent{}
	content.RawContent = "Username: monnand"

	v := url.Values{}
	v.Set("name", "monnand")

	h := http.Header{}
	h.Set("Content-Type", "text/text")
	rr := new(responseReaderMock)
	req := &Request{
		Tag:      as.Tag,
		URL:      expurl,
		Method:   method,
		Content:  content,
		URLQuery: v,
		Headers:  h,
	}
	resp := &Response{
		Status: 200,
		Body:   ioutil.NopCloser(bytes.NewBufferString(response)),
	}
	rr.On("ReadResponse", req, &env).Return(resp, &Env{}, nil)
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
	rr.AssertExpectations(t)
	if len(updates) != 1 {
		t.Errorf("Only got %v updates, instead of 1", len(updates))
		return
	}
	if !stringMapEq(expUpdates[0], updates[0].NameValuePairs) ||
		!stringMapEq(expUpdates[0], updates[0].NameValuePairs) {
		t.Errorf("updates: %+v, %+v\n", updates[0].NameValuePairs, updates[1].NameValuePairs)
	}
}
