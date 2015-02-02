package caspercloud

import (
	"encoding/json"
	"fmt"
	"github.com/BigTong/gocounter"
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
)

type CasperServer struct {
	clientData *cache.Cache
	cmdData    *ServerData
	ct         *gocounter.Counter
}

func NewCasperServer() *CasperServer {
	return &CasperServer{
		clientData: cache.New(10*time.Minute, 10*time.Minute),
		cmdData:    NewServerData(),
		ct:         gocounter.NewCounter(),
	}
}

func (self *CasperServer) setArgs(cmd Command, req *http.Request) string {
	args := self.getArgs(req)
	log.Println("setArgs:", args)
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
	self.ct.Incr("request", 1)
	params := req.URL.Query()
	id := params.Get("id")
	if len(id) == 0 {
		tmpl := params.Get("tmpl")
		proxyServer := params.Get("proxy")
		c := self.cmdData.GetNewCommand(tmpl, proxyServer)
		if c == nil {
			fmt.Fprint(w, "server is to busy")
			return
		}

		id = self.getRandId(req)
		self.clientData.Set(id, c.GetId(), kKeepMinutes*time.Minute)

		params.Add("_id", id)
		params.Add("_start", "yes")
		req.URL.RawQuery = params.Encode()
		fmt.Fprint(w, self.setArgs(c, req))
		return
	}

	log.Println("get id", id)
	value, ok := self.clientData.Get(id)
	if ok {
		if id, ok := value.(string); ok {
			c := self.cmdData.GetCommand(id)
			if c.Finished() {
				self.clientData.Delete(id)
				fmt.Fprint(w, "your input is time out")
				return
			}
			log.Println("get cmd")
			fmt.Fprint(w, self.setArgs(c, req))
			return
		}
		log.Println("not get cmd")
		return
	}
	fmt.Fprint(w, "your input is time out")
	return

}
