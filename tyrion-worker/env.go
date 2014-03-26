package main

type Env struct {
	NameValuePairs map[string]string
}

func (self *Env) Update(envs ...*Env) {
	for _, env := range envs {
		for k, v := range env.NameValuePairs {
			self.NameValuePairs[k] = v
		}
	}
}
