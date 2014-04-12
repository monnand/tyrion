package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
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
	var r *http.Request
	if len(req.Params) > 0 && len(req.Content) == 0 {
		r, err = http.NewRequest(req.Method, req.URL, bytes.NewBufferString(req.Params.Encode()))
		if len(r.Header) == 0 {
			r.Header = make(map[string][]string, len(req.Headers)+1)
		}
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if len(req.Params) > 0 && len(req.Content) > 0 {
		// Multi-part post
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		var part io.Writer
		part, err = writer.CreateFormFile(randomString(), randomString())
		if err != nil {
			return
		}
		buf := bytes.NewBufferString(req.Content)
		_, err = io.Copy(part, buf)
		if err != nil {
			return
		}
		for k, vs := range req.Params {
			for _, v := range vs {
				err = writer.WriteField(k, v)
				if err != nil {
					return
				}
			}
		}
		err = writer.Close()
		if err != nil {
			return
		}
		r, err = http.NewRequest(req.Method, req.URL, body)
	} else {
		r, err = http.NewRequest(req.Method, req.URL, bytes.NewBufferString(req.Content))
	}
	if err != nil {
		return
	}
	if len(r.Header) == 0 {
		r.Header = make(map[string][]string, len(req.Headers))
	}
	for k, vs := range req.Headers {
		for _, v := range vs {
			r.Header.Set(k, v)
		}
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
