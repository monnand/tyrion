package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type ActionSpec struct {
	URLTemplate string              `json:"url"`
	Method      string              `json:"method"`
	Params      map[string][]string `json:"parameters"`
	Content     string              `json:"content"`
	ExpStatus   int                 `json:"expected-status"`
	RespTemp    string              `json:"response-template"`
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
	if len(self.RespTemp) > 0 {
		ret.RespTemp, err = regexp.Compile(self.RespTemp)
		if err != nil {
			err = fmt.Errorf("%v is not valid regexp: %v", self.RespTemp, err)
			return
		}
	}
	if len(self.Params) > 0 {
		var paramjs []byte
		paramjs, err = json.Marshal(self.Params)
		if err != nil {
			err = fmt.Errorf("%+v is cannot be encoded into json: %v", self.Params, err)
			return
		}
		ret.Params, err = template.New(randomString()).Parse(string(paramjs))
		if err != nil {
			err = fmt.Errorf("%v is not a valid template: %v", string(paramjs), err)
			return
		}
	}
	if len(self.Content) > 0 {
		ret.Content, err = template.New(randomString()).Parse(self.Content)
		if err != nil {
			err = fmt.Errorf("%v is not a valid template: %v", self.Content, err)
			return
		}
	}
	ret.ExpStatus = self.ExpStatus
	if ret.ExpStatus < 0 {
		ret.ExpStatus = 0
	}
	a = ret
	a.rr = rr
	return
}

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