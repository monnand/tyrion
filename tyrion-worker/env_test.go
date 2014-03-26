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

func TestEnvUpdate(t *testing.T) {
	env := new(Env)
	env.NameValuePairs = make(map[string]string, 3)
	env.NameValuePairs["k1"] = "v1"
	env.NameValuePairs["k2"] = "v2"
	env.NameValuePairs["k3"] = "v3"

	n1 := new(Env)
	n1.NameValuePairs = make(map[string]string, 3)
	n1.NameValuePairs["key1"] = "value1"
	n1.NameValuePairs["key2"] = "value2"

	n2 := new(Env)
	n2.NameValuePairs = make(map[string]string, 3)
	n2.NameValuePairs["key1"] = "value1"
	n2.NameValuePairs["key2"] = "value2"

	n3 := new(Env)
	n3.NameValuePairs = make(map[string]string, 3)
	n3.NameValuePairs["key_1"] = "value_1"
	n3.NameValuePairs["key_2"] = "value_2"

	forks := env.Fork(n1, n2, n3)
	if len(forks) != 2 {
		t.Errorf("Got %v forks, not 2.", len(forks))
	}
}
