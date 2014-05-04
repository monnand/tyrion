package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

func init() {
	RegisterPlugin(&TimerResponseReaderFactory{})
}

type TimerResponseReaderFactory struct {
}

func (self *TimerResponseReaderFactory) String() string {
	return "timer"
}

func (self *TimerResponseReaderFactory) NewPlugin(params map[string]string, rest ResponseReader) (rr ResponseReader, err error) {
	ret := new(TimerResponseReader)
	if filename, ok := params["log"]; ok {
		ret.out, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return
		}
	} else {
		err = fmt.Errorf("timer needs a filename to take logs")
		return
	}
	if tagp, ok := params["tag"]; ok {
		ret.tagPattern, err = regexp.Compile(tagp)
		if err != nil {
			return
		}
	}
	ret.rest = rest
	rr = ret
	return
}

type TimerResponseReader struct {
	rest       ResponseReader
	out        io.WriteCloser
	tagPattern *regexp.Regexp
}

func (self *TimerResponseReader) Close() error {
	if self.out != nil {
		self.out.Close()
	}
	if self.rest != nil {
		return self.rest.Close()
	}
	return nil
}

func (self *TimerResponseReader) ReadResponse(req *Request, env *Env) (resp *Response, updates *Env, err error) {
	if self.tagPattern != nil {
		m := self.tagPattern.FindString(req.Tag)
		fmt.Printf("Matched pattern: %v\n", m)
		if len(m) == 0 {
			resp, updates, err = self.rest.ReadResponse(req, env)
			return
		}
	}
	var delta time.Duration
	start := time.Now()
	if self.rest != nil {
		resp, updates, err = self.rest.ReadResponse(req, env)
		delta = time.Now().Sub(start)
		if err != nil {
			return
		}
		if self.out == nil {
			return
		}
		fmt.Fprintf(self.out, "[%v]\t%v\t%v\t%v\tStatus%v\n", start, req.Tag, delta.Nanoseconds(), delta, resp.Status)
	}
	return
}
