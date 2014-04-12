package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

func init() {
	RegisterPlugin(&HttpResponseReaderFactory{})
}

type HttpResponseReaderFactory struct {
}

func (self *HttpResponseReaderFactory) String() string {
	return "http"
}

// This plugin does not accept rest in the chain.
func (self *HttpResponseReaderFactory) NewPlugin(params map[string]string, rest ResponseReader) (rr ResponseReader, err error) {
	if rest != nil {
		err = fmt.Errorf("http plugin should never be put as the last plugin")
		return
	}
	rr = &HttpResponseReader{}
	return
}

type HttpResponseReader struct {
	closer
}

func (self *HttpResponseReader) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	var httpResp *http.Response
	r, err := req.ToHttpRequest()
	if err != nil {
		return
	}
	client := &http.Client{}
	httpResp, err = client.Do(r)
	resp = new(Response)
	resp.Status = httpResp.StatusCode
	// resp.Body = httpResp.Body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return
}
