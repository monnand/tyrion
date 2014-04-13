package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"
)

type ActionSpec struct {
	Debug       bool                `json:"debug,omitempty"`
	Tag         string              `json:"tag"`
	URLTemplate string              `json:"url"`
	Method      string              `json:"method"`
	URLQuery    map[string][]string `json:"urlquery,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
	Content     *HttpRequestContent `json:"content,omitempty"`
	ExpStatuses []int               `json:"expected-statuses,omitempty"`
	RespTemps   []string            `json:"response-templates,omitempty"`
	MustMatch   bool                `json:"must-match,omitempty"`
	MaxNrForks  int                 `json:"max-nr-forks,omitempty"`
}

func randomString() string {
	var d [8]byte
	io.ReadFull(rand.Reader, d[:])
	return fmt.Sprintf("%x-%v", time.Now().Unix(), base64.URLEncoding.EncodeToString(d[:]))
}

func (self *ActionSpec) GetAction(rr ResponseReader) (a *Action, err error) {
	ret := new(Action)
	tmplName := randomString()
	ret.URLTemplate, err = template.New(tmplName).Parse(self.URLTemplate)
	if err != nil {
		err = fmt.Errorf("%v is not a valid template: %v", self.URLTemplate, err)
		return
	}
	ret.Method = strings.ToUpper(self.Method)
	if ret.Method != "GET" &&
		ret.Method != "POST" &&
		ret.Method != "PUT" &&
		ret.Method != "HEAD" &&
		ret.Method != "DELETE" &&
		ret.Method != "TRACE" &&
		ret.Method != "CONNECT" {
		err = fmt.Errorf("Unknown method: %v", ret.Method)
		return
	}
	if len(self.RespTemps) > 0 {
		for _, tmpl := range self.RespTemps {
			var t *template.Template
			if len(tmpl) == 0 {
				continue
			}
			t, err = template.New(randomString()).Parse(tmpl)
			if err != nil {
				err = fmt.Errorf("%v is not valid template: %v", t, err)
				return
			}
			ret.RespTemps = append(ret.RespTemps, t)
		}
	}
	if len(self.URLQuery) > 0 {
		var paramjs []byte
		paramjs, err = json.Marshal(self.URLQuery)
		if err != nil {
			err = fmt.Errorf("%+v is cannot be encoded into json: %v", self.URLQuery, err)
			return
		}
		ret.URLQuery, err = template.New(randomString()).Parse(string(paramjs))
		if err != nil {
			err = fmt.Errorf("%v is not a valid template: %v", string(paramjs), err)
			return
		}
	}
	if len(self.Headers) > 0 {
		var paramjs []byte
		paramjs, err = json.Marshal(self.Headers)
		if err != nil {
			err = fmt.Errorf("%+v is cannot be encoded into json: %v", self.Headers, err)
			return
		}
		ret.Headers, err = template.New(randomString()).Parse(string(paramjs))
		if err != nil {
			err = fmt.Errorf("%v is not a valid template: %v", string(paramjs), err)
			return
		}
	}
	if self.Content != nil {
		ret.Content, err = self.Content.ToTemplate()
		if err != nil {
			return
		}
	}
	if len(self.Tag) > 0 {
		ret.Tag, err = template.New(randomString()).Parse(self.Tag)
		if err != nil {
			err = fmt.Errorf("%v is not a valid template: %v", self.Tag, err)
			return
		}
	} else {
		err = fmt.Errorf("Action needs a tag to identify itself")
		return
	}
	ret.ExpStatuses = self.ExpStatuses
	ret.rr = rr
	ret.MaxNrForks = self.MaxNrForks
	/*
		if self.Debug == "true" || self.Debug == "True" {
			ret.Debug = true
		}
	*/

	ret.Debug = self.Debug
	ret.MustMatch = self.MustMatch
	a = ret
	return
}

/*
type ActionSeqSpec struct {
	MaxReqPerSec float64       `json:"max-req-per-sec"`
	MaxNrReq     int64         `json:"max-nr-req"`
	Actions      []*ActionSpec `json:"urls"`
}

func ParseActionListFromBytes(data []byte) (l *ActionSeqSpec, err error) {
	l = new(ActionSeqSpec)
	err = json.Unmarshal(data, l)
	return
}

func ParseActionListFromReader(reader io.Reader) (l *ActionSeqSpec, err error) {
	l = new(ActionSeqSpec)
	decoder := json.NewDecoder(reader)
	err = decoder.Decode(l)
	return
}
*/
