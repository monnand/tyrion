package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

type mockPlugin struct {
	Name string
	rest ResponseReader
	closer
}

func (self *mockPlugin) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	data := []byte(self.Name)
	if self.rest != nil {
		var r *Response
		r, _, err = self.rest.ReadResponse(req, env)
		if err != nil {
			return
		}
		var d []byte
		d, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}
		data = append(data, d...)
	}
	resp = new(Response)
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(data))
	return
}

type mockPluginFactory struct {
}

func (self *mockPluginFactory) String() string {
	return "mock"
}

func (self *mockPluginFactory) NewPlugin(params map[string]string, rest ResponseReader) (rr ResponseReader, err error) {
	if name, ok := params["name"]; ok {
		ret := new(mockPlugin)
		ret.Name = name + ","
		ret.rest = rest
		rr = ret
	} else {
		err = fmt.Errorf("Wrong parameter list: %+v", params)
	}
	return
}

func TestPluginManager(t *testing.T) {
	factory := &mockPluginFactory{}
	RegisterPlugin(factory)
	N := 10
	var buf bytes.Buffer
	specs := make([]*PluginSpec, 0, N)

	for i := 0; i < N; i++ {
		name := fmt.Sprintf("plugin-%v", i)
		spec := &PluginSpec{
			Name: factory.String(),
			Params: map[string]string{
				"name": name,
			},
		}
		specs = append(specs, spec)
		fmt.Fprintf(&buf, "%v,", name)
	}

	rr, err := NewPluginChain(specs)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	resp, _, err := rr.ReadResponse(nil, nil)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	if string(d) != buf.String() {
		t.Errorf("Returned: %v", string(d))
	}
}
