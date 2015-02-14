package caspercloud

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CasperCmd struct {
	proxyServer string
	id          string
	tmpl        string
	message     chan map[string]interface{}
	input       chan map[string]string
	isKill      bool
	isFinish    bool
	args        map[string]string
	status      int
}

func NewCasperCmd(id, tmpl, proxyServer string) *CasperCmd {
	ret := &CasperCmd{
		proxyServer: proxyServer,
		id:          id,
		tmpl:        tmpl,
		message:     make(chan map[string]interface{}, 1),
		input:       make(chan map[string]string, 1),
		args:        make(map[string]string),
		status:      kCommandStatusIdle,
	}
	go ret.run()
	return ret
}

func (self *CasperCmd) GetId() string {
	return self.id
}

func (self *CasperCmd) GetStatus() int {
	return self.status
}

func (self *CasperCmd) SetInputArgs(input map[string]string) {
	if self.Finished() {
		log.Println("start another casperjs")
		go self.run()
	}
	log.Println("insert input:", input)
	self.input <- input
}

func (self *CasperCmd) GetMessage() map[string]interface{} {
	return <-self.message
}

func (self *CasperCmd) readInputArgs(key string) string {
	args := <-self.input
	log.Println("read args:", args)
	for k, v := range args {
		self.args[k] = v
	}
	if val, ok := self.args[key]; ok {
		return val
	}

	message := make(map[string]interface{})
	message["id"] = self.GetArgsValue("id")
	message["need_args"] = key
	message[kJobStatus] = kJobOndoing
	self.message <- message

	return ""
}

func (self *CasperCmd) GetArgsValue(key string) string {
	if val, ok := self.args[key]; ok {
		log.Println("successfully get args value", val)
		return val
	}
	for {
		val := self.readInputArgs(key)
		if len(val) != 0 {
			log.Println("successfully get args value", val)
			return val
		}
	}
	return ""
}

func (self *CasperCmd) getArgsList(args string) []string {
	segs := strings.Split(args, "/")
	if len(segs) < 2 {
		return nil
	}
	return segs[1:]
}

func (self *CasperCmd) Finished() bool {
	return self.isKill || self.isFinish
}

func (self *CasperCmd) run() {
	self.isFinish = false
	self.isKill = false

	path := "./" + self.tmpl + "/" + self.id
	os.RemoveAll(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalln("can not create", path, err)
	}

	cookieFile, err := os.Create(path + "/cookie.txt")
	defer cookieFile.Close()
	var cmd *exec.Cmd
	if len(self.proxyServer) == 0 {
		cmd = exec.Command("casperjs", self.tmpl+".js", "--web-security=no", "--cookies-file="+path+"/cookie.txt", "--context="+path)
	} else {
		cmd = exec.Command("casperjs", self.tmpl+".js", "--web-security=no", "--cookies-file="+path+"/cookie.txt", "--proxy="+self.proxyServer, "--proxy-type=http", "--context="+path)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Panicln("can not get stdout pipe:", err)
	}
	bufout := bufio.NewReader(stdout)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Panicln("can not get stdin pipe:", err)
	}

	bufin := bufio.NewWriter(stdin)
	if err := cmd.Start(); err != nil {
		log.Panicln("can not start cmd:", err)
	}

	log.Println("begin read line from capser")
	for {
		line, err := bufout.ReadString('\n')
		if err != nil {
			log.Println(err)
			cmd.Process.Wait()
			cmd.Process.Kill()
			break
		}
		log.Println(line)

		if strings.HasPrefix(line, "CMD GET ARGS") {
			for _, v := range self.getArgsList(line) {
				key := strings.TrimRight(v, "\n")
				bufin.WriteString(self.GetArgsValue(key))
				delete(self.args, key)
				bufin.WriteRune('\n')
				bufin.Flush()
			}
			continue
		}

		if strings.HasPrefix(line, "CMD INFO RANDCODE") {
			message := make(map[string]interface{})
			message["id"] = self.GetArgsValue("id")
			result := strings.TrimPrefix(line, "CMD INFO RANDCODE")
			result = strings.Trim(result, " \n")
			message["result"] = result
			message[kJobStatus] = kJobOndoing
			log.Println("send result:", message)
			self.message <- message
			continue
		}

		if strings.HasPrefix(line, "CMD INFO CONTENT") {
			message := make(map[string]interface{})
			message["id"] = self.GetArgsValue("id")
			result := strings.TrimPrefix(line, "CMD INFO CONTENT")
			result = strings.Trim(result, " \n")
			message["result"] = result
			var out map[string]interface{}
			err := json.Unmarshal([]byte(result), &out)
			if err == nil {
				message["json"] = out
			}
			message[kJobStatus] = kJobFinished
			log.Println("send result:", message)
			self.message <- message
			time.Sleep(time.Minute)
			self.status = kCommandStatusIdle
		}

		if strings.HasPrefix(line, "CMD EXIT") {
			message := make(map[string]interface{})
			message["id"] = self.GetArgsValue("id")
			message[kJobStatus] = kJobFailed
			log.Println("send result:", message)
			self.message <- message
			self.status = kCommandStatusIdle
			cmd.Process.Wait()
			cmd.Process.Kill()
		}
	}
	self.isFinish = true
}
