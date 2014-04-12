package main

import (
	"fmt"
	"testing"
)

func genHttpReqContent(nrPostFormFields, nrMultiPartFields int) *HttpRequestContent {
	ret := new(HttpRequestContent)
	ret.Form = make(map[string][]string, nrPostFormFields)

	for i := 0; i < nrPostFormFields; i++ {
		key := fmt.Sprintf("%v-key")
		value := fmt.Sprintf("%v-v")
		ret.Form[key] = []string{value}
	}

	f := new(MultiPartFileSpec)
	f.Field = "filename"
	f.Filename = "hello.txt"
	f.Content = "fjasdklflasdj"

	mf := new(MultiPartContentSpec)
	mf.Form = make(map[string][]string, nrMultiPartFields)

	for i := 0; i < nrPostFormFields; i++ {
		key := fmt.Sprintf("%v-key")
		value := fmt.Sprintf("%v-v")
		mf.Form[key] = []string{value}
	}
	mf.Files = append(mf.Files, f)
	ret.MultiPart = mf

	return ret
}

func TestHttpContentEq(t *testing.T) {
	a := genHttpReqContent(10, 10)
	b := genHttpReqContent(10, 10)

	if !a.Eq(b) {
		t.Errorf("%+v != %+v", a, b)
	}
	a.RawContent = "hello-a"
	if a.Eq(b) {
		t.Errorf("%+v == %+v", a, b)
	}
}

func TestHttpContentTemplateTransformWithNoVar(t *testing.T) {
	a := genHttpReqContent(10, 10)
	tmpl, err := a.ToTemplate()
	if err != nil {
		t.Fatalf("Template error: %v", err)
	}
	b, err := NewContent(tmpl, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !a.Eq(b) {
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestHttpContentTemplateTransformWithVars(t *testing.T) {
	a := genHttpReqContent(10, 10)
	a.Form["some-key-{{.name}}"] = []string{"{{.tel}}"}
	tmpl, err := a.ToTemplate()
	if err != nil {
		t.Fatalf("Template error: %v", err)
	}
	env := EmptyEnv()
	env.NameValuePairs["name"] = "Someone"
	env.NameValuePairs["tel"] = "12345"
	b, err := NewContent(tmpl, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	delete(a.Form, "some-key-{{.name}}")
	a.Form["some-key-Someone"] = []string{"12345"}
	if !a.Eq(b) {
		t.Errorf("%+v != %+v", a, b)
	}
}
