package caspercloud

import (
	"crawler/common/counter"
	"encoding/json"
	"fmt"
	"github.com/pmylund/go-cache"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	kInternalErrorResut = "server get internal result"
	kWorkTypeNoInteract = "no_interact"
	kWorkTypeInputArgs  = "input_args"
)
const (
	kJobStatus   = "job_status"
	kJobOndoing  = "ondoing"
	kJobFinished = "finished"
)

type CasperServer struct {
	data *cache.Cache
	ct   *counter.Counter
}

func NewCasperServer() *CasperServer {
	return &CasperServer{
		data: cache.New(10*time.Minute, 5*time.Minute),
		ct:   counter.NewCounter(),
	}
}

func (self *CasperServer) executeCmd(cmd Command, req *http.Request) string {
	go cmd.Run()
	if message := cmd.GetMessage(); message != nil {
		if data, err := json.Marshal(&message); err == nil {
			return string(data)
		}
	}
	return kInternalErrorResut
}

func (self *CasperServer) setArgs(cmd Command, req *http.Request) string {
	args := self.getArgs(req)
	cmd.SetInputArgs(args)

	if message := cmd.GetMessage(); message != nil {
		if data, err := json.Marshal(&message); err == nil {
			return string(data)
		}
	}
	return kInternalErrorResut
}

func (self *CasperServer) getArgs(req *http.Request) map[string]string {
	params := req.URL.Query()
	args := make(map[string]string)
	for k, _ := range params {
		if strings.HasPrefix(k, "_") {
			key := strings.TrimPrefix(k, "_")
			args[key] = params.Get(k)
		}
	}
	return args
}

func (self *CasperServer) getRandId(req *http.Request) string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (self *CasperServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR: http submit", r)
			debug.PrintStack()
		}
	}()
	params := req.URL.Query()
	id := params.Get("id")
	if len(id) == 0 {
		id = self.getRandId(req)
		tmpl := params.Get("tmpl")
		proxyServer := params.Get("proxy")
		cmd := NewCasperCmd(id, tmpl, proxyServer)
		args := self.getArgs(req)
		cmd.SetInitialArgs(args)
		self.data.Set(id, *cmd, 0)
		fmt.Fprint(w, self.executeCmd(cmd, req))
		return
	}

	log.Println("get id", id)
	cmd, ok := self.data.Get(id)
	if ok {
		if c, ok := cmd.(CasperCmd); ok {
			log.Println(" get cmd")
			fmt.Fprint(w, self.setArgs(&c, req))
			return
		}
		log.Println("not get cmd")
		return
	}
	fmt.Fprint(w, "your input is time out")
	return

}
