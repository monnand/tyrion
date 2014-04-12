package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func testHttpResponseReader(t *testing.T, method, content string, params url.Values, headers http.Header) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		if r.Method != method {
			t.Errorf("Method should be %v; Received %v", method, r.Method)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		c := string(b)
		if c != content {
			t.Errorf("content should be %v; Received %v", content, c)
		}

		if len(params) > 0 {
			q := r.URL.Query()
			for k, _ := range params {
				if params.Get(k) != q.Get(k) {
					t.Errorf("Parameter %v should be %v; received %v. content: %v", k, params.Get(k), r.PostForm.Get(k), c)
				}
			}
		}
		if len(headers) > 0 {
			for k, _ := range headers {
				if headers.Get(k) != r.Header.Get(k) {
					t.Errorf("Header %v should be %v; received %v", k, headers.Get(k), r.Header.Get(k))
				}
			}
		}
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	rr := &HttpResponseReader{}
	req := &Request{
		Tag:     "test",
		URL:     ts.URL,
		Method:  method,
		Content: &HttpRequestContent{RawContent: content},
		Params:  params,
		Headers: headers,
	}
	resp, u, err := rr.ReadResponse(req, nil)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if resp == nil {
		t.Errorf("Http response is nil")
	} else if resp.Body == nil {
		t.Errorf("Http response body is nil")
	} else {
		defer resp.Body.Close()
	}
	if !u.IsEmpty() {
		t.Errorf("Updated env should be empty: %v", u)
	}
	if resp.Status != 200 {
		t.Errorf("Status: %v", resp.Status)
	}
}

func TestHttpResponseReader(t *testing.T) {
	testHttpResponseReader(t, "GET", "", nil, nil)
	testHttpResponseReader(t, "POST", "", nil, nil)
	testHttpResponseReader(t, "GET", "content", nil, nil)
	params := url.Values{}
	params.Set("hello", "world")
	params.Set("hello2", "world2")
	testHttpResponseReader(t, "POST", "", params, nil)
	headers := http.Header{}
	headers.Set("hello", "world")
	headers.Set("hello2", "world2")
	testHttpResponseReader(t, "GET", "", nil, headers)
	testHttpResponseReader(t, "GET", "something", nil, headers)
	testHttpResponseReader(t, "POST", "hello", nil, headers)
}
