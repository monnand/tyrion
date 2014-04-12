package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
)

type MultiPartFileSpec struct {
	Field    string `json:"field,omitempty"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"file,omitempty"`
}

func (self *MultiPartFileSpec) getFilename() string {
	if len(self.Filename) > 0 && self.Filename[0] == '@' {
		return filepath.Base(self.Filename)
	}
	return self.Filename
}
func (self *MultiPartFileSpec) getContentReader() (r io.ReadCloser, err error) {
	if len(self.Filename) > 0 && self.Filename[0] == '@' {
		r, err = os.Open(self.Filename[1:])
		return
	}
	if len(self.Content) == 0 {
		r = ioutil.NopCloser(&bytes.Buffer{})
		return
	}
	r = ioutil.NopCloser(bytes.NewBufferString(self.Content))
	return
}

func (self *MultiPartFileSpec) WriteFile(writer *multipart.Writer) error {
	if self == nil {
		return nil
	}
	content, err := self.getContentReader()
	if err != nil {
		return fmt.Errorf("cannot write file %v. %v", self.Filename, err)
	}
	defer content.Close()
	fn := self.getFilename()
	if len(fn) == 0 || len(self.Field) == 0 {
		return nil
	}
	part, err := writer.CreateFormFile(self.Field, fn)
	if err != nil {
		return fmt.Errorf("cannot create form for file %v. %v", self.Filename, err)
	}
	_, err = io.Copy(part, content)
	if err != nil {
		return fmt.Errorf("cannot write form for file %v. %v", self.Filename, err)
	}
	return nil
}

type MultiPartContentSpec struct {
	Form  map[string][]string  `json:"form,omitempty"`
	Files []*MultiPartFileSpec `json:"files,omitempty"`
}

// If RawContent is specified, others will be ignored.
// Otherwise, if MultiPart is specified, Form will be ignored.
// Otherwise, use Form or empty string.
type HttpRequestContent struct {
	RawContent string                `josn:"raw-content,omitempty"`
	MultiPart  *MultiPartContentSpec `json:"multipart,omitempty"`
	Form       map[string][]string   `json:"form,omitempty"`
}

func (self *HttpRequestContent) Eq(b *HttpRequestContent) bool {
	return reflect.DeepEqual(self, b)
}

func (self *HttpRequestContent) ToTemplate() (tmpl *template.Template, err error) {
	js, err := json.Marshal(self)
	if err != nil {
		err = fmt.Errorf("%+v is cannot be encoded into json: %v", self, err)
		return
	}
	tmpl, err = template.New(randomString()).Parse(string(js))
	if err != nil {
		err = fmt.Errorf("%v is not a valid template: %v", string(js), err)
		return
	}
	return
}

func (self *HttpRequestContent) DecorateRequest(req *http.Request) error {
	if req == nil {
		return nil
	}
	if req.Header == nil {
		req.Header = make(map[string][]string, 10)
	}
	body := &bytes.Buffer{}
	if self == nil {
		req.Body = ioutil.NopCloser(body)
		return nil
	}
	if len(self.RawContent) > 0 {
		req.Body = ioutil.NopCloser(bytes.NewBufferString(self.RawContent))
		return nil
	}
	if self.MultiPart != nil {
		writer := multipart.NewWriter(body)
		for k, vs := range self.MultiPart.Form {
			for _, v := range vs {
				err := writer.WriteField(k, v)
				if err != nil {
					return fmt.Errorf("Multipart: cannot write field %v with value %v. %v", k, v, err)
				}
			}
		}
		for _, file := range self.MultiPart.Files {
			err := file.WriteFile(writer)
			if err != nil {
				return err
			}
		}
		req.Body = ioutil.NopCloser(body)
		req.Header.Add("Content-Type", writer.FormDataContentType())
	}
	if len(self.Form) > 0 {
		form := url.Values(self.Form)
		body = bytes.NewBufferString(form.Encode())
		req.Body = ioutil.NopCloser(body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return nil
}

func NewContent(tmpl *template.Template, env *Env) (content *HttpRequestContent, err error) {
	var out bytes.Buffer
	if tmpl == nil {
		return
	}
	if env == nil {
		err = tmpl.Execute(&out, nil)
	} else {
		err = tmpl.Execute(&out, env.NameValuePairs)
	}
	if err != nil {
		return
	}
	ret := new(HttpRequestContent)
	err = json.Unmarshal(out.Bytes(), ret)
	if err != nil {
		return
	}
	content = ret
	return
}
