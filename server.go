package caspercloud

import (
	"encoding/json"
	"fmt"
	"github.com/BigTong/gocounter"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

const (
	kInternalErrorResut = "server get internal result"
)

type CasperServer struct {
	cmdCache   *CommandCache
	ct         *gocounter.Counter
	cmdFactory CommandFactory
}

func NewCasperServer(cf CommandFactory) *CasperServer {
	return &CasperServer{
		cmdCache:   NewCommandCache(),
		ct:         gocounter.NewCounter(),
		cmdFactory: cf,
	}
}

func (self *CasperServer) setArgs(cmd Command, params url.Values) *Output {
	args := self.getArgs(params)
	log.Println("setArgs:", args)
	cmd.SetInputArgs(args)

	if message := cmd.GetMessage(); message != nil {
		return message
	}
	return nil
}

func (self *CasperServer) getArgs(params url.Values) map[string]string {
	args := make(map[string]string)
	for k, v := range params {
		args[k] = v[0]
	}
	return args
}

func (self *CasperServer) Process(params url.Values) *Output {
	log.Println(params.Encode())
	id := params.Get("id")
	if len(id) == 0 {
		c := self.cmdFactory.CreateCommand(params)
		if c == nil {
			return &Output{Status: FAIL}
		}
		self.cmdCache.SetCommand(c)
		params.Set("id", c.GetId())
		return self.setArgs(c, params)
	}

	log.Println("get id", id)
	c := self.cmdCache.GetCommand(id)
	if c == nil {
		return &Output{Status: FAIL}
	}

	if c.Finished() {
		self.cmdCache.Delete(id)
		return &Output{Status: FINISH_ALL}
	}

	log.Println("get cmd", id)
	return self.setArgs(c, params)
}

func (self *CasperServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR: http submit", r)
			debug.PrintStack()
		}
	}()
	self.ct.Incr("request", 1)
	params := req.URL.Query()
	ret := self.Process(params)
	output, _ := json.Marshal(ret)
	fmt.Fprint(w, string(output))
	return

}
