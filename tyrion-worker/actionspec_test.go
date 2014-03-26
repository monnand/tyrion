package main

import (
	"testing"
)

func TestParseActionList(t *testing.T) {
	content := `{
		"max-nr-req":1000,
		"max-req-per-sec":100.0,
		"urls":[{
			"url": "http://localhost:8080/userlist",
			"method": "get",
			"parameters": {
				"hello":["world"],
				"param":["value1","value2"]
			},
		"content":"Something"},
		{"url": "http://localhost:8080/userlist",
		"method":"get"}]
	}`
	l, err := ParseActionListFromBytes([]byte(content))
	if err != nil {
		t.Errorf("Parse error: %v", err)
		return
	}
	if l.MaxNrReq != 1000 {
		t.Errorf("MaxNrReq should be 1000, but got %v", l.MaxNrReq)
	}
	if l.MaxReqPerSec != 100.0 {
		t.Errorf("MaxReqPerSecond should be 100.0, but got %v", l.MaxReqPerSec)
	}
	if len(l.Actions) != 2 {
		t.Errorf("There should be 2 urls, but got %v", len(l.Actions))
	}
	if len(l.Actions[0].Params) != 2 {
		t.Errorf("There should be 2 parameters in the first url, but got %v", len(l.Actions[0].Params))
	}
}
