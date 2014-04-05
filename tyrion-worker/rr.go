package main

import (
	"io"
	"net/http"
	"net/url"
)

type Request struct {
	Tag     string
	URL     string
	Method  string
	Content string
	Params  url.Values
	Headers http.Header
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
