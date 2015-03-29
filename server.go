package caspercloud

import (
	"encoding/json"
	"fmt"
	"github.com/BigTong/gocounter"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	kInternalErrorResut = "server get internal result"
)

type CasperServer struct {
	cmdData *ServerData
	ct      *gocounter.Counter
	Host    string
}

func NewCasperServer(host string) *CasperServer {
	return &CasperServer{
		cmdData: NewServerData(),
		ct:      gocounter.NewCounter(),
		Host:    host,
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

func (self *CasperServer) stringify(m map[string]interface{}) string {
	output, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(output)
}

func (self *CasperServer) Process(params url.Values) *Output {
	log.Println(params.Encode())
	id := params.Get("id")
	if len(id) == 0 {
		tmpl := params.Get("tmpl")
		proxyServer := ""
		if params.Get("proxy") == "true" {
			proxyServer := self.getProxy()
			log.Println("use proxy:", proxyServer)
		}
		c := self.cmdData.CreateCommand(tmpl, proxyServer)
		if c == nil {
			return &Output{Status: FAIL}
		}

		params.Add("id", c.GetId())
		return self.setArgs(c, params)
	}

	log.Println("get id", id)
	c := self.cmdData.GetCommand(id)
	if c == nil {
		return &Output{Status: FAIL}
	}

	if c.Finished() {
		self.cmdData.Delete(id)
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
