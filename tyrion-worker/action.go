package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"text/template"
)

/*
type ResponseReader interface {
	ReadResponse(tag, url, method, content string, params url.Values, headers http.Header) (status int, body io.ReadCloser, err error)
}
*/

// Any method of Action will never change the Action itself.
// i.e. concurretly running any method of the same Action should be fine.
// and the Action will not be changed after each call.
type Action struct {
	Debug       bool
	URLTemplate *template.Template
	Tag         *template.Template
	Method      string
	Params      *template.Template
	Headers     *template.Template
	Content     *template.Template
	ExpStatuses []int
	MaxNrForks  int
	RespTemps   []*template.Template
	MustMatch   bool
	rr          ResponseReader
}

func (self *Action) getURL(vars *Env) (url string, err error) {
	var out bytes.Buffer
	if self.URLTemplate == nil {
		err = fmt.Errorf("No URL template")
		return
	}
	err = self.URLTemplate.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	url = out.String()
	return
}

func (self *Action) getParams(vars *Env) (params url.Values, err error) {
	var out bytes.Buffer
	if self.Params == nil {
		return
	}
	err = self.Params.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	ret := make(map[string][]string, 10)
	err = json.Unmarshal(out.Bytes(), &ret)
	if err != nil {
		return
	}
	if len(ret) > 0 {
		params = ret
	}
	return
}

func (self *Action) getHeaders(vars *Env) (headers http.Header, err error) {
	var out bytes.Buffer
	if self.Headers == nil {
		return
	}
	err = self.Headers.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	ret := make(map[string][]string, 10)
	err = json.Unmarshal(out.Bytes(), &ret)
	if err != nil {
		return
	}
	if len(ret) > 0 {
		headers = ret
	}
	return
}

func (self *Action) getTag(vars *Env) (tag string, err error) {
	var out bytes.Buffer
	if self.Tag == nil {
		return
	}
	err = self.Tag.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	tag = out.String()
	return
}

func (self *Action) getContent(vars *Env) (content string, err error) {
	var out bytes.Buffer
	if self.Content == nil {
		return
	}
	err = self.Content.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	content = out.String()
	return
}

func (self *Action) getRespPattern(vars *Env, idx int) (resp *regexp.Regexp, err error) {
	if idx < 0 || idx >= len(self.RespTemps) {
		resp = nil
		return
	}
	tmpl := self.RespTemps[idx]
	var out bytes.Buffer
	// FIXME This is dangours! Need to escape first
	err = tmpl.Execute(&out, vars.NameValuePairs)
	if err != nil {
		return
	}
	pattern := out.String()
	resp, err = regexp.Compile(pattern)
	return
}

func (self *Action) Perform(vars *Env) (updates []*Env, err error) {
	if vars == nil {
		vars = EmptyEnv()
	}
	url, err := self.getURL(vars)
	if err != nil {
		err = fmt.Errorf("invalid URL template: %v", err)
		return
	}
	params, err := self.getParams(vars)
	if err != nil {
		err = fmt.Errorf("invalid parameter template: %v", err)
		return
	}
	headers, err := self.getHeaders(vars)
	if err != nil {
		err = fmt.Errorf("invalid header template: %v", err)
		return
	}
	content, err := self.getContent(vars)
	if err != nil {
		err = fmt.Errorf("invalid content template: %v", err)
		return
	}
	tag, err := self.getTag(vars)
	if err != nil {
		err = fmt.Errorf("invalid tag template: %v", err)
		return
	}

	req := &Request{
		Tag:     tag,
		URL:     url,
		Method:  self.Method,
		Content: content,
		Params:  params,
		Headers: headers,
	}

	resp, rupdates, err := self.rr.ReadResponse(req, vars)
	if err != nil {
		return
	}

	var u []*Env
	hasMatched := false
	if resp != nil && resp.Body != nil && len(self.RespTemps) > 0 {
		body := resp.Body
		defer body.Close()
		var d []byte
		d, err = ioutil.ReadAll(body)
		if err != nil {
			err = fmt.Errorf("URL %v: read body error. %v", url, err)
			return
		}
		data := string(d)
		if self.Debug {
			fmt.Printf("\n[DEBUG MESSAGE BEGIN]\n\tReq=%+v\n\tResp: %v\n[DEBUG MESSAGE END]\n", req, data)
		}
		var respPattern *regexp.Regexp
		for i := 0; i < len(self.RespTemps); i++ {
			respPattern, err = self.getRespPattern(vars, i)
			if err != nil {
				err = fmt.Errorf("Tag=%v URL=%v %v", self.Tag, url, err)
				return
			}
			if respPattern == nil {
				continue
			}
			matched := respPattern.FindAllStringSubmatch(data, -1)
			if len(matched) == 0 {
				continue
			}
			if self.Debug {
				fmt.Printf("\n[DEBUG MESSAGE BEGIN]\n\tReq=%+v\n\tMatched: %+v\n[DEBUG MESSAGE END]\n", req, matched)
			}
			hasMatched = true
			if self.MaxNrForks > 0 {
				if len(matched) > self.MaxNrForks {
					permedIdx := rand.Perm(len(matched))
					m := make([][]string, self.MaxNrForks)
					for i, idx := range permedIdx[:self.MaxNrForks] {
						m[i] = matched[idx]
					}
					matched = m
				}
			}
			var_names := respPattern.SubexpNames()
			u = make([]*Env, 0, len(matched))
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
					e.Update(rupdates)
					u = append(u, e)
				}
			}
			// Only match at most one pattern
			break
		}
	}

	if len(self.RespTemps) > 0 && !hasMatched && self.MustMatch {
		err = fmt.Errorf("URL %v: cannot find matched patterns in the response", url)
		return
	}

	if len(self.ExpStatuses) > 0 && resp != nil {
		found := false
		for _, s := range self.ExpStatuses {
			if resp.Status == s {
				found = true
			}
		}
		if !found {
			err = fmt.Errorf("Reuqest URL %v, expected status codes are %+v, but received %v", url, self.ExpStatuses, resp.Status)
		}
	}

	if len(self.RespTemps) == 0 {
		return
	}

	if len(u) == 0 {
		if !rupdates.IsEmpty() {
			u = append(u, rupdates)
		}
	}

	updates = u
	return
}
