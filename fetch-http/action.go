package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"regexp"
	"text/template"
)

type Env struct {
	NameValuePairs map[string]string
}

func (self *Env) Update(envs ...*Env) {
	for _, env := range envs {
		for k, v := range env.NameValuePairs {
			self.NameValuePairs[k] = v
		}
	}
}

type ResponseReader interface {
	ReadResponse(url, method, content string, params url.Values) (status int, body io.ReadCloser, err error)
}

type Action struct {
	URLTemplate *template.Template
	Method      string
	Params      url.Values
	Content     string
	ExpStatus   int
	RespTemp    *regexp.Regexp
	rr          ResponseReader
}

func (self *Action) Perform(vars *Env) (updates []*Env, err error) {
	var out bytes.Buffer
	err = self.URLTemplate.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	url := out.String()
	status, body, err := self.rr.ReadResponse(url, self.Method, self.Content, self.Params)
	if err != nil {
		return
	}
	defer body.Close()

	if self.ExpStatus > 0 {
		if self.ExpStatus != status {
			err = fmt.Errorf("Reuqest URL %v, expected status code %v, but received %v", url, self.ExpStatus, status)
		}
	}
	if self.RespTemp == nil {
		return
	}

	d, err := ioutil.ReadAll(body)
	if err != nil {
		err = fmt.Errorf("URL %v: read body error. %v", url, err)
		return
	}
	data := string(d)
	matched := self.RespTemp.FindAllStringSubmatch(data, -1)
	if len(matched) == 0 {
		err = fmt.Errorf("URL %v: cannot find matched patterns in the response", url)
		return
	}
	var_names := self.RespTemp.SubexpNames()
	u := make([]*Env, 0, len(matched))
	for _, m := range matched {
		e := new(Env)

		e.NameValuePairs = make(map[string]string, len(var_names))
		for i, v := range var_names {
			if len(v) == 0 {
				continue
			}
			e.NameValuePairs[v] = m[i]
		}
		if len(e.NameValuePairs) > 0 {
			u = append(u, e)
		}
	}

	updates = u
	return
}
