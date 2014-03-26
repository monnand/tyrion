package main

import "testing"

func TestEnvClone(t *testing.T) {
	env := new(Env)
	env.NameValuePairs = make(map[string]string, 3)
	env.NameValuePairs["k1"] = "v1"
	env.NameValuePairs["k2"] = "v2"
	env.NameValuePairs["k3"] = "v3"
	n := env.Clone()
	if !n.Equals(env) {
		t.Errorf("Cloned env should be the same")
	}
	if !env.Equals(n) {
		t.Errorf("Equals() should be commutative")
	}
}
