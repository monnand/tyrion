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
	convertError := true
	if str, ok := params["convert-error"]; ok {
		if str == "false" {
			convertError = false
		}
	}
	rr = &HttpResponseReader{convertError: convertError}
	return
}

type HttpResponseReader struct {
	closer
	convertError bool
}

func (self *HttpResponseReader) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	var httpResp *http.Response
	r, err := req.ToHttpRequest()
	if err != nil {
		return
	}
	defer r.Body.Close()
	/*
		d, _ := ioutil.ReadAll(r.Body)
		fmt.Printf("\n******\n%v\n**********\n", string(d))
		pretty.Printf("%# v\n", r.Header)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(d))
	*/
	client := &http.Client{}
	httpResp, err = client.Do(r)
	if err != nil {
		if self.convertError {
			resp = new(Response)
			resp.Status = 500
			resp.Body = ioutil.NopCloser(&bytes.Buffer{})
			err = nil
			return
		}
		return
	}
	resp = new(Response)
	resp.Status = httpResp.StatusCode
	resp.Body = httpResp.Body
	/*
		defer httpResp.Body.Close()
		body, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return
		}
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	*/

	return
}
