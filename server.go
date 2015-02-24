package caspercloud

import (
	"encoding/json"
	"fmt"
	"github.com/BigTong/gocounter"
	"github.com/pmylund/go-cache"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
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

func (self *CasperServer) setArgs(cmd Command, params url.Values) string {
	args := self.getArgs(params)
	log.Println("setArgs:", args)
	cmd.SetInputArgs(args)

	if message := cmd.GetMessage(); message != nil {
		if data, err := json.Marshal(&message); err == nil {
			return string(data)
		}
	}
	return kInternalErrorResut
}

func (self *CasperServer) getArgs(params url.Values) map[string]string {
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

func (self *CasperServer) getProxy() string {
	c := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				deadline := time.Now().Add(5 * time.Second)
				c, err := net.DialTimeout(network, addr, 5*time.Second)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 5 * time.Second,
			DisableCompression:    false,
		},
	}
	resp, err := c.Get("http://54.223.171.0:7183/select")
	if err != nil {
		log.Print("select proxy fail:", err)
		return ""
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(b)
}

func (self *CasperServer) ServeWebSocket(ws *websocket.Conn) {
	for {
		var reply string

		if err := websocket.Message.Receive(ws, &reply); err != nil {
			log.Println("Can't receive")
			break
		}

		params, err := url.ParseQuery(reply)
		if err != nil {
			break
		}

		msg := self.Process(params)
		if err := websocket.Message.Send(ws, msg); err != nil {
			log.Println("Can't send")
			break
		}
	}
}

func (self *CasperServer) Process(params url.Values) string {
	id := params.Get("id")
	if len(id) == 0 {
		tmpl := params.Get("tmpl")
		proxyServer := self.getProxy()
		log.Println("use proxy:", proxyServer)
		c := self.cmdData.GetNewCommand(tmpl, proxyServer)
		if c == nil {
			log.Println("server is to busy", tmpl)
			return "server is too busy"
		}

		id = self.getRandId(nil)
		self.clientData.Set(id, c.GetId(), kKeepMinutes*time.Minute)
		log.Println("cmd, ", c.GetId(), self.cmdData.index)

		params.Add("_id", id)
		params.Add("_start", "yes")
		return self.setArgs(c, params)
	}

	log.Println("get id", id)
	value, ok := self.clientData.Get(id)
	if ok {
		if id, ok := value.(string); ok {
			c := self.cmdData.GetCommand(id)
			if c == nil {
				log.Println("your input is time out", id)
				return "your input is time out"
			}

			if c.Finished() {
				self.clientData.Delete(id)
				log.Println("your input is finished", id)
				return "your input is finished"
			}
			log.Println("get cmd", id)
			return self.setArgs(c, params)
		}
		log.Println("not get cmd")
	}
	return "your input is time out"
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
	fmt.Fprint(w, ret)
	return

}
