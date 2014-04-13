package main

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/kr/pretty"
)

type Request struct {
	Tag      string
	URL      string
	Method   string
	Content  *HttpRequestContent
	URLQuery url.Values
	Headers  http.Header
}

func (self *Request) ToHttpRequest() (req *http.Request, err error) {
	ret, err := http.NewRequest(self.Method, self.URL, &bytes.Buffer{})
	if err != nil {
		return
	}
	err = self.Content.DecorateRequest(ret)
	if err != nil {
		return
	}
	pretty.Printf("After dec: %# v\n", ret.Header)
	if ret.Header == nil {
		ret.Header = make(map[string][]string, 10)
	}
	for k, vs := range self.Headers {
		for _, v := range vs {
			ret.Header.Add(k, v)
		}
	}
	pretty.Printf("After header: %# v\n", ret.Header)

	if len(self.URLQuery) > 0 {
		ret.URL.RawQuery = self.URLQuery.Encode()
	}
	req = ret
	return
}

type Response struct {
	Status int
	Body   io.ReadCloser
}

type ResponseReader interface {
	ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error)
	Close() error
}

type closer struct {
}

func (self *closer) Close() error {
	return nil
}
